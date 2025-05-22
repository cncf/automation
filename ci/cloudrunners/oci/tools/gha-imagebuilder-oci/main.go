package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/cncf/automation/cloudrunners/oci/pkg/oci"
	"github.com/cncf/automation/cloudrunners/pkg/ghaimage"
	"github.com/cncf/automation/cloudrunners/pkg/remote"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"golang.org/x/crypto/ssh"
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

	displayName := hostname + "-" + time.Now().UTC().Format("20060102-150405")
	compartmentID := os.Getenv("OCI_COMPARTMENT_ID")
	createImageName := "gha-" + time.Now().UTC().Format("20060102-150405")
	availabilityDomain := "1" // Default to first AD
	shape := "VM.Standard.E5.Flex"
	ocpus := 1
	memoryInGBs := 16
	sshUserName := "ubuntu"

	baseImage := "ocid1.image.oc1..aaaaaaaaxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Replace with actual Ubuntu image OCID
	subnetID := os.Getenv("OCI_SUBNET_ID")

	flag.StringVar(&createImageName, "create-image-name", createImageName, "image name to create")
	flag.StringVar(&compartmentID, "compartment", compartmentID, "OCI compartment ID")
	flag.StringVar(&baseImage, "base-image", baseImage, "image to use as base")
	flag.StringVar(&availabilityDomain, "availability-domain", availabilityDomain, "OCI availability domain")
	flag.StringVar(&shape, "shape", shape, "OCI instance shape")
	flag.IntVar(&ocpus, "ocpus", ocpus, "Number of OCPUs")
	flag.IntVar(&memoryInGBs, "memory", memoryInGBs, "Memory in GBs")
	flag.StringVar(&subnetID, "subnet", subnetID, "OCI subnet ID")

	flag.Parse()

	if subnetID == "" {
		return fmt.Errorf("--subnet is required")
	}

	// Create OCI clients
	configProvider := common.DefaultConfigProvider()
	computeClient, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}

	networkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
	if err != nil {
		return fmt.Errorf("failed to create network client: %w", err)
	}

	if compartmentID == "" {
		// Get the tenancy OCID from the config provider
		tenancyID, err := configProvider.TenancyOCID()
		if err != nil {
			return fmt.Errorf("cannot determine tenancy id: %w", err)
		}
		compartmentID = tenancyID
		log.Info("using tenancy as compartment", "compartment", compartmentID)
	}

	sshKeyPair, err := remote.CreateSSHKeyPair()
	if err != nil {
		return fmt.Errorf("creating ssh key pair: %w", err)
	}

	// Create instance
	instanceDetails := core.LaunchInstanceDetails{
		DisplayName:        common.String(displayName),
		CompartmentId:      common.String(compartmentID),
		AvailabilityDomain: common.String(availabilityDomain),
		Shape:              common.String(shape),
		ShapeConfig: &core.LaunchInstanceShapeConfigDetails{
			Ocpus:       common.Float32(float32(ocpus)),
			MemoryInGBs: common.Float32(float32(memoryInGBs)),
		},
		SourceDetails: core.InstanceSourceViaImageDetails{
			ImageId: common.String(baseImage),
		},
		CreateVnicDetails: &core.CreateVnicDetails{
			SubnetId: common.String(subnetID),
		},
		Metadata: map[string]string{
			"ssh_authorized_keys": sshKeyPair.PublicKey,
		},
	}

	log.Info("creating instance", "instance", instanceDetails)

	instance, err := oci.NewEphemeralMachine(ctx, computeClient, networkClient, instanceDetails)
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
		User: sshUserName,
		Auth: []ssh.AuthMethod{
			sshKeyPair.SSHAuth,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	if err := ghaimage.InstallComponents(ctx, ip, sshConfig); err != nil {
		return fmt.Errorf("running github action: %w", err)
	}

	if err := instance.Stop(ctx); err != nil {
		return fmt.Errorf("stopping instance: %w", err)
	}

	image, err := instance.CreateDiskImage(ctx, createImageName)
	if err != nil {
		return fmt.Errorf("creating machine image: %w", err)
	}
	log.Info("created disk image", "disk.name", ValueOf(image.Id))

	return nil
}

func ValueOf[T any](p *T) T {
	if p == nil {
		var t T
		return t
	}
	return *p
}
