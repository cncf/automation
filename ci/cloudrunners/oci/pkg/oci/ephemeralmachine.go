package oci

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/core"
	"k8s.io/klog/v2"
)

type EphemeralMachine struct {
	computeClient      core.ComputeClient
	networkClient      core.VirtualNetworkClient
	compartmentID      string
	availabilityDomain string
	instanceID         string

	instance *core.Instance
	vnics    []core.Vnic
}

func NewEphemeralMachine(ctx context.Context, computeClient core.ComputeClient, networkClient core.VirtualNetworkClient, config core.LaunchInstanceDetails) (*EphemeralMachine, error) {
	log := klog.FromContext(ctx)

	m := &EphemeralMachine{
		computeClient:      computeClient,
		networkClient:      networkClient,
		compartmentID:      ValueOf(config.CompartmentId),
		availabilityDomain: ValueOf(config.AvailabilityDomain),
	}

	// Launch the instance
	launchInstanceRequest := core.LaunchInstanceRequest{
		LaunchInstanceDetails: config,
	}
	log.Info("launching instance", "request", launchInstanceRequest)
	launchInstanceResponse, err := computeClient.LaunchInstance(ctx, launchInstanceRequest)
	if err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}

	m.instanceID = ValueOf(launchInstanceResponse.Id)

	return m, nil
}

func (m *EphemeralMachine) WaitForInstanceReady(ctx context.Context) error {
	log := klog.FromContext(ctx)

	// Wait for the instance to be running
	for {
		getInstanceRequest := core.GetInstanceRequest{
			InstanceId: &m.instanceID,
		}
		getInstanceResponse, err := m.computeClient.GetInstance(ctx, getInstanceRequest)
		if err != nil {
			if err := m.Delete(context.Background()); err != nil {
				log.Error(err, "deleting machine that failed to create")
			}
			return fmt.Errorf("reading created instance: %w", err)
		}
		log.Info("waiting for instance to be running", "instanceID", m.instanceID, "instance.lifecycleState", getInstanceResponse.Instance.LifecycleState)
		if getInstanceResponse.Instance.LifecycleState == core.InstanceLifecycleStateRunning {
			m.instance = &getInstanceResponse.Instance
			break
		}
		time.Sleep(2 * time.Second)
	}

	// List VNIC attachments for the instance
	listVnicAttachmentsRequest := core.ListVnicAttachmentsRequest{
		CompartmentId: &m.compartmentID,
		InstanceId:    &m.instanceID,
	}
	listVnicAttachmentsResponse, err := m.computeClient.ListVnicAttachments(context.Background(), listVnicAttachmentsRequest)
	if err != nil {
		return fmt.Errorf("listing vnic attachments: %w", err)
	}
	// Get each VNIC and check for private IP
	for _, attachment := range listVnicAttachmentsResponse.Items {
		if attachment.VnicId == nil {
			continue
		}
		getVnicRequest := core.GetVnicRequest{
			VnicId: attachment.VnicId,
		}
		getVnicResponse, err := m.networkClient.GetVnic(context.Background(), getVnicRequest)
		if err != nil {
			continue
		}
		m.vnics = append(m.vnics, getVnicResponse.Vnic)
	}

	return nil
}

func (m *EphemeralMachine) Close() error {
	return m.Delete(context.Background())
}

func (m *EphemeralMachine) Delete(ctx context.Context) error {
	log := klog.FromContext(ctx)
	terminateInstanceRequest := core.TerminateInstanceRequest{
		InstanceId: &m.instanceID,
	}
	_, err := m.computeClient.TerminateInstance(ctx, terminateInstanceRequest)
	if err != nil {
		return fmt.Errorf("deleting instance: %w", err)
	}

	// Wait for the instance to be terminated
	for {
		getInstanceRequest := core.GetInstanceRequest{
			InstanceId: &m.instanceID,
		}
		getInstanceResponse, err := m.computeClient.GetInstance(ctx, getInstanceRequest)
		if err != nil {
			// if serviceError, ok := common.IsServiceError(err); ok {
			// 	if serviceError.GetHTTPStatusCode() == 404 {
			// 		break
			// 	}
			// }
			return fmt.Errorf("waiting for instance deletion: %w", err)
		}
		log.Info("waiting for instance to be terminated", "instanceID", m.instanceID, "instance.lifecycleState", getInstanceResponse.Instance.LifecycleState)
		if getInstanceResponse.Instance.LifecycleState == core.InstanceLifecycleStateTerminated {
			break
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func (m *EphemeralMachine) Stop(ctx context.Context) error {
	log := klog.FromContext(ctx)

	stopInstanceRequest := core.InstanceActionRequest{
		InstanceId: &m.instanceID,
		Action:     core.InstanceActionActionStop,
	}
	_, err := m.computeClient.InstanceAction(ctx, stopInstanceRequest)
	if err != nil {
		return fmt.Errorf("stopping instance: %w", err)
	}

	// Wait for the instance to be stopped
	for {
		getInstanceRequest := core.GetInstanceRequest{
			InstanceId: &m.instanceID,
		}
		instance, err := m.computeClient.GetInstance(ctx, getInstanceRequest)
		if err != nil {
			return fmt.Errorf("waiting for instance stop: %w", err)
		}
		log.Info("waiting for instance stop", "instanceID", m.instanceID, "instance.lifecycleState", instance.LifecycleState)
		if instance.LifecycleState == core.InstanceLifecycleStateStopped {
			break
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func (m *EphemeralMachine) ExternalIP() string {
	for _, vnic := range m.vnics {
		if ip := ValueOf(vnic.PublicIp); ip != "" {
			return ip
		}
	}
	return ""
}

func (m *EphemeralMachine) InternalIP() string {
	for _, vnic := range m.vnics {
		if ip := ValueOf(vnic.PrivateIp); ip != "" {
			return ip
		}
	}
	return ""
}

func (m *EphemeralMachine) CreateDiskImage(ctx context.Context, displayName string) (*core.Image, error) {
	log := klog.FromContext(ctx)

	// if m.instance == nil || len(m.instance.BootVolumeId) == 0 {
	if m.instance == nil {
		return nil, fmt.Errorf("instance or boot volume not found")
	}

	createImageRequest := core.CreateImageRequest{
		CreateImageDetails: core.CreateImageDetails{
			CompartmentId: &m.compartmentID,
			DisplayName:   &displayName,
			InstanceId:    &m.instanceID,
		},
	}

	image, err := m.computeClient.CreateImage(ctx, createImageRequest)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}

	// Wait for the image to be created (before terminating the machine)
	for {
		getImageRequest := core.GetImageRequest{
			ImageId: image.Id,
		}
		image, err := m.computeClient.GetImage(ctx, getImageRequest)
		if err != nil {
			return nil, fmt.Errorf("waiting for image creation: %w", err)
		}
		log.Info("waiting for image creation", "imageID", image.Id, "image.lifecycleState", image.LifecycleState)
		if image.LifecycleState == core.ImageLifecycleStateAvailable {
			break
		}
		time.Sleep(2 * time.Second)
	}
	log.Info("created image", "image.name", ValueOf(image.DisplayName))

	return &image.Image, nil
}

func ValueOf[T any](p *T) T {
	if p == nil {
		var t T
		return t
	}
	return *p
}
