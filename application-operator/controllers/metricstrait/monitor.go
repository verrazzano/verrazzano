// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"fmt"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) updatePodMonitor(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, log vzlog.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	rel := vzapi.QualifiedResourceRelation{}

	// If the metricsTrait is being disabled then return nil for the config
	if !isEnabled(trait) || workload == nil {
		return rel, controllerutil.OperationResultNone, nil
	}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := r.fetchSourceCredentialsSecretIfRequired(ctx, trait, traitDefaults, workload)
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to fetch metrics source credentials: %v", err)
	}
	if secret != nil && secret.Name == "" {
		return rel, controllerutil.OperationResultNone, err
	}

	podMonitor := promoperapi.PodMonitor{}
	pmName, err := createPodMonitorName(trait, 0)
	if err != nil {
		return rel, controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to create Pod Monitor name: %v", err)
	}
	podMonitor.SetName(pmName)
	podMonitor.SetNamespace(workload.GetNamespace())
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &podMonitor, func() error {
		return r.mutatePodMonitorFromTrait(ctx, &podMonitor, trait, workload, traitDefaults, log)
	})
	if err != nil {
		return rel, result, err
	}
	return rel, result, nil
}

// mutatePodMonitorFromTrait mutates the Pod Monitor to prepare for a create or update
// the Pod Monitor reflects the specifications of the trait and the trait defaults
func (r *Reconciler) mutatePodMonitorFromTrait(ctx context.Context, podMonitor *promoperapi.PodMonitor, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, log vzlog.VerrazzanoLogger) error {
	// Create the Pod monitor name from the trait if the label exists
	// Create the Pod Monitor selector from the trait label i fit exists
	if podMonitor.ObjectMeta.Labels == nil {
		podMonitor.ObjectMeta.Labels = map[string]string{}
	}
	podMonitor.ObjectMeta.Labels["name"] = podMonitor.GetName()
	podMonitor.Spec.Selector = metav1.LabelSelector{MatchLabels: map[string]string{appObjectMetaLabel: podMonitor.GetName()}}

	// Loop through ports in the trait and create scrape targets for each
	ports := getPortSpecs(trait, traitDefaults)
	for i := range ports {
		err := r.createPodMetricsEndpoint(ctx, podMonitor, trait, workload, traitDefaults, i)
		if err != nil {
			return log.ErrorfNewErr("Failed to create the pod metrics endpoint for the pod monitor: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) createPodMetricsEndpoint(ctx context.Context, podMonitor *promoperapi.PodMonitor, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, portIncrement int) error {
	job, err := createPodMonitorName(trait, portIncrement)
	if err != nil {
		return err
	}

	podMonitor.Spec.PodMetricsEndpoints = append(podMonitor.Spec.PodMetricsEndpoints, promoperapi.PodMetricsEndpoint{
		Port: "__meta_kubernetes_pod_annotation_verrazzano_io_metricsPort",
		Path: job,
	})
	return nil
}

// fetchSourceCredentialsSecretIfRequired fetches the metrics endpoint authentication credentials if a secret is provided.
func (r *Reconciler) fetchSourceCredentialsSecretIfRequired(ctx context.Context, trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, workload *unstructured.Unstructured) (*k8score.Secret, error) {
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
	err = r.Get(ctx, secretKey, &secretObj)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret %v: %w", secretKey, err)
	}
	return &secretObj, nil
}
