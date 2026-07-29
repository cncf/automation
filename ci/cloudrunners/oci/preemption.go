package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const serviceAccountDir = "/var/run/secrets/kubernetes.io/serviceaccount"

type ephemeralRunner struct {
	Status struct {
		JobRepositoryName string `json:"jobRepositoryName"`
		WorkflowRunID     int64  `json:"workflowRunId"`
	} `json:"status"`
}

// reportPreemption records the preempted job in the rerun queue ConfigMap so
// the preemption-rerun CronJob (ci/cluster/oci/hacks/) can re-run it once the
// workflow run completes. It reads the job's repository and run ID from this
// pod's own EphemeralRunner, whose name equals the pod name.
func reportPreemption(ctx context.Context, queueConfigMap string) error {
	token, err := os.ReadFile(serviceAccountDir + "/token")
	if err != nil {
		return fmt.Errorf("reading serviceaccount token: %w", err)
	}
	namespace, err := os.ReadFile(serviceAccountDir + "/namespace")
	if err != nil {
		return fmt.Errorf("reading serviceaccount namespace: %w", err)
	}
	caCert, err := os.ReadFile(serviceAccountDir + "/ca.crt")
	if err != nil {
		return fmt.Errorf("reading serviceaccount ca.crt: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("parsing serviceaccount ca.crt")
	}

	podName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("reading pod hostname: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: caPool},
		},
	}
	apiServer := fmt.Sprintf("https://%s:%s",
		os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT"))
	ns := strings.TrimSpace(string(namespace))
	bearer := "Bearer " + strings.TrimSpace(string(token))

	runnerURL := fmt.Sprintf("%s/apis/actions.github.com/v1alpha1/namespaces/%s/ephemeralrunners/%s",
		apiServer, ns, podName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, runnerURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", bearer)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("getting ephemeralrunner %s: %w", podName, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("getting ephemeralrunner %s: HTTP %d: %s", podName, resp.StatusCode, body)
	}

	var runner ephemeralRunner
	if err := json.Unmarshal(body, &runner); err != nil {
		return fmt.Errorf("parsing ephemeralrunner %s: %w", podName, err)
	}
	if runner.Status.WorkflowRunID == 0 || runner.Status.JobRepositoryName == "" {
		return fmt.Errorf("ephemeralrunner %s has no job info (workflowRunId=%d, jobRepositoryName=%q)",
			podName, runner.Status.WorkflowRunID, runner.Status.JobRepositoryName)
	}

	entry, err := json.Marshal(map[string]any{
		"repo":       runner.Status.JobRepositoryName,
		"run_id":     runner.Status.WorkflowRunID,
		"first_seen": time.Now().Unix(),
	})
	if err != nil {
		return err
	}
	patch, err := json.Marshal(map[string]any{
		"data": map[string]string{
			fmt.Sprintf("run-%d", runner.Status.WorkflowRunID): string(entry),
		},
	})
	if err != nil {
		return err
	}

	queueURL := fmt.Sprintf("%s/api/v1/namespaces/%s/configmaps/%s", apiServer, ns, queueConfigMap)
	req, err = http.NewRequestWithContext(ctx, http.MethodPatch, queueURL, bytes.NewReader(patch))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", bearer)
	req.Header.Set("Content-Type", "application/merge-patch+json")
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("patching configmap %s: %w", queueConfigMap, err)
	}
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("patching configmap %s: HTTP %d: %s", queueConfigMap, resp.StatusCode, body)
	}

	return nil
}
