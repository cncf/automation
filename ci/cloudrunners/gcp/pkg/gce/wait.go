package gce

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/compute/v1"
	"k8s.io/klog/v2"
)

func WaitForZonalOperation(ctx context.Context, computeClient *compute.Service, projectID, zone, operationName string) error {
	log := klog.FromContext(ctx)

	for {
		// TODO: Use Wait
		op, err := computeClient.ZoneOperations.Get(projectID, zone, operationName).Do()
		if err != nil {
			return fmt.Errorf("getting status of operation: %w", err)
		}
		if op.Status == "DONE" {
			log.Info("operation is done", "operationType", op.OperationType, "selfLink", op.SelfLink, "status", op.Status)
			return nil
		}
		time.Sleep(10 * time.Second)
		log.Info("operation not yet done", "operationType", op.OperationType, "selfLink", op.SelfLink, "status", op.Status)
	}
}

func WaitForGlobalOperation(ctx context.Context, computeClient *compute.Service, projectID, operationName string) error {
	log := klog.FromContext(ctx)

	for {
		// TODO: Use Wait
		op, err := computeClient.GlobalOperations.Get(projectID, operationName).Do()
		if err != nil {
			return fmt.Errorf("getting status of operation: %w", err)
		}
		if op.Status == "DONE" {
			log.Info("operation is done", "operationType", op.OperationType, "selfLink", op.SelfLink, "status", op.Status)
			return nil
		}
		time.Sleep(10 * time.Second)
		log.Info("operation not yet done", "operationType", op.OperationType, "selfLink", op.SelfLink, "status", op.Status)
	}
}
