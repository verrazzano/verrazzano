// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package configmap

import (
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"strings"
)

const (
	ConfigMapName             = "metrics-workload-resources"
	mutatingWebhookConfigName = "verrazzano-application-metrics-binding"
	WebhookName               = "metrics-binding-generator-workload.verrazzano.io"
	finalizerName             = "configmap.finalizers.verrazzano.io/finalizer"
	resourceIdentifier        = "workloads"
)

var defaultResourceList = []string{"deployment", "statefulset", "replicaset", "pod", "domains", "coherences"}

// getWorkloadWebhook returns the webhook for the Workload Resource
func getWorkloadWebhook(mwc *admissionv1.MutatingWebhookConfiguration) *admissionv1.MutatingWebhook {
	for _, webhook := range mwc.Webhooks {
		if webhook.Name == WebhookName {
			return &webhook
		}
	}
	return nil
}

// formatWorkloadResources formats the new workload data to the old workload list
func formatWorkloadResources(newWorkloadData string, webhookWorkloads []string) []string {
	// Split the ConfigMap data by new lines
	splitWorkloads := strings.Split(newWorkloadData, "\n")

	// Generate a map so that no duplicate values are copied
	uniqueWorkloads := generateUniqueMap(webhookWorkloads)

	// Insert all workload resources
	for _, workload := range splitWorkloads {
		if len(workload) > 1 && !uniqueWorkloads[workload] {
			webhookWorkloads = append(webhookWorkloads, strings.ToLower(strings.TrimSpace(workload)))
		}
	}
	return webhookWorkloads
}

// generateUniqueMap generates a map with the unique values in a string list
func generateUniqueMap(values []string) map[string]bool {
	uniqueMap := map[string]bool{}
	for _, value := range values {
		if result := uniqueMap[value]; !result {
			uniqueMap[value] = true
		}
	}
	return uniqueMap
}
