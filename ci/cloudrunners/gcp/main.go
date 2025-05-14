package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/cncf/automation/cloudrunners/gcp/pkg/gce"
	"github.com/cncf/automation/cloudrunners/pkg/remote"
	"golang.org/x/crypto/ssh"
	"google.golang.org/api/compute/v1"
	"k8s.io/klog/v2"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	log := klog.FromContext(ctx)

	buildInfo, _ := debug.ReadBuildInfo()
	log.Info("starting gha-cloudrunner-gcp", "buildInfo", buildInfo)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("getting hostname: %w", err)
	}

	instanceName := hostname
	zone := os.Getenv("RUNNER_ZONE")
	sourceImage := os.Getenv("RUNNER_IMAGE")
	machineType := os.Getenv("RUNNER_MACHINE_TYPE")
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	networkName := "default"

	computeClient, err := compute.NewService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}

	metadataClient := metadata.NewWithOptions(&metadata.Options{})

	if projectID == "" {
		s, err := metadataClient.ProjectIDWithContext(ctx)
		if err != nil {
			return fmt.Errorf("cannot determine project id: %w", err)
		}
		projectID = s
		log.Info("got project from metadata server", "project", projectID)
	}

	if zone == "" {
		s, err := metadataClient.ZoneWithContext(ctx)
		if err != nil {
			return fmt.Errorf("cannot determine GCE zone: %w", err)
		}
		zone = s
	}

	if machineType == "" {
		machineType = "n1-standard-4"
	}
	if sourceImage == "" {
		sourceImage = "projects/ubuntu-os-cloud/global/images/family/ubuntu-minimal-2404-lts-amd64"
	}

	// TODO: Should we manage a pool of instances?

	sshKeyPair, err := remote.CreateSSHKeyPair()
	if err != nil {
		return fmt.Errorf("creating ssh key pair: %w", err)
	}

	sshKeys := "gha:" + sshKeyPair.PublicKey

	startRequest := &compute.Instance{
		Name:        instanceName,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType),
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: sourceImage,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Name: "External NAT",
						Type: "ONE_TO_ONE_NAT",
					},
				},
				Network: "global/networks/" + networkName,
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				// {
				// 	Key:   "startup-script",
				// 	Value: &startupScript,
				// },
				{
					Key:   "ssh-keys",
					Value: &sshKeys,
				},
			},
		},
	}
	log.Info("creating instance", "instance", startRequest)

	instance, err := gce.NewEphemeralMachine(ctx, computeClient, projectID, zone, startRequest)
	if err != nil {
		return fmt.Errorf("creating instance: %w", err)
	}
	defer func() {
		if err := instance.Delete(context.Background()); err != nil {
			log.Error(err, "error cleaning up instance")
		}
	}()

	if err := instance.WaitForInstanceReady(ctx); err != nil {
		return fmt.Errorf("waiting for instance: %w", err)
	}
	// TODO: Use internal IP or external IP?  Internal IP might be tricky cross-project.  External IP means we need a public IP.
	ip := instance.InternalIP()
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

	if err := runGHA(ctx, ip, sshConfig); err != nil {
		return fmt.Errorf("running github action: %w", err)
	}

	return nil
}

func runGHA(ctx context.Context, ip string, sshConfig *ssh.ClientConfig) error {
	log := klog.FromContext(ctx)

	sshClient, err := remote.DialWithRetry(ctx, "tcp", ip+":22", sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to ssh on %q: %w", ip, err)
	}
	defer sshClient.Close()

	commands := []string{
		// These commands should be done by the base image, but can be done here for development
		// "sudo mkdir -p /opt/actions-runner",
		// "cd /opt/actions-runner && sudo curl -O -L https://github.com/actions/runner/releases/download/v2.323.0/actions-runner-linux-x64-2.323.0.tar.gz",
		// "cd /opt/actions-runner && sudo tar xzf ./actions-runner-linux-x64-2.323.0.tar.gz",

		"/opt/actions-runner/run.sh --jitconfig \"${ACTIONS_RUNNER_INPUT_JITCONFIG}\"",
	}

	for _, cmd := range commands {
		log.Info("running ssh command", "command", cmd)

		// Avoid logging token
		expanded := strings.ReplaceAll(cmd, "${ACTIONS_RUNNER_INPUT_JITCONFIG}", os.Getenv("ACTIONS_RUNNER_INPUT_JITCONFIG"))

		output, err := sshClient.RunCommand(ctx, expanded)
		if err != nil {
			log.Error(err, "running ssh command", "command", cmd, "output", output)
			return fmt.Errorf("running command %q: %w", cmd, err)
		}
		log.Info("command succeeded", "command", cmd, "output", string(output))
	}

	return nil
}
