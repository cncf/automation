package main

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	linodepkg "github.com/cncf/automation/cloudrunners/linode/pkg/linode"
	"github.com/cncf/automation/cloudrunners/pkg/remote"
	"github.com/linode/linodego"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

var Cmd = &cobra.Command{
	Use:  "gha-runner",
	Long: "Run a GitHub Actions runner (on Linode/Akamai Cloud)",
	RunE: run,
}

var args struct {
	debug bool

	arch         string
	region       string
	instanceType string
	image        string
	runEnv       string

	fallbackRegion string
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	if err := Cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(0)
}

// findImage returns the latest GHA runner image available in the Linode account.
func findImage(ctx context.Context, client linodego.Client, arch, runEnv string) (*linodego.Image, error) {
	prefix := fmt.Sprintf("ubuntu-24.04-%s-gha-image", arch)
	if runEnv != "production" {
		prefix = fmt.Sprintf("rc-ubuntu-24.04-%s-gha-image", arch)
	}

	images, err := client.ListImages(ctx, linodego.NewListOptions(0, ""))
	if err != nil {
		return nil, fmt.Errorf("listing images: %w", err)
	}

	var latest *linodego.Image
	for i := range images {
		img := &images[i]
		if !strings.HasPrefix(img.Label, prefix) {
			continue
		}
		if img.Status != "available" {
			continue
		}
		if latest == nil || img.Created.After(*latest.Created) {
			latest = img
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no images found matching prefix %q", prefix)
	}
	return latest, nil
}

func run(cmd *cobra.Command, argv []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	token := os.Getenv("LINODE_TOKEN")
	if token == "" {
		return fmt.Errorf("LINODE_TOKEN environment variable is required")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	client := linodego.NewClient(oauth2Client)

	// Build the ordered list of regions.
	regions := []string{args.region}
	if args.fallbackRegion != "" {
		regions = append(regions, args.fallbackRegion)
	}

	// Create SSH key pair once — reused across retry attempts.
	sshKeyPair, err := remote.CreateSSHKeyPair()
	if err != nil {
		return fmt.Errorf("creating ssh key pair: %w", err)
	}

	var lastErr error
	for _, region := range regions {
		log.Printf("attempting launch: region=%s type=%s", region, args.instanceType)

		machine, err := tryLaunch(ctx, client, region, sshKeyPair)
		if err != nil {
			log.Printf("failed to launch in region %s: %v", region, err)
			lastErr = err
			continue
		}

		// Instance created — make sure it gets cleaned up on
		// normal exit *and* on SIGTERM / SIGINT (pod termination).
		cleanup := func() {
			log.Println("cleaning up: delete machine", machine.ExternalIP())
			if err := machine.Delete(context.Background()); err != nil {
				log.Printf("failed to delete machine: %v", err)
			}
		}
		defer cleanup()

		go func() {
			<-ctx.Done()
			if ctx.Err() == context.Canceled {
				log.Println("received shutdown signal, deleting machine")
				cleanup()
			}
		}()

		log.Printf("instance launched successfully: region=%s", region)
		return runOnMachine(ctx, machine, sshKeyPair)
	}

	return fmt.Errorf("all regions exhausted: %w", lastErr)
}

// tryLaunch finds the latest image and attempts to launch an instance in the given region.
func tryLaunch(ctx context.Context, client linodego.Client, region string, sshKeyPair *remote.SSHKeyPair) (*linodepkg.EphemeralMachine, error) {
	imageID := args.image
	if imageID == "" {
		// Default to the Linode-provided Ubuntu 24.04 image.
		imageID = "linode/ubuntu24.04"
	}
	log.Printf("using image: %s", imageID)

	name := fmt.Sprintf("gha-runner-%s-%s", args.arch, time.Now().Format("20060102-150405"))

	// Linode requires a root password when using public images.
	// Generate a random one — SSH key auth is used for actual access.
	rootPass, err := generateRandomPassword(32)
	if err != nil {
		return nil, fmt.Errorf("generating root password: %w", err)
	}

	opts := linodego.InstanceCreateOptions{
		Region:         region,
		Type:           args.instanceType,
		Label:          name,
		Image:          imageID,
		RootPass:       rootPass,
		AuthorizedKeys: []string{strings.TrimSpace(sshKeyPair.PublicKey)},
		Booted:         linodego.Pointer(true),
	}

	return linodepkg.NewEphemeralMachine(ctx, client, opts)
}

// runOnMachine waits for the instance to be ready, connects via SSH and
// executes the GitHub Actions runner.
func runOnMachine(ctx context.Context, machine *linodepkg.EphemeralMachine, sshKeyPair *remote.SSHKeyPair) error {
	// Sleep before checking if the instance is ready.
	time.Sleep(15 * time.Second)

	if err := machine.WaitForInstanceReady(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance to be ready: %w", err)
	}

	ip := machine.ExternalIP()
	if ip == "" {
		return fmt.Errorf("cannot find ip for instance")
	}

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			sshKeyPair.SSHAuth,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	sshClient, err := remote.DialWithRetry(ctx, "tcp", ip+":22", sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to ssh on %q: %w", ip, err)
	}
	defer sshClient.Close()

	commands := []string{
		"tar -zxf /opt/runner-cache/actions-runner-linux-*.tar.gz",
		"rm -rf \\$HOME",
		"sudo chown -R 1000:1000 /etc/skel/",
		"mv /etc/skel/.cargo /home/ubuntu/",
		"mv /etc/skel/.nvm /home/ubuntu/",
		"mv /etc/skel/.rustup /home/ubuntu/",
		"mv /etc/skel/.dotnet /home/ubuntu/",
		"mv /etc/skel/.composer /home/ubuntu/",
		"sudo setfacl -m u:ubuntu:rw /var/run/docker.sock",
		"sudo sysctl fs.inotify.max_user_instances=1280",
		"sudo sysctl fs.inotify.max_user_watches=655360",
		"export PATH=$PATH:/home/ubuntu/.local/bin && export HOME=/home/ubuntu && export NVM_DIR=/home/ubuntu/.nvm && bash -x /home/ubuntu/run.sh --jitconfig \"${ACTIONS_RUNNER_INPUT_JITCONFIG}\"",
	}

	for _, cmd := range commands {
		log.Println("running ssh command", "command", cmd)

		// Avoid logging token
		expanded := strings.ReplaceAll(cmd, "${ACTIONS_RUNNER_INPUT_JITCONFIG}", os.Getenv("ACTIONS_RUNNER_INPUT_JITCONFIG"))

		output, err := sshClient.RunCommand(ctx, expanded)
		if err != nil {
			log.Println(err, "running ssh command", "command", cmd, "output", string(output[:]))
			return fmt.Errorf("running command %q: %w", cmd, err)
		}
		log.Println("command succeeded", "command", cmd, "output", string(output))
	}

	return nil
}

func init() {
	flags := Cmd.Flags()

	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug logging",
	)
	flags.StringVar(
		&args.arch,
		"arch",
		"x86",
		"Machine architecture",
	)
	flags.StringVar(
		&args.region,
		"region",
		"us-ord",
		"Linode region (e.g. us-ord, us-iad, eu-de)",
	)
	flags.StringVar(
		&args.instanceType,
		"instance-type",
		"g6-dedicated-8",
		"Linode instance type (e.g. g6-dedicated-8, g6-standard-6)",
	)
	flags.StringVar(
		&args.image,
		"image",
		"",
		"Linode image ID to use (overrides automatic image discovery)",
	)
	flags.StringVar(
		&args.runEnv,
		"running-environment",
		"production",
		"Running Environment: production or ci",
	)
	flags.StringVar(
		&args.fallbackRegion,
		"fallback-region",
		"",
		"Fallback Linode region to try when primary fails",
	)
}

// generateRandomPassword returns a URL-safe base64-encoded random string.
func generateRandomPassword(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := cryptorand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
