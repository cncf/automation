package gce

import (
	"context"
	"fmt"

	"google.golang.org/api/compute/v1"
	"k8s.io/klog/v2"
)

type EphemeralMachine struct {
	computeClient *compute.Service
	projectID     string
	zone          string
	instanceName  string

	instance *compute.Instance
}

func NewEphemeralMachine(ctx context.Context, computeClient *compute.Service, projectID, zone string, config *compute.Instance) (*EphemeralMachine, error) {
	log := klog.FromContext(ctx)

	m := &EphemeralMachine{
		computeClient: computeClient,
		projectID:     projectID,
		zone:          zone,
		instanceName:  config.Name,
	}
	createOp, err := computeClient.Instances.Insert(projectID, zone, config).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}
	if err := WaitForZonalOperation(ctx, computeClient, projectID, zone, createOp.Name); err != nil {
		if err := m.Delete(context.Background()); err != nil {
			log.Error(err, "deleting machine that failed to create")
		}
		return nil, fmt.Errorf("waiting for instance create: %w", err)
	}

	instance, err := computeClient.Instances.Get(projectID, zone, m.instanceName).Context(ctx).Do()
	if err != nil {
		if err := m.Delete(context.Background()); err != nil {
			log.Error(err, "deleting machine that failed to create")
		}
		return nil, fmt.Errorf("reading created instance: %w", err)
	}

	m.instance = instance

	return m, nil
}

func (m *EphemeralMachine) WaitForInstanceReady(ctx context.Context) error {
	// TODO: move the wait here?
	return nil
}

func (m *EphemeralMachine) Close() error {
	return m.Delete(context.Background())
}

func (m *EphemeralMachine) Delete(ctx context.Context) error {
	deleteOp, err := m.computeClient.Instances.Delete(m.projectID, m.zone, m.instanceName).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("deleting instance: %w", err)
	}

	if err := WaitForZonalOperation(ctx, m.computeClient, m.projectID, m.zone, deleteOp.Name); err != nil {
		return fmt.Errorf("waiting for instance deletion: %w", err)
	}
	return nil
}

func (m *EphemeralMachine) Stop(ctx context.Context) error {
	deleteOp, err := m.computeClient.Instances.Stop(m.projectID, m.zone, m.instanceName).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("stopping instance: %w", err)
	}

	if err := WaitForZonalOperation(ctx, m.computeClient, m.projectID, m.zone, deleteOp.Name); err != nil {
		return fmt.Errorf("waiting for instance stop: %w", err)
	}
	return nil
}

func (m *EphemeralMachine) ExternalIP() string {
	for _, nic := range m.instance.NetworkInterfaces {
		for _, config := range nic.AccessConfigs {
			if config.NatIP != "" {
				return config.NatIP
			}
		}
	}
	return ""
}

func (m *EphemeralMachine) InternalIP() string {
	for _, nic := range m.instance.NetworkInterfaces {
		if nic.NetworkIP != "" {
			return nic.NetworkIP
		}
	}
	return ""
}

// CreateMachineImage creates a full snapshot of the machine
// func (m *EphemeralMachine) CreateMachineImage(ctx context.Context, diskName string) (*compute.MachineImage, error) {
// 	log := klog.FromContext(ctx)

// 	req := compute.MachineImage{
// 		SourceInstance: m.instance.SelfLink,
// 		Name:           diskName,
// 	}
// 	op, err := m.computeClient.MachineImages.Insert(m.projectID, &req).Context(ctx).Do()
// 	if err != nil {
// 		return nil, fmt.Errorf("creating machine image: %w", err)
// 	}
// 	if err := WaitForGlobalOperation(ctx, m.computeClient, m.projectID, op.Name); err != nil {
// 		return nil, fmt.Errorf("waiting for machine image create: %w", err)
// 	}

// 	disk, err := m.computeClient.MachineImages.Get(m.projectID, diskName).Context(ctx).Do()
// 	if err != nil {
// 		return nil, fmt.Errorf("reading created disk: %w", err)
// 	}
// 	log.Info("created disk", "disk", disk)

// 	return disk, nil
// }

func (m *EphemeralMachine) CreateDiskImage(ctx context.Context, imageName string) (*compute.Image, error) {
	log := klog.FromContext(ctx)

	req := compute.Image{
		SourceDisk: m.instance.Disks[0].Source,
		Name:       imageName,
	}
	op, err := m.computeClient.Images.Insert(m.projectID, &req).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}
	if err := WaitForGlobalOperation(ctx, m.computeClient, m.projectID, op.Name); err != nil {
		return nil, fmt.Errorf("waiting for image creation: %w", err)
	}

	image, err := m.computeClient.Images.Get(m.projectID, imageName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("reading created image: %w", err)
	}
	log.Info("created image", "image.name", image.Name)

	return image, nil
}
