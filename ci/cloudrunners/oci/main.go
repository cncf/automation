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
		CompartmentId:   common.String("ocid1.compartment.oc1..aaaaaaaa22icap66vxktktubjlhf6oxvfhev6n7udgje2chahyrtq65ga63a"),
		OperatingSystem: common.String("ubuntu-24.04-x86-gha-image"),
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

	sshKeys := "gha:" + sshKeyPair.PublicKey

	// Create a new ephemeral machine
	machine, err := oci.NewEphemeralMachine(ctx, computeClient, networkClient, core.LaunchInstanceDetails{
		AvailabilityDomain: common.String(args.availabilityDomain),
		CompartmentId:      common.String(args.compartmentId),
		ImageId:            common.String(*latestImage.Id),
		CreateVnicDetails: &core.CreateVnicDetails{
			SubnetId:       common.String(args.subnetId),
			AssignPublicIp: common.Bool(true),
		},
		Metadata: map[string]string{
			"ssh_authorized_keys": sshKeys,
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
		User: "gha",
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
		"/home/runner/run.sh --jitconfig \"${ACTIONS_RUNNER_INPUT_JITCONFIG}\"",
	}

	for _, cmd := range commands {
		log.Println("running ssh command", "command", cmd)

		// Avoid logging token
		expanded := strings.ReplaceAll(cmd, "${ACTIONS_RUNNER_INPUT_JITCONFIG}", os.Getenv("ACTIONS_RUNNER_INPUT_JITCONFIG"))

		output, err := sshClient.RunCommand(ctx, expanded)
		if err != nil {
			log.Println(err, "running ssh command", "command", cmd, "output", output)
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
		"SJC-AD-1",
		"Availability Domain",
	)
	flags.StringVar(
		&args.compartmentId,
		"compartment-id",
		"ocid1.compartment.oc1..aaaaaaaa22icap66vxktktubjlhf6oxvfhev6n7udgje2chahyrtq65ga63a",
		"Compartment ID",
	)
	flags.StringVar(
		&args.compartmentId,
		"subnet-id",
		"ocid1.subnet.oc1.us-sanjose-1.aaaaaaaahgdslvujnywu3hvhqbvgz23souseseozvypng7ehnxgcotislubq",
		"Subnet ID",
	)

}
