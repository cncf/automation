package kubevirt

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var vmGVR = schema.GroupVersionResource{
	Group:    "kubevirt.io",
	Version:  "v1",
	Resource: "virtualmachines",
}

var vmiGVR = schema.GroupVersionResource{
	Group:    "kubevirt.io",
	Version:  "v1",
	Resource: "virtualmachineinstances",
}

// EphemeralMachine manages the lifecycle of a KubeVirt VirtualMachine
// (with embedded dataVolumeTemplates) that exists only for the duration
// of a GitHub Actions runner job.
type EphemeralMachine struct {
	dynamic   dynamic.Interface
	k8s       kubernetes.Interface
	namespace string
	name      string
	podIP     string
}

// NewEphemeralMachine creates a VirtualMachine in the cluster.
// The VM spec should include dataVolumeTemplates and running: true so that
// KubeVirt provisions the DataVolume and spawns the VMI automatically.
// The caller must call Delete (or the deferred Close) when done.
func NewEphemeralMachine(ctx context.Context, dynClient dynamic.Interface, k8sClient kubernetes.Interface, vm *unstructured.Unstructured) (*EphemeralMachine, error) {
	log := klog.FromContext(ctx)

	ns := vm.GetNamespace()
	name := vm.GetName()

	m := &EphemeralMachine{
		dynamic:   dynClient,
		k8s:       k8sClient,
		namespace: ns,
		name:      name,
	}

	log.Info("creating VirtualMachine", "name", name, "namespace", ns)
	if _, err := dynClient.Resource(vmGVR).Namespace(ns).Create(ctx, vm, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("creating VirtualMachine: %w", err)
	}

	return m, nil
}

// WaitForInstanceReady polls until the VMI spawned by the VirtualMachine
// reaches Running phase, then resolves the IP of the backing virt-launcher pod.
func (m *EphemeralMachine) WaitForInstanceReady(ctx context.Context) error {
	log := klog.FromContext(ctx)

	for {
		result, err := m.dynamic.Resource(vmiGVR).Namespace(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("VMI not yet created by controller, waiting...", "name", m.name)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second):
				}
				continue
			}
			return fmt.Errorf("getting VMI status: %w", err)
		}

		phase, found, err := unstructured.NestedString(result.Object, "status", "phase")
		if err != nil {
			return fmt.Errorf("reading VMI status.phase: %w", err)
		}
		if !found {
			return fmt.Errorf("VMI %q is missing status.phase", m.name)
		}
		log.Info("waiting for VMI to be running", "name", m.name, "phase", phase)

		switch phase {
		case "Failed":
			return fmt.Errorf("VMI %q entered Failed phase", m.name)
		case "Running":
			ip, err := m.waitForVMIIP(ctx)
			if err != nil {
				return err
			}
			m.podIP = ip
			log.Info("VMI is running", "name", m.name, "ip", m.podIP)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// waitForVMIIP polls the VMI status until an IP address is reported.
func (m *EphemeralMachine) waitForVMIIP(ctx context.Context) (string, error) {
	log := klog.FromContext(ctx)

	for {
		result, err := m.dynamic.Resource(vmiGVR).Namespace(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("getting VMI %q for IP: %w", m.name, err)
		}

		interfaces, _, _ := unstructured.NestedSlice(result.Object, "status", "interfaces")
		for _, iface := range interfaces {
			ifaceMap, ok := iface.(map[string]interface{})
			if !ok {
				continue
			}
			if ip, ok := ifaceMap["ipAddress"].(string); ok && ip != "" {
				return ip, nil
			}
		}

		log.Info("VMI has no IP yet, retrying...", "name", m.name)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

// IP returns the routable IP (virt-launcher pod IP) of the running VMI.
// Only valid after WaitForInstanceReady has returned nil.
func (m *EphemeralMachine) IP() string {
	return m.podIP
}

// Delete removes the VirtualMachine from the cluster. KubeVirt garbage-collects
// the owned VMI and DataVolume/PVC automatically.
func (m *EphemeralMachine) Delete(ctx context.Context) error {
	log := klog.FromContext(ctx)
	log.Info("deleting VirtualMachine", "name", m.name, "namespace", m.namespace)
	if err := m.dynamic.Resource(vmGVR).Namespace(m.namespace).Delete(ctx, m.name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("deleting VirtualMachine: %w", err)
	}
	return nil
}

// Close implements io.Closer for deferred cleanup.
func (m *EphemeralMachine) Close() error {
	return m.Delete(context.Background())
}
