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
				time.Sleep(5 * time.Second)
				continue
			}
			return fmt.Errorf("getting VMI status: %w", err)
		}

		phase, _, _ := unstructured.NestedString(result.Object, "status", "phase")
		log.Info("waiting for VMI to be running", "name", m.name, "phase", phase)

		switch phase {
		case "Failed":
			return fmt.Errorf("VMI %q entered Failed phase", m.name)
		case "Running":
			pods, err := m.k8s.CoreV1().Pods(m.namespace).List(ctx, metav1.ListOptions{
LabelSelector: fmt.Sprintf("kubevirt.io/domain=%s", m.name),
})
			if err != nil {
				return fmt.Errorf("listing virt-launcher pods for VMI %q: %w", m.name, err)
			}
			if len(pods.Items) == 0 {
				return fmt.Errorf("no virt-launcher pod found for VMI %q", m.name)
			}
			m.podIP = pods.Items[0].Status.PodIP
			if m.podIP == "" {
				return fmt.Errorf("virt-launcher pod for VMI %q has no IP yet", m.name)
			}
			log.Info("VMI is running", "name", m.name, "podIP", m.podIP)
			return nil
		}

		time.Sleep(5 * time.Second)
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
