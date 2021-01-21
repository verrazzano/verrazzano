// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/controllers/reconcileresults"
	"k8s.io/apimachinery/pkg/util/rand"

	"strconv"

	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Kubernetes resource Kinds
	configMapKind   = "ConfigMap"
	deploymentKind  = "Deployment"
	statefulSetKind = "StatefulSet"
	podKind         = "Pod"

	// In code defaults for metrics trait configuration
	defaultScrapePort    = 8080
	defaultCohScrapePort = 9612
	defaultScrapePath    = "/metrics"
	defaultWLSScrapePath = "/wls-exporter/metrics"

	// The finalizer name used by this controller
	finalizerName = "metricstrait.finalizers.verrazzano.io"

	// Markers used during the processing of prometheus scrape configurations
	prometheusConfigKey          = "prometheus.yml"
	prometheusScrapeConfigsLabel = "scrape_configs"
	prometheusJobNameLabel       = "job_name"

	// Annotation names for metrics read by the controller
	prometheusPortAnnotation = "prometheus.io/port"
	prometheusPathAnnotation = "prometheus.io/path"

	// Annotation names for metrics set by the controller
	verrazzanoMetricsPortAnnotation    = "verrazzano.io/metricsPort"
	verrazzanoMetricsPathAnnotation    = "verrazzano.io/metricsPath"
	verrazzanoMetricsEnabledAnnotation = "verrazzano.io/metricsEnabled"

	// Label names for the OAM application and component references
	appObjectMetaLabel  = "app.oam.dev/name"
	compObjectMetaLabel = "app.oam.dev/component"

	// basicAuthLabel config label for Prometheus basic auth
	basicAuthLabel = "basic_auth"
	// basicAuthUsernameLabel config label for Prometheus username
	basicAuthUsernameLabel = "username"
	// basicPathPasswordLabel config label for Prometheus password
	basicPathPasswordLabel = "password"

	// Template placeholders for the prometheus scrape config template
	appNameHolder   = "##APP_NAME##"
	compNameHolder  = "##COMP_NAME##"
	jobNameHolder   = "##JOB_NAME##"
	namespaceHolder = "##NAMESPACE##"

	// Roles for use in qualified resource relations
	scraperRole = "scraper"
	sourceRole  = "source"
)

// prometheusScrapeConfigTemplate configuration for general prometheus scrape target template
// Used to add new scrape config to a pormetheus configmap
const prometheusScrapeConfigTemplate = `job_name: ##JOB_NAME##
kubernetes_sd_configs:
- role: pod
  namespaces:
    names:
    - ##NAMESPACE##
relabel_configs:
- action: keep
  source_labels: [__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled,__meta_kubernetes_pod_label_app_oam_dev_name,__meta_kubernetes_pod_label_app_oam_dev_component]
  regex: true;##APP_NAME##;##COMP_NAME##
- action: replace
  source_labels: [__meta_kubernetes_pod_annotation_verrazzano_io_metricsPath]
  target_label: __metrics_path__
  regex: (.+)
- action: replace
  source_labels: [__address__, __meta_kubernetes_pod_annotation_verrazzano_io_metricsPort]
  target_label: __address__
  regex: ([^:]+)(?::\d+)?;(\d+)
  replacement: $1:$2
- action: replace
  source_labels: [__meta_kubernetes_namespace]
  target_label: namespace
  regex: (.*)
  replacement: $1
- action: labelmap
  regex: __meta_kubernetes_pod_label_(.+)
- action: replace
  source_labels: [__meta_kubernetes_pod_name]
  target_label: pod_name
- action: labeldrop
  regex: '(controller_revision_hash)'
- action: replace
  source_labels: [name]
  target_label: webapp
  regex: '.*/(.*)$'
  replacement: $1
`

// Reconciler reconciles a MetricsTrait object
type Reconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Scraper string
}

// SetupWithManager creates a controller and adds it to the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.MetricsTrait{}).
		Complete(r)
}

// Reconcile reconciles a metrics trait with related resources
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=metricstraits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=metricstraits/status,verbs=get;update;patch
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconcile metrics trait", "trait", req.NamespacedName)
	ctx := context.Background()
	var err error

	// Fetch the trait.
	var trait *vzapi.MetricsTrait
	if trait, err = vznav.FetchTrait(ctx, r, r.Log, req.NamespacedName); err != nil || trait == nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if trait.DeletionTimestamp.IsZero() {
		return r.reconcileTraitCreateOrUpdate(ctx, trait)
	}
	return r.reconcileTraitDelete(ctx, trait)
}

// reconcileTraitDelete reconciles a metrics trait that is being deleted.
func (r *Reconciler) reconcileTraitDelete(ctx context.Context, trait *vzapi.MetricsTrait) (ctrl.Result, error) {
	status := r.deleteOrUpdateObsoleteResources(ctx, trait, &reconcileresults.ReconcileResults{})
	// Only remove the finalizer if all related resources were successfully updated.
	if !status.ContainsErrors() {
		r.removeFinalizerIfRequired(ctx, trait)
	}
	return r.updateTraitStatus(ctx, trait, status)
}

// reconcileTraitCreateOrUpdate reconciles a metrics trait that is being created or updated.
func (r *Reconciler) reconcileTraitCreateOrUpdate(ctx context.Context, trait *vzapi.MetricsTrait) (ctrl.Result, error) {
	var err error

	// Add finalizer if required.
	if err = r.addFinalizerIfRequired(ctx, trait); err != nil {
		return reconcile.Result{}, err
	}

	// Fetch workload resource using information from the trait
	var workload *unstructured.Unstructured
	if workload, err = vznav.FetchWorkloadFromTrait(ctx, r, r.Log, trait); err != nil {
		return reconcile.Result{}, err
	}

	// Resolve trait defaults from the trait and the workload.
	var traitDefaults *vzapi.MetricsTraitSpec
	if traitDefaults, err = r.fetchTraitDefaults(ctx, workload); err != nil {
		return reconcile.Result{}, err
	}

	var scraper *k8sapps.Deployment
	if scraper, err = r.fetchPrometheusDeploymentFromTrait(ctx, trait, traitDefaults); err != nil {
		return reconcile.Result{}, err
	}

	// Find the child resources of the workload based on the childResourceKinds from the
	// workload definition, workload uid and the ownerReferences of the children.
	var children []*unstructured.Unstructured
	if children, err = vznav.FetchWorkloadChildren(ctx, r, r.Log, workload); err != nil {
		return reconcile.Result{}, err
	}

	// Create or update the related resources of the trait and collect the outcomes.
	status := r.createOrUpdateRelatedResources(ctx, trait, workload, traitDefaults, scraper, children)
	// Delete or update any previously (but no longer) related resources of the trait.
	status = r.deleteOrUpdateObsoleteResources(ctx, trait, status)

	// Update the status of the trait resource using the outcomes of the create or update.
	return r.updateTraitStatus(ctx, trait, status)
}

// addFinalizerIfRequired adds the finalizer to the trait if required
// The finalizer is only added if the trait is not being deleted and the finalizer has not previously been added
func (r *Reconciler) addFinalizerIfRequired(ctx context.Context, trait *vzapi.MetricsTrait) error {
	if trait.GetDeletionTimestamp().IsZero() && !stringSliceContainsString(trait.Finalizers, finalizerName) {
		traitName := vznav.GetNamespacedNameFromObjectMeta(trait.ObjectMeta).String()
		r.Log.Info("Adding finalizer from trait", "trait", traitName)
		trait.Finalizers = append(trait.Finalizers, finalizerName)
		if err := r.Update(ctx, trait); err != nil {
			r.Log.Error(err, "failed to add finalizer to trait", "trait", traitName)
			return err
		}
	}
	return nil
}

// removeFinalizerIfRequired removes the finalizer from the trait if required
// The finalizer is only removed if the trait is being deleted and the finalizer had been added
func (r *Reconciler) removeFinalizerIfRequired(ctx context.Context, trait *vzapi.MetricsTrait) error {
	if !trait.DeletionTimestamp.IsZero() && stringSliceContainsString(trait.Finalizers, finalizerName) {
		traitName := vznav.GetNamespacedNameFromObjectMeta(trait.ObjectMeta).String()
		r.Log.Info("Removing finalizer from trait", "trait", traitName)
		trait.Finalizers = removeStringFromStringSlice(trait.Finalizers, finalizerName)
		if err := r.Update(ctx, trait); err != nil {
			r.Log.Error(err, "failed to remove finalizer to trait", "trait", traitName)
			return err
		}
	}
	return nil
}

// createOrUpdateRelatedResources creates or updates resources related to this trait
// The related resources are the workload children and the prometheus config
func (r *Reconciler) createOrUpdateRelatedResources(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, deployment *k8sapps.Deployment, children []*unstructured.Unstructured) *reconcileresults.ReconcileResults {
	status := reconcileresults.ReconcileResults{}
	for _, child := range children {
		switch child.GroupVersionKind() {
		case k8sapps.SchemeGroupVersion.WithKind(deploymentKind):
			status.RecordOutcome(r.updateRelatedDeployment(ctx, trait, workload, traitDefaults, child))
		case k8sapps.SchemeGroupVersion.WithKind(statefulSetKind):
			status.RecordOutcome(r.updateRelatedStatefulSet(ctx, trait, workload, traitDefaults, child))
		case k8score.SchemeGroupVersion.WithKind(podKind):
			status.RecordOutcome(r.updateRelatedPod(ctx, trait, workload, traitDefaults, child))
		}
	}
	status.RecordOutcome(r.updatePrometheusScraperConfigMap(ctx, trait, workload, traitDefaults, deployment))
	return &status
}

// deleteOrUpdateObsoleteResources deletes or updates resources that should no longer be related to this trait.
// This includes previous scrapers when the scraper has changed.
// This also includes previous workload children that are no longer referenced.
func (r *Reconciler) deleteOrUpdateObsoleteResources(ctx context.Context, trait *vzapi.MetricsTrait, status *reconcileresults.ReconcileResults) *reconcileresults.ReconcileResults {
	// For each reference in the trait status references but not in the reconcile status
	//   For references of role "scraper" attempt to remove the scrape config
	//   For references of role "source" attempt to remove the scrape annotations
	//   If the reference is not found or updated dont' add it to the reconcile status
	//   Otherwise carry the reference over in the status as an error.

	// Cleanup the relations that are in the trait status relations but not in the reconcile status.
	update := reconcileresults.ReconcileResults{}
	for _, rel := range trait.Status.Resources {
		if !status.ContainsRelation(rel) {
			switch rel.Role {
			case scraperRole:
				update.RecordOutcomeIfError(r.deleteOrUpdateScraperConfigMap(ctx, trait, rel)) // Need to pass down traitDefaults, current scraper or current scraper deployment
			case sourceRole:
				update.RecordOutcomeIfError(r.deleteOrUpdateMetricSourceResource(ctx, trait, rel))
			default:
				update.RecordOutcome(rel, controllerutil.OperationResultNone, fmt.Errorf("unknown related resource role %s", rel.Role))
			}
		}
	}
	// Copy the reconcile outcomes from the current reconcile.
	for i, rel := range status.Relations {
		if !update.ContainsRelation(rel) {
			update.RecordOutcome(status.Relations[i], status.Results[i], status.Errors[i])
		}
	}
	return &update
}

// deleteOrUpdateMetricSourceResource deletes or updates the related resources that are the source of metrics.
// These are the children of the workloads.  For example for containerized workloads these are deployments.
// For WLS workloads these are pods.
func (r *Reconciler) deleteOrUpdateMetricSourceResource(ctx context.Context, trait *vzapi.MetricsTrait, rel vzapi.QualifiedResourceRelation) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	child := unstructured.Unstructured{}
	child.SetAPIVersion(rel.APIVersion)
	child.SetKind(rel.Kind)
	child.SetNamespace(rel.Namespace)
	child.SetName(rel.Name)
	switch rel.Kind {
	case "Deployment":
		return r.updateRelatedDeployment(ctx, trait, nil, nil, &child)
	case "StatefulSet":
		return r.updateRelatedStatefulSet(ctx, trait, nil, nil, &child)
	case "Pod":
		return r.updateRelatedPod(ctx, trait, nil, nil, &child)
	default:
		return rel, controllerutil.OperationResultNone, fmt.Errorf("unknown source kind %s", rel.Kind)
	}
}

// deleteOrUpdateScraperConfigMap cleans up a scraper (i.e. prometheus) configmap.
// The scraper config for the trait is removed if present.
func (r *Reconciler) deleteOrUpdateScraperConfigMap(ctx context.Context, trait *vzapi.MetricsTrait, rel vzapi.QualifiedResourceRelation) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	deployment := &k8sapps.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Namespace: rel.Namespace, Name: rel.Name}, deployment)
	if err != nil {
		return rel, controllerutil.OperationResultNone, client.IgnoreNotFound(err)
	}
	return r.updatePrometheusScraperConfigMap(ctx, trait, nil, nil, deployment)
}

// updatePrometheusScraperConfigMap updates the prometheus scraper configmap.
// This updates only the scrape_configs section of the prometheus configmap.
// Only the rules for the provided trait will be affected.
// trait - The trait to update scrape_config rules for.
// traitDefaults - Default to use for values not provided in the trait.
// deployment - The prometheus deployment.
func (r *Reconciler) updatePrometheusScraperConfigMap(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, deployment *k8sapps.Deployment) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	rel := vzapi.QualifiedResourceRelation{APIVersion: deployment.APIVersion, Kind: deployment.Kind, Name: deployment.Name, Namespace: deployment.Namespace, Role: scraperRole}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := r.fetchSourceCredentialsSecretIfRequired(ctx, trait, traitDefaults, workload)
	if err != nil {
		return rel, controllerutil.OperationResultNone, err
	}

	configmapName, err := r.findPrometheusScrapeConfigMapNameFromDeployment(deployment)
	if err != nil {
		return rel, controllerutil.OperationResultNone, err
	}
	configmap := &k8score.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: k8score.SchemeGroupVersion.Identifier(), Kind: configMapKind},
		ObjectMeta: metav1.ObjectMeta{Namespace: deployment.Namespace, Name: configmapName},
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, configmap, func() error {
		if configmap.CreationTimestamp.IsZero() {
			r.Log.Info("Create prometheus configmap", "configmap", vznav.GetNamespacedNameFromObjectMeta(configmap.ObjectMeta))
		} else {
			r.Log.Info("Update prometheus configmap", "configmap", vznav.GetNamespacedNameFromObjectMeta(configmap.ObjectMeta))
		}
		yamlStr, exists := configmap.Data[prometheusConfigKey]
		if !exists {
			yamlStr = ""
		}
		prometheusConf, err := parseYAMLString(yamlStr)
		if err != nil {
			return err
		}
		prometheusConf, err = mutatePrometheusScrapeConfig(trait, traitDefaults, prometheusConf, secret)
		if err != nil {
			return err
		}
		yamlStr, err = writeYAMLString(prometheusConf)
		if err != nil {
			return err
		}
		if configmap.Data == nil {
			configmap.Data = map[string]string{}
		}
		configmap.Data[prometheusConfigKey] = yamlStr
		return nil
	})
	// If the prometheus configmap was updated then restart the premetheus pods.
	if res == controllerutil.OperationResultUpdated {
		return rel, res, r.restartPrometheusPods(ctx, deployment)
	}
	return rel, res, err
}

// fetchPrometheusDeploymentFromTrait fetches the prometheus deployment from information in the trait.
func (r *Reconciler) fetchPrometheusDeploymentFromTrait(ctx context.Context, trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec) (*k8sapps.Deployment, error) {
	scraperRef := trait.Spec.Scraper
	if scraperRef == nil {
		scraperRef = traitDefaults.Scraper
	}
	scraperName, err := vznav.ParseNamespacedNameFromQualifiedName(*scraperRef)
	if err != nil {
		return nil, err
	}
	deployment := &k8sapps.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Namespace: scraperName.Namespace, Name: scraperName.Name}, deployment)
	if err != nil {
		return nil, err
	}
	r.Log.Info("Found prometheus deployment", "deployment", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta))
	return deployment, nil
}

// findPrometheusScrapeConfigMapNameFromDeployment finds the prometheus configmap name from the prometheus deployment.
func (r *Reconciler) findPrometheusScrapeConfigMapNameFromDeployment(deployment *k8sapps.Deployment) (string, error) {
	volumes := deployment.Spec.Template.Spec.Volumes
	for _, volume := range volumes {
		if volume.Name == "config-volume" && volume.ConfigMap != nil && len(volume.ConfigMap.Name) > 0 {
			name := volume.ConfigMap.Name
			r.Log.Info("Found prometheus configmap name", "configmap", name)
			return name, nil
		}
	}
	return "", fmt.Errorf("failed to find prometheus configmap name from deployment %s", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta))
}

// restartPrometheusPods finds and restarts the pods associated with a prometheus deployment.
func (r *Reconciler) restartPrometheusPods(ctx context.Context, deployment *k8sapps.Deployment) error {
	replicaSets, err := vznav.FetchUnstructuredChildResourcesByAPIVersionKinds(ctx, r, r.Log, deployment.Namespace, deployment.UID, []v1alpha2.ChildResourceKind{{APIVersion: "apps/v1", Kind: "ReplicaSet"}})
	if err != nil {
		return err
	}
	for _, replicaSet := range replicaSets {
		r.Log.Info("Found prometheus replicaset", "replicaset", vznav.GetNamespacedNameFromUnstructured(replicaSet))
		pods, err := vznav.FetchUnstructuredChildResourcesByAPIVersionKinds(ctx, r, r.Log, replicaSet.GetNamespace(), replicaSet.GetUID(), []v1alpha2.ChildResourceKind{{APIVersion: "v1", Kind: "Pod"}})
		if err != nil {
			return err
		}
		for _, pod := range pods {
			r.Log.Info("Found prometheus pod", "pod", vznav.GetNamespacedNameFromUnstructured(pod))
			err = r.Delete(ctx, pod)
			if err != nil {
				return err
			}
			r.Log.Info("Deleted prometheus pod", "pod", vznav.GetNamespacedNameFromUnstructured(pod))
		}
	}
	return nil
}

// updateRelatedDeployment updates the labels and annotations of a related workload deployment.
// For example containerized workloads produce related deployments.
func (r *Reconciler) updateRelatedDeployment(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	r.Log.Info("Update workload deployment", "deployment", vznav.GetNamespacedNameFromUnstructured(child))
	ref := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}
	deployment := &k8sapps.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec.Template.ObjectMeta.Annotations = mutateAnnotations(trait, workload, traitDefaults, deployment.Spec.Template.ObjectMeta.Annotations)
		deployment.Spec.Template.ObjectMeta.Labels = mutateLabels(trait, workload, deployment.Spec.Template.ObjectMeta.Labels)
		return nil
	})
	if err != nil {
		r.Log.Error(err, "Failed to update workload deployment")
	}
	return ref, res, err
}

// updateRelatedStatefulSet updates the labels and annotations of a related workload stateful set.
// For example coherence workloads produce related stateful sets.
func (r *Reconciler) updateRelatedStatefulSet(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	r.Log.Info("Update workload stateful set", "statefulSet", vznav.GetNamespacedNameFromUnstructured(child))
	ref := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}
	statefulSet := &k8sapps.StatefulSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, statefulSet, func() error {
		statefulSet.Spec.Template.ObjectMeta.Annotations = mutateAnnotations(trait, workload, traitDefaults, statefulSet.Spec.Template.ObjectMeta.Annotations)
		statefulSet.Spec.Template.ObjectMeta.Labels = mutateLabels(trait, workload, statefulSet.Spec.Template.ObjectMeta.Labels)
		return nil
	})
	if err != nil {
		r.Log.Error(err, "Failed to update workload stateful set")
	}
	return ref, res, err
}

// updateRelatedPod updates the labels and annotations of a related workload pod.
// For example WLS workloads produce related pods.
func (r *Reconciler) updateRelatedPod(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	r.Log.Info("Update workload pod", "pod", vznav.GetNamespacedNameFromUnstructured(child))
	rel := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}
	pod := &k8score.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, pod, func() error {
		pod.ObjectMeta.Annotations = mutateAnnotations(trait, workload, traitDefaults, pod.ObjectMeta.Annotations)
		pod.ObjectMeta.Labels = mutateLabels(trait, workload, pod.ObjectMeta.Labels)
		return nil
	})
	if err != nil {
		r.Log.Error(err, "Failed to update workload pod")
	}
	return rel, res, err
}

// updateTraitStatus updates the trait's status conditions and resources if they have changed.
// The return value can be used as the result of the Reconcile method.
func (r *Reconciler) updateTraitStatus(ctx context.Context, trait *vzapi.MetricsTrait, results *reconcileresults.ReconcileResults) (reconcile.Result, error) {
	name := vznav.GetNamespacedNameFromObjectMeta(trait.ObjectMeta)

	// If the status content has changed persist the updated status.
	if trait.DeletionTimestamp.IsZero() && updateStatusIfRequired(&trait.Status, results) {
		err := r.Status().Update(ctx, trait)
		if err != nil {
			r.Log.Info("Failed to update metrics trait status", "trait", name)
			return reconcile.Result{}, err
		}
		r.Log.Info("Updated metrics trait status", "trait", name)
	}

	// If the results contained errors then requeue immediately.
	if results.ContainsErrors() {
		r.Log.Info("Failed to reconciled metrics trait", "trait", name)
		return reconcile.Result{Requeue: true}, nil
	}

	// If the status has not change and there are no errors
	// requeue with a jittered delay to account for situations where a workload
	// changes but without necessarily updating the trait spec.
	var seconds = rand.IntnRange(45, 90)
	var duration = time.Duration(seconds) * time.Second
	r.Log.Info("Successfully reconciled metrics trait", "trait", name)
	return reconcile.Result{Requeue: true, RequeueAfter: duration}, nil
}

// fetchWLSDomainCredentialsSecretName fetches the credentials from the WLS workload resource (i.e. domain).
// These credentials are used in the population of the prometheus scraper configuration.
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

// fetchCoherenceMetricsSpec fetches the metrics configuration from the Coherence workload resource spec.
// These configuration values are used in the population of the prometheus scraper configuration.
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

// fetchTraitDefaults fetches metrics trait default values.
// These default values are workload type dependent.
func (r *Reconciler) fetchTraitDefaults(ctx context.Context, workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil {
		return nil, err
	}
	// Match any version of APIVersion=weblogic.oracle and Kind=Domain
	if matched, _ := regexp.MatchString("^weblogic.oracle/.*\\.Domain$", apiVerKind); matched {
		return r.newTraitDefaultsForWLSDomainWorkload(ctx, workload)
	}
	// Match any version of APIVersion=coherence.oracle and Kind=Coherence
	if matched, _ := regexp.MatchString("^coherence.oracle.com/.*\\.Coherence$", apiVerKind); matched {
		return r.newTraitDefaultsForCOHWorkload(ctx, workload)
	}
	// Match any version of APIVersion=core.oam.dev and Kind=ContainerizedWorkload
	if matched, _ := regexp.MatchString("^core.oam.dev/.*\\.ContainerizedWorkload$", apiVerKind); matched {
		return r.newTraitDefaultsForOAMContainerizedWorkload()
	}
	gvk, _ := vznav.GetAPIVersionKindOfUnstructured(workload)
	return nil, fmt.Errorf("unsupported kind %s of workload %s", gvk, vznav.GetNamespacedNameFromUnstructured(workload))
}

// newTraitDefaultsForWLSDomainWorkload creates metrics trait default values for a WLS domain workload.
func (r *Reconciler) newTraitDefaultsForWLSDomainWorkload(ctx context.Context, workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
	// Port precedence: trait, workload annotation, default
	port := defaultScrapePort
	path := defaultWLSScrapePath
	secret, err := r.fetchWLSDomainCredentialsSecretName(ctx, workload)
	if err != nil {
		return nil, err
	}
	return &vzapi.MetricsTraitSpec{
		Port:    &port,
		Path:    &path,
		Secret:  secret,
		Scraper: &r.Scraper}, nil
}

// newTraitDefaultsForCOHWorkload creates metrics trait default values for a Coherence workload.
func (r *Reconciler) newTraitDefaultsForCOHWorkload(ctx context.Context, workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
	path := defaultScrapePath
	port := defaultCohScrapePort
	var secret *string = nil

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
		Port:    &port,
		Path:    &path,
		Secret:  secret,
		Scraper: &r.Scraper}, nil
}

// newTraitDefaultsForOAMContainerizedWorkload creates metrics trait default values for a containerized workload.
func (r *Reconciler) newTraitDefaultsForOAMContainerizedWorkload() (*vzapi.MetricsTraitSpec, error) {
	port := defaultScrapePort
	path := defaultScrapePath
	return &vzapi.MetricsTraitSpec{
		Port:    &port,
		Path:    &path,
		Secret:  nil,
		Scraper: &r.Scraper}, nil
}

// updateStatusIfRequired updates the traits status (i.e. resources and conditions) if they have changed.
// Returns a boolean indicating if status resources or conditions have been updated.
func updateStatusIfRequired(status *vzapi.MetricsTraitStatus, results *reconcileresults.ReconcileResults) bool {
	updated := false
	if !vzapi.QualifiedResourceRelationSlicesEquivalent(status.Resources, results.Relations) {
		status.Resources = results.Relations
		updated = true
	}
	conditionedStatus := results.CreateConditionedStatus()
	if !reconcileresults.ConditionedStatusEquivalent(&status.ConditionedStatus, &conditionedStatus) {
		status.ConditionedStatus = conditionedStatus
		updated = true
	}
	return updated
}

// mutatePrometheusScrapeConfig mutates the prometheus scrape configuration.
// Scrap configuration rules will be added, updated, deleted depending on the state of the trait.
func mutatePrometheusScrapeConfig(trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, prometheusScrapeConfig *gabs.Container, secret *k8score.Secret) (*gabs.Container, error) {
	oldScrapeConfigs := prometheusScrapeConfig.Search(prometheusScrapeConfigsLabel).Children()
	prometheusScrapeConfig.Array(prometheusScrapeConfigsLabel) // zero out the array of scrape configs
	newScrapeJob, newScrapeConfig, err := createScrapeConfigFromTrait(trait, traitDefaults, secret)
	if err != nil {
		return prometheusScrapeConfig, err
	}
	existingReplaced := false
	for _, oldScrapeConfig := range oldScrapeConfigs {
		oldScrapeJob := oldScrapeConfig.Search(prometheusJobNameLabel).Data()
		if newScrapeJob == oldScrapeJob {
			// If the scrape config should be removed then skip adding it to the result slice.
			// This will occur in two situations.
			// 1. The trait is being deleted.
			// 2. The trait scraper has been changed and the old scrape config is being updated.
			//    In this case the traitDefaults will be nil.
			if trait.DeletionTimestamp.IsZero() && traitDefaults != nil {
				prometheusScrapeConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
			}
			existingReplaced = true
		} else {
			prometheusScrapeConfig.ArrayAppendP(oldScrapeConfig.Data(), prometheusScrapeConfigsLabel)
		}
	}
	if !existingReplaced {
		prometheusScrapeConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
	}
	return prometheusScrapeConfig, nil
}

// mutateAnnotations mutates annotations with values used by the scraper config.
// Annotations are either set or removed depending on the state of the trait.
func mutateAnnotations(trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, annotations map[string]string) map[string]string {
	mutated := annotations

	// If the trait is being deleted, remove the annotations.
	if !trait.DeletionTimestamp.IsZero() {
		delete(mutated, verrazzanoMetricsEnabledAnnotation)
		delete(mutated, verrazzanoMetricsPathAnnotation)
		delete(mutated, verrazzanoMetricsPortAnnotation)
		return mutated
	}

	mutated = updateStringMap(mutated, verrazzanoMetricsEnabledAnnotation, strconv.FormatBool(true))

	// Merge trait, default and existing value.
	var found bool
	var path string
	if trait.Spec.Path != nil {
		path = *trait.Spec.Path
	} else {
		path, found = annotations[prometheusPathAnnotation]
		if !found {
			path = *traitDefaults.Path
		}
	}
	mutated = updateStringMap(mutated, verrazzanoMetricsPathAnnotation, path)

	// Merge trait, default and existing value.
	var port string
	if trait.Spec.Port != nil {
		port = strconv.Itoa(*trait.Spec.Port)
	} else {
		port, found = annotations[prometheusPortAnnotation]
		if !found {
			port = strconv.Itoa(*traitDefaults.Port)
		}
	}
	mutated = updateStringMap(mutated, verrazzanoMetricsPortAnnotation, port)

	return mutated
}

// mutateLabels mutates the labels associated with a related resources.
func mutateLabels(trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, labels map[string]string) map[string]string {
	mutated := labels
	// If the trait is not being deleted, copy specific labels from the trait.
	if trait.DeletionTimestamp.IsZero() {
		mutated = copyStringMapEntries(mutated, trait.Labels, appObjectMetaLabel, compObjectMetaLabel)
	}
	return mutated
}

// createPrometheusScrapeConfigMapJobName creates a prometheus scrape configmap job name from a trait.
// Format is {oam_app}_{cluster}_{namespace}_{oam_comp}
func createPrometheusScrapeConfigMapJobName(trait *vzapi.MetricsTrait) (string, error) {
	cluster := getClusterNameFromObjectMetaOrDefault(trait.ObjectMeta)
	namespace := getNamespaceFromObjectMetaOrDefault(trait.ObjectMeta)
	app, found := trait.Labels[appObjectMetaLabel]
	if !found {
		return "", fmt.Errorf("metrics trait missing application name label")
	}
	comp, found := trait.Labels[compObjectMetaLabel]
	if !found {
		return "", fmt.Errorf("metrics trait missing component name label")
	}
	return fmt.Sprintf("%s_%s_%s_%s", app, cluster, namespace, comp), nil
}

// createScrapeConfigFromTrait creates prometheus scrape config for a trait.
// This populates the prometheus scrape config template.
// The job name is returned.
// The YAML container populated from the prometheus scrape config template is returned.
func createScrapeConfigFromTrait(trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, secret *k8score.Secret) (string, *gabs.Container, error) {
	job, err := createPrometheusScrapeConfigMapJobName(trait)
	if err != nil {
		return "", nil, err
	}

	// Populate the prometheus scrape config template
	context := map[string]string{
		appNameHolder:   trait.Labels[appObjectMetaLabel],
		compNameHolder:  trait.Labels[compObjectMetaLabel],
		jobNameHolder:   job,
		namespaceHolder: trait.Namespace}

	// Populate the prometheus scrape config template
	template := mergeTemplateWithContext(prometheusScrapeConfigTemplate, context)

	// Parse the populate the prometheus scrape config template.
	config, err := parseYAMLString(template)
	if err != nil {
		return job, nil, fmt.Errorf("failed to parse built-in prometheus scrape config template: %w", err)
	}

	// Add basic auth credentials if provided
	if secret != nil {
		username, secretFound := secret.Data["username"]
		if secretFound {
			config.Set(string(username), basicAuthLabel, basicAuthUsernameLabel)
		}
		password, passwordFound := secret.Data["password"]
		if passwordFound {
			config.Set(string(password), basicAuthLabel, basicPathPasswordLabel)
		}
	}

	return job, config, nil
}
