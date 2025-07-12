package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cncf/automation/cloudrunners/oci/pkg/oci"
	"github.com/cncf/automation/cloudrunners/pkg/remote"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var Cmd = &cobra.Command{
	Use:  "gha-runner",
	Long: "Run a GitHub Actions runner (on Oracle Cloud Infrastructure)",
	RunE: run,
}
var args struct {
	debug bool

	arch               string
	compartmentId      string
	subnetId           string
	availabilityDomain string
	shape              string
	shapeOcpus         float32
	shapeMemoryInGBs   float32
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	if err := Cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()
	// Initialize the OCI client
	computeClient, err := core.NewComputeClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}
	networkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return fmt.Errorf("failed to create network client: %w", err)
	}

	// List Images and retrieve the latest ID by type and arch

	images, err := computeClient.ListImages(ctx, core.ListImagesRequest{
		CompartmentId:   common.String(args.compartmentId),
		OperatingSystem: common.String(fmt.Sprintf("ubuntu-24.04-%s-gha-image", args.arch)),
		SortBy:          core.ListImagesSortByTimecreated,
		SortOrder:       core.ListImagesSortOrderDesc,
		Limit:           common.Int(1),
	})
	if err != nil {
		panic(err)
	}
	if len(images.Items) == 0 {
		return fmt.Errorf("no images found")
	}
	latestImage := images.Items[0]

	// Create SSH Key Pair
	sshKeyPair, err := remote.CreateSSHKeyPair()
	if err != nil {
		return fmt.Errorf("creating ssh key pair: %w", err)
	}

	// Create a new ephemeral machine
	machine, err := oci.NewEphemeralMachine(ctx, computeClient, networkClient, core.LaunchInstanceDetails{
		AvailabilityDomain: common.String(args.availabilityDomain),
		CompartmentId:      common.String(args.compartmentId),
		Shape:              common.String(args.shape),
		DisplayName:        common.String(fmt.Sprintf("gha-runner-%s-%s", args.arch, time.Now().Format("20060102-150405"))),
		CreateVnicDetails: &core.CreateVnicDetails{
			SubnetId:       common.String(args.subnetId),
			AssignPublicIp: common.Bool(true),
		},
		ShapeConfig: &core.LaunchInstanceShapeConfigDetails{
			MemoryInGBs: common.Float32(args.shapeMemoryInGBs),
			Ocpus:       common.Float32(args.shapeOcpus),
		},
		Metadata: map[string]string{
			"ssh_authorized_keys": sshKeyPair.PublicKey,
		},
		SourceDetails: &core.InstanceSourceViaImageDetails{
			ImageId:             common.String(*latestImage.Id),
			BootVolumeSizeInGBs: common.Int64(600),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create machine: %w", err)
	}

	defer func() {
		err := machine.Delete(context.Background())
		if err != nil {
			log.Printf("failed to delete machine: %v", err)
		}
	}()

	// Wait for the machine to be ready
	err = machine.WaitForInstanceReady(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for instance to be ready: %w", err)
	}

	// TODO: Use internal IP or external IP?  Internal IP might be tricky cross-project.  External IP means we need a public IP.
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
		"tar -zxf /opt/runner-cache/actions-runner-linux-*.tar.gz",
		"rm -rf \\$HOME",
		"sudo chown -R 1000:1000 /etc/skel/",
		"mv /etc/skel/.cargo /home/ubuntu/",
		"mv /etc/skel/.nvm /home/ubuntu/",
		"mv /etc/skel/.rustup /home/ubuntu/",
		"mv /etc/skel/.dotnet /home/ubuntu/",
		"mv /etc/skel/.composer /home/ubuntu/",
		`sudo usermod -aG docker ubuntu && newgrp docker <<EOF
export PATH=$PATH:/home/ubuntu/.local/bin && export HOME=/home/ubuntu && export NVM_DIR=/home/ubuntu/.nvm && bash -x /home/ubuntu/run.sh --jitconfig "${ACTIONS_RUNNER_INPUT_JITCONFIG}"
EOF`,
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
		"VM.Standard.E2.2",
		"VM Shape",
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
}
