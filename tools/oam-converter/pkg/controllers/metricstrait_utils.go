// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// source file: application-operator/controllers/metricstrait/metricstrait_utils.go
package controllers

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	corev1 "k8s.io/api/core/v1"
	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func CreateServiceMonitorName(trait *vzapi.MetricsTrait, appName string, compName string, portNum int) (string, error) {
	sname, err := createJobOrServiceMonitorName(trait, appName, compName, portNum)
	if err != nil {
		return "", err
	}
	return strings.Replace(sname, "_", "-", -1), nil
}
func createJobOrServiceMonitorName(trait *vzapi.MetricsTrait,appName string, compName string, portNum int) (string, error) {
	namespace := getNamespaceFromObjectMetaOrDefault(trait.ObjectMeta)
	portStr := ""
	if portNum > 0 {
		portStr = fmt.Sprintf("_%d", portNum)
	}

	finalName := fmt.Sprintf("%s_%s_%s%s", appName, namespace, compName, portStr)
	// Check for Kubernetes name length requirement
	if len(finalName) > 63 {
		finalName = fmt.Sprintf("%s_%s%s", appName, namespace, portStr)
		if len(finalName) > 63 {
			return finalName[:63], nil
		}
	}
	return finalName, nil
}
func getNamespaceFromObjectMetaOrDefault(meta metav1.ObjectMeta) string {
	name := meta.Namespace
	if name == "" {
		return "default"
	}
	return name
}
func UseHTTPSForScrapeTarget(trait *vzapi.MetricsTrait) (bool, error) {
	if trait.Spec.WorkloadReference.Kind == "VerrazzanoCoherenceWorkload" || trait.Spec.WorkloadReference.Kind == "Coherence" {
		return false, nil
	}
	// Get the namespace resource that the MetricsTrait is deployed to
	namespace := &corev1.Namespace{}

	value, ok := namespace.Labels["istio-injection"]
	if ok && value == "enabled" {
		return true, nil
	}
	return false, nil
}
func IsWLSWorkload(workload *unstructured.Unstructured) (bool, error) {
	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil {
		return false, err
	}
	// Match any version of APIVersion=weblogic.oracle and Kind=Domain
	if matched, _ := regexp.MatchString("^weblogic.oracle/.*\\.Domain$", apiVerKind); matched {
		return true, nil
	}
	return false, nil
}
func GetPortSpecs(trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec) []vzapi.PortSpec {
	ports := trait.Spec.Ports
	if len(ports) == 0 {
		// create a port spec from the existing port
		ports = []vzapi.PortSpec{{Port: trait.Spec.Port, Path: trait.Spec.Path}}
	} else {
		// if there are existing ports and a port/path setting, add the latter to the ports
		if trait.Spec.Port != nil {
			// add the port to the ports
			path := trait.Spec.Path
			if path == nil {
				path = traitDefaults.Path
			}
			portSpec := vzapi.PortSpec{
				Port: trait.Spec.Port,
				Path: path,
			}
			ports = append(ports, portSpec)
		}
	}
	return ports
}
// fetchSourceCredentialsSecretIfRequired fetches the metrics endpoint authentication credentials if a secret is provided.
func FetchSourceCredentialsSecretIfRequired(ctx context.Context, trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, workload *unstructured.Unstructured, cli client.Client) (*k8score.Secret, error) {
	secretName := trait.Spec.Secret
	// If no secret name explicitly provided use the default secret name.
	if secretName == nil && traitDefaults != nil {
		secretName = traitDefaults.Secret
	}
	// If neither an explicit or default secret name provided do not fetch a secret.
	if secretName == nil {
		return nil, nil
	}
	// Use the workload namespace for the secret to fetch.
	secretNamespace, found, err := unstructured.NestedString(workload.Object, "metadata", "namespace")
	if err != nil {
		return nil, fmt.Errorf("failed to determine namespace for secret %s: %w", *secretName, err)
	}
	if !found {
		return nil, fmt.Errorf("failed to find namespace for secret %s", *secretName)
	}
	// Fetch the secret.
	secretKey := client.ObjectKey{Namespace: secretNamespace, Name: *secretName}
	secretObj := k8score.Secret{}
	err = cli.Get(ctx, secretKey, &secretObj)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret %v: %w", secretKey, err)
	}
	return &secretObj, nil
}