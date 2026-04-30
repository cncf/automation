package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cncf/automation/cloudrunners/oci/pkg/oci"
	"github.com/cncf/automation/cloudrunners/pkg/remote"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// regionConfig holds the per-region parameters needed to launch an instance.
type regionConfig struct {
	Region             string
	AvailabilityDomain string
	SubnetID           string
}

var Cmd = &cobra.Command{
	Use:  "gha-runner",
	Long: "Run a GitHub Actions runner (on Oracle Cloud Infrastructure)",
	RunE: run,
}
var args struct {
	debug bool

	arch               string
	region             string
	compartmentId      string
	subnetId           string
	availabilityDomain string
	shape              string
	shapeOcpus         float32
	shapeMemoryInGBs   float32
	runEnv             string

	fallbackRegion             string
	fallbackAvailabilityDomain string
	fallbackSubnetId           string
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	if err := Cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(0)
}

// isOutOfCapacityError returns true when the OCI error indicates the host capacity in the target region/AD has been exhausted.
func isOutOfCapacityError(err error) bool {
	var svcErr common.ServiceError
	if errors.As(err, &svcErr) {
		code := svcErr.GetCode()
		msg := strings.ToLower(svcErr.GetMessage())
		if strings.Contains(msg, "out of host capacity") ||
			strings.Contains(msg, "out of capacity") ||
			code == "LimitExceeded" ||
			code == "InternalError" && strings.Contains(msg, "capacity") {
			return true
		}
	}
	return false
}

// findImage returns the latest GHA runner image available in the current region of the given compute client.
func findImage(ctx context.Context, computeClient core.ComputeClient, compartmentId, arch, runEnv string) (*core.Image, error) {
	osname := fmt.Sprintf("ubuntu-24.04-%s-gha-image", arch)
	if runEnv != "production" {
		osname = fmt.Sprintf("rc-ubuntu-24.04-%s-gha-image", arch)
	}
	images, err := computeClient.ListImages(ctx, core.ListImagesRequest{
		CompartmentId:   common.String(compartmentId),
		OperatingSystem: common.String(osname),
		SortBy:          core.ListImagesSortByTimecreated,
		SortOrder:       core.ListImagesSortOrderDesc,
		Limit:           common.Int(1),
		LifecycleState:  core.ImageLifecycleStateAvailable,
	})
	if err != nil {
		return nil, fmt.Errorf("listing images: %w", err)
	}
	if len(images.Items) == 0 {
		return nil, fmt.Errorf("no images found for %s", osname)
	}
	return &images.Items[0], nil
}

func run(cmd *cobra.Command, argv []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// Parse the comma-separated shape list (single value is fine too).
	shapes := strings.Split(args.shape, ",")
	for i := range shapes {
		shapes[i] = strings.TrimSpace(shapes[i])
	}

	// Build the ordered list of regions: primary (from flags) + fallbacks.
	regions := []regionConfig{
		{
			Region:             args.region,
			AvailabilityDomain: args.availabilityDomain,
			SubnetID:           args.subnetId,
		},
	}
	if args.fallbackRegion != "" {
		if args.fallbackAvailabilityDomain == "" || args.fallbackSubnetId == "" {
			return fmt.Errorf("--fallback-availability-domain and --fallback-subnet-id are required when --fallback-region is set")
		}
		regions = append(regions, regionConfig{
			Region:             args.fallbackRegion,
			AvailabilityDomain: args.fallbackAvailabilityDomain,
			SubnetID:           args.fallbackSubnetId,
		})
	}

	// Create SSH key pair once — reused across retry attempts.
	sshKeyPair, err := remote.CreateSSHKeyPair()
	if err != nil {
		return fmt.Errorf("creating ssh key pair: %w", err)
	}

	var lastErr error
	// Try each shape across all regions before falling back to the next shape,
	// so we prefer the higher-performance shape (e.g. E6 in any region before E5).
	for _, shape := range shapes {
		for _, region := range regions {
			log.Printf("attempting launch: region=%s shape=%s", region.Region, shape)

			machine, err := tryLaunch(ctx, region, shape, sshKeyPair)
			if err != nil {
				if isOutOfCapacityError(err) {
					log.Printf("out of capacity: region=%s shape=%s: %v", region.Region, shape, err)
					lastErr = err
					continue
				}
				return fmt.Errorf("failed to create machine (region=%s, shape=%s): %w", region.Region, shape, err)
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

			// If the context was cancelled by a signal, clean up immediately.
			go func() {
				<-ctx.Done()
				if ctx.Err() == context.Canceled {
					log.Println("received shutdown signal, deleting machine")
					cleanup()
				}
			}()

			log.Printf("instance launched successfully: region=%s shape=%s", region.Region, shape)
			return runOnMachine(ctx, machine, sshKeyPair)
		}
	}

	return fmt.Errorf("all regions and shapes exhausted: %w", lastErr)
}

// tryLaunch creates OCI clients for the given region, finds the latest image
// and attempts to launch an instance with the specified shape.
func tryLaunch(ctx context.Context, region regionConfig, shape string, sshKeyPair *remote.SSHKeyPair) (*oci.EphemeralMachine, error) {
	configProvider := common.DefaultConfigProvider()

	computeClient, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}
	networkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create network client: %w", err)
	}

	computeClient.SetRegion(region.Region)
	networkClient.SetRegion(region.Region)

	latestImage, err := findImage(ctx, computeClient, args.compartmentId, args.arch, args.runEnv)
	if err != nil {
		return nil, err
	}

	launchDetails := core.LaunchInstanceDetails{
		AvailabilityDomain: common.String(region.AvailabilityDomain),
		CompartmentId:      common.String(args.compartmentId),
		Shape:              common.String(shape),
		DisplayName:        common.String(fmt.Sprintf("gha-runner-%s-%s", args.arch, time.Now().Format("20060102-150405"))),
		CreateVnicDetails: &core.CreateVnicDetails{
			SubnetId:       common.String(region.SubnetID),
			AssignPublicIp: common.Bool(true),
		},
		Metadata: map[string]string{
			"ssh_authorized_keys": sshKeyPair.PublicKey,
		},
		SourceDetails: &core.InstanceSourceViaImageDetails{
			ImageId:             common.String(*latestImage.Id),
			BootVolumeSizeInGBs: common.Int64(600),
			BootVolumeVpusPerGB: common.Int64(120),
		},
		AgentConfig: &core.LaunchInstanceAgentConfigDetails{
			PluginsConfig: []core.InstanceAgentPluginConfigDetails{{
				DesiredState: core.InstanceAgentPluginConfigDetailsDesiredStateEnabled,
				Name:         common.String("Compute Instance Monitoring"),
			}},
			AreAllPluginsDisabled: common.Bool(false),
			IsMonitoringDisabled:  common.Bool(false),
		},
	}

	// Only set flexible shape config when OCPUs/memory are specified and
	// the shape is actually flexible.
	if args.shapeMemoryInGBs > 0.0 && args.shapeOcpus > 0.0 && strings.Contains(shape, "Flex") {
		launchDetails.ShapeConfig = &core.LaunchInstanceShapeConfigDetails{
			MemoryInGBs: common.Float32(args.shapeMemoryInGBs),
			Ocpus:       common.Float32(args.shapeOcpus),
		}
	}

	return oci.NewEphemeralMachine(ctx, computeClient, networkClient, launchDetails)
}

// runOnMachine waits for the instance to be ready, connects via SSH and
// executes the GitHub Actions runner.
func runOnMachine(ctx context.Context, machine *oci.EphemeralMachine, sshKeyPair *remote.SSHKeyPair) error {
	// Sleep before checking if the instance is ready
	time.Sleep(30 * time.Second)

	// Wait for the machine to be ready
	err := machine.WaitForInstanceReady(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for instance to be ready: %w", err)
	}

	ip := machine.ExternalIP()
	if ip == "" {
		return fmt.Errorf("cannot find ip for instance")
	}

	sshConfig := &ssh.ClientConfig{
		User: "ubuntu",
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
		"sudo sh -c 'echo \"install algif_aead /bin/false\" > /etc/modprobe.d/disable-algif.conf'",
		"sudo rmmod algif_aead 2>/dev/null || true",
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
		"us-sanjose-1",
		"OCI region",
	)
	flags.StringVar(
		&args.availabilityDomain,
		"availability-domain",
		"bzBe:US-SANJOSE-1-AD-1",
		"Availability Domain",
	)
	flags.StringVar(
		&args.compartmentId,
		"compartment-id",
		"ocid1.compartment.oc1..aaaaaaaa22icap66vxktktubjlhf6oxvfhev6n7udgje2chahyrtq65ga63a",
		"Compartment ID",
	)
	flags.StringVar(
		&args.subnetId,
		"subnet-id",
		"ocid1.subnet.oc1.us-sanjose-1.aaaaaaaahgdslvujnywu3hvhqbvgz23souseseozvypng7ehnxgcotislubq",
		"Subnet ID",
	)
	flags.StringVar(
		&args.shape,
		"shape",
		"VM.Standard.E6.Flex,VM.Standard.E5.Flex,VM.Standard.E4.Flex",
		"Comma-separated list of VM Shapes to try in order of preference (failover on capacity errors)",
	)
	flags.Float32Var(
		&args.shapeOcpus,
		"shape-ocpus",
		0.0, // Default to 0, indicating not set.
		"Number of OCPUs for flexible shapes (e.g., 1.0, 2.0). Required if a '.Flex' shape is used.",
	)
	flags.Float32Var(
		&args.shapeMemoryInGBs,
		"shape-memory-in-gbs",
		0.0, // Default to 0.
		"Amount of memory in GBs for flexible shapes (e.g., 16.0, 32.0). Required if a '.Flex' shape is used.",
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
		"us-ashburn-1",
		"Fallback OCI region to try when primary is out of capacity",
	)
	flags.StringVar(
		&args.fallbackAvailabilityDomain,
		"fallback-availability-domain",
		"bzBe:US-ASHBURN-AD-1",
		"Availability domain for the fallback region",
	)
	flags.StringVar(
		&args.fallbackSubnetId,
		"fallback-subnet-id",
		"ocid1.subnet.oc1.iad.aaaaaaaagygdzd4xgbz4xhqhvnbxnoemhjd5ick7vodx4ghk4kg6a6c4xh5q",
		"Subnet ID for the fallback region",
	)
}
