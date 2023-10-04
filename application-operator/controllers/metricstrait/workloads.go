// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"fmt"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"github.com/verrazzano/verrazzano/application-operator/controllers/reconcileresults"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// createOrUpdateWorkloads creates or updates resources related to this trait
// The related resources are the workload children and the Prometheus config
func (r *Reconciler) createOrUpdateRelatedWorkloads(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, children []*unstructured.Unstructured, log vzlog.VerrazzanoLogger) *reconcileresults.ReconcileResults {
	log.Debugf("Creating or updating workload children of the Prometheus workload: %s", workload.GetName())
	status := reconcileresults.ReconcileResults{}
	for _, child := range children {
		switch child.GroupVersionKind() {
		case k8sapps.SchemeGroupVersion.WithKind(deploymentKind):
			// In the case of VerrazzanoHelidonWorkload, it isn't unwrapped so we need to check to see
			// if the workload is a wrapper kind in addition to checking to see if the owner is a wrapper kind.
			// In the case of a wrapper kind or owner, the status is not being updated here as this is handled by the
			// wrapper owner which is the corresponding Verrazzano wrapper resource/controller.
			if !vznav.IsOwnedByVerrazzanoWorkloadKind(workload) && !vznav.IsVerrazzanoWorkloadKind(workload) {
				status.RecordOutcome(r.updateRelatedDeployment(ctx, trait, workload, traitDefaults, child, log))
			}
		case k8sapps.SchemeGroupVersion.WithKind(statefulSetKind):
			// In the case of a workload having an owner that is a wrapper kind, the status is not being updated here
			// as this is handled by the wrapper owner which is the corresponding Verrazzano wrapper resource/controller.
			if !vznav.IsOwnedByVerrazzanoWorkloadKind(workload) {
				status.RecordOutcome(r.updateRelatedStatefulSet(ctx, trait, workload, traitDefaults, child, log))
			}
		case k8score.SchemeGroupVersion.WithKind(podKind):
			// In the case of a workload having an owner that is a wrapper kind, the status is not being updated here
			// as this is handled by the wrapper owner which is the corresponding Verrazzano wrapper resource/controller.
			if !vznav.IsOwnedByVerrazzanoWorkloadKind(workload) {
				status.RecordOutcome(r.updateRelatedPod(ctx, trait, workload, traitDefaults, child, log))
			}
		}
	}
	return &status
}

// updateRelatedDeployment updates the labels and annotations of a related workload deployment.
// For example containerized workloads produce related deployments.
func (r *Reconciler) updateRelatedDeployment(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured, log vzlog.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	deployment := &k8sapps.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	ref := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}

	err := r.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)

	if err != nil && !apierrors.IsNotFound(err) {
		return ref, controllerutil.OperationResultNone, fmt.Errorf("failed to getworkload child deployment %s: %v", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta).Name, err)
	}

	if apierrors.IsNotFound(err) || deployment.CreationTimestamp.IsZero() {
		log.Debug("Workload child deployment not found")
		return ref, controllerutil.OperationResultNone, apierrors.NewNotFound(schema.GroupResource{Group: deployment.APIVersion, Resource: deployment.Kind}, deployment.Name)
	}

	replicaSets := &k8sapps.ReplicaSetList{}
	err = r.List(ctx, replicaSets, client.InNamespace(deployment.Namespace))
	if err != nil && !apierrors.IsNotFound(err) {
		return ref, controllerutil.OperationResultNone, fmt.Errorf("failed to get replicasets of workload child deployment %s: %v", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta).Name, err)
	}

	if apierrors.IsNotFound(err) || len(replicaSets.Items) == 0 {
		log.Debug("Replicasets of Workload child deployment not found")
		return ref, controllerutil.OperationResultNone, apierrors.NewNotFound(schema.GroupResource{Group: replicaSets.APIVersion, Resource: replicaSets.Kind}, deployment.Name)
	}

	for _, replicaSet := range replicaSets.Items {
		res, err := r.updateRelatedPods(ctx, trait, workload, traitDefaults, log, ref.Namespace, replicaSet.Kind, replicaSet.Name, replicaSet.APIVersion)
		if err != nil && !apierrors.IsNotFound(err) {
			return ref, res, err
		}
	}

	return ref, controllerutil.OperationResultUpdated, nil

}

// updateRelatedStatefulSet updates the labels and annotations of a related workload stateful set.
// For example coherence workloads produce related stateful sets.
func (r *Reconciler) updateRelatedStatefulSet(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured, log vzlog.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	log.Debugf("Update workload stateful set %s", vznav.GetNamespacedNameFromUnstructured(child))
	ref := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}
	statefulSet := &k8sapps.StatefulSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}

	err := r.Get(ctx, client.ObjectKeyFromObject(statefulSet), statefulSet)

	if err != nil && !apierrors.IsNotFound(err) {
		return ref, controllerutil.OperationResultNone, fmt.Errorf("unable to fetch workload child statefulset %s: %v", vznav.GetNamespacedNameFromObjectMeta(statefulSet.ObjectMeta).Name, err)
	}

	if apierrors.IsNotFound(err) || statefulSet.CreationTimestamp.IsZero() {
		log.Debug("Workload child statefulset not found")
		return ref, controllerutil.OperationResultNone, apierrors.NewNotFound(schema.GroupResource{Group: statefulSet.APIVersion, Resource: statefulSet.Kind}, statefulSet.Name)
	}

	res, err := r.updateRelatedPods(ctx, trait, workload, traitDefaults, log, ref.Namespace, statefulSet.Kind, statefulSet.Name, statefulSet.APIVersion)
	return ref, res, err
}

// updateRelatedPod updates the labels and annotations of a related workload pod.
// For example WLS workloads produce related pods.
func (r *Reconciler) updateRelatedPod(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured, log vzlog.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	log.Debug("Update workload pod %s", vznav.GetNamespacedNameFromUnstructured(child))
	pod := &k8score.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	return r.updatePod(ctx, trait, workload, traitDefaults, log, *pod)
}

// NewTraitDefaultsForWLSDomainWorkload creates metrics trait default values for a WLS domain workload.
func (r *Reconciler) NewTraitDefaultsForWLSDomainWorkload(ctx context.Context, workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
	// Port precedence: trait, workload annotation, default
	port := defaultWLSAdminScrapePort
	path := defaultWLSScrapePath
	secret, err := r.fetchWLSDomainCredentialsSecretName(ctx, workload)
	if err != nil {
		return nil, err
	}
	return &vzapi.MetricsTraitSpec{
		Ports: []vzapi.PortSpec{{
			Port: &port,
			Path: &path,
		}},
		Path:    &path,
		Secret:  secret,
		Scraper: &r.Scraper}, nil
}

// NewTraitDefaultsForCOHWorkload creates metrics trait default values for a Coherence workload.
func (r *Reconciler) NewTraitDefaultsForCOHWorkload(ctx context.Context, workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
	path := defaultScrapePath
	port := defaultCohScrapePort
	var secret *string

	enabled, p, s, err := r.fetchCoherenceMetricsSpec(ctx, workload)
	if err != nil {
		return nil, err
	}
	if enabled == nil || *enabled {
		if p != nil {
			port = *p
		}
		if s != nil {
			secret = s
		}
	}
	return &vzapi.MetricsTraitSpec{
		Ports: []vzapi.PortSpec{{
			Port: &port,
			Path: &path,
		}},
		Path:    &path,
		Secret:  secret,
		Scraper: &r.Scraper}, nil
}

// NewTraitDefaultsForGenericWorkload creates metrics trait default values for a containerized workload.
func (r *Reconciler) NewTraitDefaultsForGenericWorkload() (*vzapi.MetricsTraitSpec, error) {
	port := defaultScrapePort
	path := defaultScrapePath
	return &vzapi.MetricsTraitSpec{
		Ports: []vzapi.PortSpec{{
			Port: &port,
			Path: &path,
		}},
		Path:    &path,
		Secret:  nil,
		Scraper: &r.Scraper}, nil
}

// fetchCoherenceMetricsSpec fetches the metrics configuration from the Coherence workload resource spec.
// These configuration values are used in the population of the Prometheus scraper configuration.
func (r *Reconciler) fetchCoherenceMetricsSpec(ctx context.Context, workload *unstructured.Unstructured) (*bool, *int, *string, error) {
	// determine if metrics is enabled
	enabled, found, err := unstructured.NestedBool(workload.Object, "spec", "coherence", "metrics", "enabled")
	if err != nil {
		return nil, nil, nil, err
	}
	var e *bool
	if found {
		e = &enabled
	}

	// get the metrics port
	port, found, err := unstructured.NestedInt64(workload.Object, "spec", "coherence", "metrics", "port")
	if err != nil {
		return nil, nil, nil, err
	}
	var p *int
	if found {
		p2 := int(port)
		p = &p2
	}

	// get the secret if ssl is enabled
	enabled, found, err = unstructured.NestedBool(workload.Object, "spec", "coherence", "metrics", "ssl", "enabled")
	if err != nil {
		return nil, nil, nil, err
	}
	var s *string
	if found && enabled {
		secret, found, err := unstructured.NestedString(workload.Object, "spec", "coherence", "metrics", "ssl", "secrets")
		if err != nil {
			return nil, nil, nil, err
		}
		if found {
			s = &secret
		}
	}
	return e, p, s, nil
}

// fetchWLSDomainCredentialsSecretName fetches the credentials from the WLS workload resource (i.e. domain).
// These credentials are used in the population of the Prometheus scraper configuration.
func (r *Reconciler) fetchWLSDomainCredentialsSecretName(ctx context.Context, workload *unstructured.Unstructured) (*string, error) {
	secretName, found, err := unstructured.NestedString(workload.Object, "spec", "webLogicCredentialsSecret", "name")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &secretName, nil
}

func (r *Reconciler) updateRelatedPods(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, log vzlog.VerrazzanoLogger, namespace string, ownerKind string, ownerName string, ownerAPIVersion string) (controllerutil.OperationResult, error) {
	pods := &k8score.PodList{}
	err := r.List(ctx, pods, client.InNamespace(namespace))
	if err != nil && !apierrors.IsNotFound(err) {
		return controllerutil.OperationResultNone, fmt.Errorf("unable to fetch pods from namespace %s: %v", namespace, err)
	}

	if apierrors.IsNotFound(err) || len(pods.Items) == 0 {
		log.Debugf("pods of %s %s not found", ownerKind, ownerName)
		return controllerutil.OperationResultNone, apierrors.NewNotFound(schema.GroupResource{Group: pods.APIVersion, Resource: pods.Kind}, ownerName)
	}

	for _, pod := range pods.Items {
		for _, podOwnerRef := range pod.GetOwnerReferences() {
			if podOwnerRef.APIVersion == ownerAPIVersion && podOwnerRef.Kind == ownerKind && podOwnerRef.Name == ownerName {
				_, _, err := r.updatePod(ctx, trait, workload, traitDefaults, log, pod)
				if err != nil && !apierrors.IsNotFound(err) {
					return controllerutil.OperationResultNone, fmt.Errorf("failed to update labels for pod %s of %s %s: %v", vznav.GetNamespacedNameFromObjectMeta(pod.ObjectMeta).Name, ownerKind, ownerName, err)
				}
			}

		}
	}

	return controllerutil.OperationResultUpdated, nil

}

// updatePod updates the labels and annotations of a workload pod.
func (r *Reconciler) updatePod(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, log vzlog.VerrazzanoLogger, pod k8score.Pod) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	rel := vzapi.QualifiedResourceRelation{APIVersion: pod.APIVersion, Kind: pod.Kind, Namespace: pod.GetNamespace(), Name: pod.GetName(), Role: sourceRole}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, &pod, func() error {
		// If the statefulset was not found don't attempt to create or update it.
		if pod.CreationTimestamp.IsZero() {
			log.Debug("Workload child pod not found")
			return apierrors.NewNotFound(schema.GroupResource{Group: pod.APIVersion, Resource: pod.Kind}, pod.Name)
		}
		pod.ObjectMeta.Annotations = MutateAnnotations(trait, traitDefaults, pod.ObjectMeta.Annotations)
		pod.ObjectMeta.Labels = MutateLabels(trait, workload, pod.ObjectMeta.Labels)
		return nil
	})

	if err != nil && !apierrors.IsNotFound(err) {
		return rel, res, log.ErrorfThrottledNewErr("Failed to update workload child pod %s: %v", vznav.GetNamespacedNameFromObjectMeta(pod.ObjectMeta), err)
	}

	return rel, controllerutil.OperationResultUpdated, nil
}
