package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
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
	log.Info("starting gha-imagebuilder", "buildInfo", buildInfo)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("getting hostname: %w", err)
	}

	instanceName := hostname + "-" + time.Now().UTC().Format("20060102-150405")
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	diskName := "gha-" + time.Now().UTC().Format("20060102-150405")
	zone := "us-central1-a"
	machineType := "n1-standard-4"

	baseImage := "projects/ubuntu-os-cloud/global/images/family/ubuntu-minimal-2404-lts-amd64"
	networkName := "default"

	flag.StringVar(&diskName, "disk-name", diskName, "disk image name")
	flag.StringVar(&projectID, "project", projectID, "google cloud project ID")
	flag.StringVar(&baseImage, "base-image", baseImage, "image to use as base")
	flag.StringVar(&zone, "zone", zone, "GCE zone to create VM in")
	flag.StringVar(&machineType, "machine-type", machineType, "GCE machine type to use")

	flag.Parse()

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
					SourceImage: baseImage,
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
	ip := instance.ExternalIP()
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

	if err := gce.CreateImage(ctx, ip, sshConfig); err != nil {
		return fmt.Errorf("running github action: %w", err)
	}

	if err := instance.Stop(ctx); err != nil {
		return fmt.Errorf("stopping instance: %w", err)
	}

	image, err := instance.CreateDiskImage(ctx, diskName)
	if err != nil {
		return fmt.Errorf("creating machine image: %w", err)
	}
	log.Info("created disk image", "disk.name", image.Name, "disk.selfLink", image.SelfLink)

	return nil
}
