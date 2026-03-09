package main

import (
"context"
"fmt"
"log"
"os"
"strings"
"time"

kubevirtpkg "github.com/cncf/automation/cloudrunners/kubevirt/pkg/kubevirt"
"github.com/cncf/automation/cloudrunners/pkg/remote"
"github.com/spf13/cobra"
"golang.org/x/crypto/ssh"
"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
"k8s.io/client-go/dynamic"
"k8s.io/client-go/kubernetes"
"k8s.io/client-go/rest"
"k8s.io/client-go/tools/clientcmd"
)

var Cmd = &cobra.Command{
	Use:  "gha-runner",
	Long: "Run a GitHub Actions runner (on KubeVirt)",
	RunE: run,
}

var args struct {
	debug               bool
	arch                string
	namespace           string
	datasource          string
	datasourceNamespace string
	storageClassName    string
	diskSize            string
	cpu                 int
	memory              string
	runEnv              string
	kubeconfig          string
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

	// Build Kubernetes clients from kubeconfig or in-cluster config.
	restConfig, err := buildRestConfig(args.kubeconfig)
	if err != nil {
		return fmt.Errorf("building rest config: %w", err)
	}
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("creating dynamic client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("creating kubernetes client: %w", err)
	}

	// Create SSH Key Pair.
	sshKeyPair, err := remote.CreateSSHKeyPair()
	if err != nil {
		return fmt.Errorf("creating ssh key pair: %w", err)
	}

	// cloud-init: inject the ephemeral SSH public key.
	userData := fmt.Sprintf(`#cloud-config
password: ubuntu
chpasswd: { expire: False }
ssh_authorized_keys:
  - %s`, strings.TrimSpace(sshKeyPair.PublicKey))

	name := fmt.Sprintf("gha-runner-%s-%s", args.arch, time.Now().Format("20060102-150405"))
	rootDiskName := name + "-rootdisk"

	// Build VirtualMachine with embedded dataVolumeTemplates.
	vm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": args.namespace,
			},
			"spec": map[string]interface{}{
				"running": true,
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"domain": map[string]interface{}{
							"cpu": map[string]interface{}{
								"cores": int64(args.cpu),
							},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"memory": args.memory,
								},
							},
							"devices": map[string]interface{}{
								"disks": []interface{}{
									map[string]interface{}{
										"name": "rootdisk",
										"disk": map[string]interface{}{
											"bus": "virtio",
										},
									},
									map[string]interface{}{
										"name": "cloudinitdisk",
										"disk": map[string]interface{}{
											"bus": "virtio",
										},
									},
								},
								"interfaces": []interface{}{
									map[string]interface{}{
										"name":       "default",
										"masquerade": map[string]interface{}{},
										"ports": []interface{}{
											map[string]interface{}{
												"port": int64(22),
											},
										},
									},
								},
							},
						},
						"networks": []interface{}{
							map[string]interface{}{
								"name": "default",
								"pod":  map[string]interface{}{},
							},
						},
						"volumes": []interface{}{
							map[string]interface{}{
								"name": "rootdisk",
								"dataVolume": map[string]interface{}{
									"name": rootDiskName,
								},
							},
							map[string]interface{}{
								"name": "cloudinitdisk",
								"cloudInitNoCloud": map[string]interface{}{
									"userData": userData,
								},
							},
						},
					},
				},
				"dataVolumeTemplates": []interface{}{
					map[string]interface{}{
						"metadata": map[string]interface{}{
							"name": rootDiskName,
						},
						"spec": map[string]interface{}{
							"sourceRef": map[string]interface{}{
								"kind":      "DataSource",
								"name":      args.datasource,
								"namespace": args.datasourceNamespace,
							},
							"pvc": map[string]interface{}{
								"storageClassName": args.storageClassName,
								"volumeMode":       "Block",
								"accessModes":      []interface{}{"ReadWriteOnce"},
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{
										"storage": args.diskSize,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	machine, err := kubevirtpkg.NewEphemeralMachine(ctx, dynClient, k8sClient, vm)
	if err != nil {
		return fmt.Errorf("failed to create machine: %w", err)
	}

	defer func() {
		if err := machine.Delete(context.Background()); err != nil {
			log.Printf("failed to delete machine: %v", err)
		}
	}()

	// Wait for the VMI (spawned from the VM) to reach Running phase.
	if err := machine.WaitForInstanceReady(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance to be ready: %w", err)
	}

	ip := machine.IP()
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
		"sudo setfacl -m u:ubuntu:rw /var/run/docker.sock",
		"sudo sysctl fs.inotify.max_user_instances=1280",
		"sudo sysctl fs.inotify.max_user_watches=655360",
		"export PATH=$PATH:/home/ubuntu/.local/bin && export HOME=/home/ubuntu && export NVM_DIR=/home/ubuntu/.nvm && bash -x /home/ubuntu/run.sh --jitconfig \"${ACTIONS_RUNNER_INPUT_JITCONFIG}\"",
	}

	for _, cmd := range commands {
		log.Println("running ssh command", "command", cmd)

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

// buildRestConfig returns a *rest.Config preferring, in order:
//  1. An explicit --kubeconfig path.
//  2. The in-cluster service-account config (when running inside a pod).
//  3. The default kubeconfig file / KUBECONFIG env var.
func buildRestConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	return cc.ClientConfig()
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
		"amd64",
		"Machine architecture (amd64 or arm64)",
	)
	flags.StringVar(
&args.namespace,
		"namespace",
		"machines",
		"Kubernetes namespace in which the VirtualMachine will be created",
	)
	flags.StringVar(
&args.datasource,
		"datasource",
		"",
		"CDI DataSource name for the root disk (e.g. ubuntu-24.04-x86-gha-image). Derived from --arch and --running-environment if not set.",
	)
	flags.StringVar(
&args.datasourceNamespace,
		"datasource-namespace",
		"arc-systems",
		"Namespace where the CDI DataSource lives",
	)
	flags.StringVar(
&args.storageClassName,
		"storage-class",
		"oci-bv-immediate",
		"StorageClass for the root disk PVC",
	)
	flags.StringVar(
&args.diskSize,
		"disk-size",
		"500Gi",
		"PVC size for the root disk",
	)
	flags.IntVar(
&args.cpu,
		"cpu",
		8,
		"Number of CPU cores for the VM",
	)
	flags.StringVar(
&args.memory,
		"memory",
		"32Gi",
		"Memory for the VM, e.g. 8Gi or 32Gi",
	)
	flags.StringVar(
&args.runEnv,
		"running-environment",
		"production",
		"Running environment: production or ci",
	)
	flags.StringVar(
&args.kubeconfig,
		"kubeconfig",
		"",
		"Path to kubeconfig file (uses in-cluster config or KUBECONFIG env var if not set)",
	)
}
