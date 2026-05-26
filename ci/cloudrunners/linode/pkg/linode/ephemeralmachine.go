package linode

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/linode/linodego"
	"k8s.io/klog/v2"
)

// EphemeralMachine manages the lifecycle of a Linode instance that exists
// only for the duration of a GitHub Actions runner job.
type EphemeralMachine struct {
	client     linodego.Client
	instanceID int
	instance   *linodego.Instance
}

// NewEphemeralMachine creates a new Linode instance.
func NewEphemeralMachine(ctx context.Context, client linodego.Client, opts linodego.InstanceCreateOptions) (*EphemeralMachine, error) {
	log := klog.FromContext(ctx)

	log.Info("launching instance", "label", opts.Label, "region", opts.Region, "type", opts.Type)
	instance, err := client.CreateInstance(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}

	m := &EphemeralMachine{
		client:     client,
		instanceID: instance.ID,
		instance:   instance,
	}

	return m, nil
}

// WaitForInstanceReady polls until the Linode instance reaches "running" status.
func (m *EphemeralMachine) WaitForInstanceReady(ctx context.Context) error {
	log := klog.FromContext(ctx)

	for {
		instance, err := m.client.GetInstance(ctx, m.instanceID)
		if err != nil {
			return fmt.Errorf("reading instance %d: %w", m.instanceID, err)
		}

		log.Info("waiting for instance to be running", "instanceID", m.instanceID, "status", instance.Status)

		switch instance.Status {
		case linodego.InstanceRunning:
			m.instance = instance
			return nil
		case linodego.InstanceOffline:
			return fmt.Errorf("instance %d entered offline state", m.instanceID)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// Close implements io.Closer for deferred cleanup.
func (m *EphemeralMachine) Close() error {
	return m.Delete(context.Background())
}

// Delete removes the Linode instance.
func (m *EphemeralMachine) Delete(ctx context.Context) error {
	log := klog.FromContext(ctx)
	log.Info("deleting instance", "instanceID", m.instanceID)

	err := m.client.DeleteInstance(ctx, m.instanceID)
	if err != nil {
		return fmt.Errorf("deleting instance %d: %w", m.instanceID, err)
	}

	// Wait for the instance to be fully deleted.
	for {
		_, err := m.client.GetInstance(ctx, m.instanceID)
		if err != nil {
			// A 404 means the instance is gone.
			if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code == http.StatusNotFound {
				log.Info("instance deleted", "instanceID", m.instanceID)
				return nil
			}
			return fmt.Errorf("waiting for instance deletion: %w", err)
		}
		log.Info("waiting for instance to be deleted", "instanceID", m.instanceID)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// ExternalIP returns the first public IPv4 address of the instance.
func (m *EphemeralMachine) ExternalIP() string {
	if m.instance == nil {
		return ""
	}
	for _, ip := range m.instance.IPv4 {
		if ip.IsGlobalUnicast() && !ip.IsPrivate() {
			return ip.String()
		}
	}
	return ""
}

// InternalIP returns the first private IPv4 address of the instance.
func (m *EphemeralMachine) InternalIP() string {
	if m.instance == nil {
		return ""
	}
	for _, ip := range m.instance.IPv4 {
		if ip.IsPrivate() {
			return ip.String()
		}
	}
	return ""
}
