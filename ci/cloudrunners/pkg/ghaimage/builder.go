package ghaimage

import (
	"context"
	"fmt"

	"github.com/cncf/automation/cloudrunners/pkg/remote"
	"golang.org/x/crypto/ssh"
	"k8s.io/klog/v2"
)

func InstallComponents(ctx context.Context, ip string, sshConfig *ssh.ClientConfig) error {
	log := klog.FromContext(ctx)

	sshClient, err := remote.DialWithRetry(ctx, "tcp", ip+":22", sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to ssh on %q: %w", ip, err)
	}
	defer sshClient.Close()

	log.Info("copying build-image.sh to . over scp")
	if err := sshClient.WriteFile(ctx, ".", "build-image.sh", buildImage, "0755"); err != nil {
		return fmt.Errorf("doing scp of build-image.sh: %w", err)
	}
	for _, cmd := range []string{
		"sudo ./build-image.sh",
	} {
		log.Info("running ssh command", "command", cmd)

		output, err := sshClient.RunCommand(ctx, cmd)
		if err != nil {
			log.Error(err, "running ssh command", "command", cmd, "output", output)
			return fmt.Errorf("running command %q: %w", cmd, err)
		}
		log.Info("command succeeded", "command", cmd, "output", string(output))
	}

	return nil
}
