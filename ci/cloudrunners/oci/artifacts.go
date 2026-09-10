package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cncf/automation/cloudrunners/pkg/remote"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

// uploadArtifacts copies the given remote files from the runner VM and puts
// them into the OCI Object Storage bucket under <prefix>/<pod-name>/<timestamp>/.
func uploadArtifacts(ctx context.Context, sshClient *remote.SSHClient, region string, files []string) error {
	client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return fmt.Errorf("creating object storage client: %w", err)
	}
	client.SetRegion(region)

	namespace := args.artifactNamespace
	if namespace == "" {
		resp, err := client.GetNamespace(ctx, objectstorage.GetNamespaceRequest{})
		if err != nil {
			return fmt.Errorf("resolving object storage namespace: %w", err)
		}
		namespace = *resp.Value
	}

	podName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("reading pod hostname: %w", err)
	}
	prefix := path.Join(args.artifactPrefix, podName, time.Now().UTC().Format("20060102T150405Z"))

	for _, file := range files {
		content, err := sshClient.ReadFile(ctx, file)
		if err != nil {
			return err
		}
		objectName := path.Join(prefix, path.Base(file))

		contentType := "application/octet-stream"
		if strings.HasSuffix(file, ".json") {
			contentType = "application/json"
		}

		_, err = client.PutObject(ctx, objectstorage.PutObjectRequest{
			NamespaceName: common.String(namespace),
			BucketName:    common.String(args.artifactBucket),
			ObjectName:    common.String(objectName),
			ContentLength: common.Int64(int64(len(content))),
			ContentType:   common.String(contentType),
			PutObjectBody: io.NopCloser(bytes.NewReader(content)),
		})
		if err != nil {
			return fmt.Errorf("uploading %s to %s/%s: %w", file, args.artifactBucket, objectName, err)
		}
		log.Printf("uploaded artifact: %s -> oci://%s/%s/%s (%d bytes)", file, namespace, args.artifactBucket, objectName, len(content))
	}
	return nil
}
