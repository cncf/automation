package remote

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
	"k8s.io/klog/v2"
)

type SSHClient struct {
	sshClient *ssh.Client
}

func DialWithRetry(ctx context.Context, network, addr string, sshConfig *ssh.ClientConfig) (*SSHClient, error) {
	log := klog.FromContext(ctx)

	log.Info("dialing ssh", "addr", addr)

	// The VM is just booting, so give it some time to start responding to SSH
	attempt := 0
	maxAttempts := 60
	for {
		attempt++

		sshClient, err := ssh.Dial(network, addr, sshConfig)
		if err == nil {
			return &SSHClient{sshClient: sshClient}, nil
		}
		if attempt >= maxAttempts {
			return nil, fmt.Errorf("failed to connect to ssh on %q: %w", addr, err)
		}
		log.Info("retrying ssh connection", "attempt", attempt, "error", err)
		attempt++
		time.Sleep(2 * time.Second)
	}
}

func (s *SSHClient) Close() error {
	return s.sshClient.Close()
}

func (s *SSHClient) RunCommand(ctx context.Context, cmd string) ([]byte, error) {
	session, err := s.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return output, err
}

func (s *SSHClient) WriteFile(ctx context.Context, dir string, file string, b []byte, mode string) error {
	session, err := s.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("creating ssh session: %w", err)
	}
	defer session.Close()

	errors := make(chan error)

	writeFile := func() error {
		w, err := session.StdinPipe()
		if err != nil {
			return fmt.Errorf("getting ssh stdin: %w", err)
		}
		defer w.Close()
		if _, err := fmt.Fprintf(w, "C%s %d %s\n", mode, len(b), file); err != nil {
			return fmt.Errorf("writing to scp: %w", err)
		}
		if _, err := w.Write(b); err != nil {
			return fmt.Errorf("writing to scp: %w", err)
		}
		if _, err := fmt.Fprintf(w, "\x00"); err != nil {
			return fmt.Errorf("writing to scp: %w", err)
		}
		return nil
	}
	go func() {
		err := writeFile()
		errors <- err
	}()

	if err := session.Run("/usr/bin/scp -t " + dir); err != nil {
		return fmt.Errorf("doing scp: %w", err)
	}

	if err := <-errors; err != nil {
		return fmt.Errorf("doing scp: %w", err)
	}
	return nil
}
