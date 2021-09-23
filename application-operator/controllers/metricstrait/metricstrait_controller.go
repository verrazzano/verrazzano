// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"github.com/verrazzano/verrazzano/application-operator/controllers/reconcileresults"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// Kubernetes resource Kinds
	configMapKind   = "ConfigMap"
	deploymentKind  = "Deployment"
	statefulSetKind = "StatefulSet"
	podKind         = "Pod"

	// In code defaults for metrics trait configuration
	defaultWLSAdminScrapePort = 7001
	defaultCohScrapePort      = 9612
	defaultScrapePort         = 8080
	defaultScrapePath         = "/metrics"
	defaultWLSScrapePath      = "/wls-exporter/metrics"

	// The finalizer name used by this controller
	finalizerName = "metricstrait.finalizers.verrazzano.io"

	// Markers used during the processing of Prometheus scrape configurations
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

	// Template placeholders for the Prometheus scrape config template
	appNameHolder     = "##APP_NAME##"
	compNameHolder    = "##COMP_NAME##"
	jobNameHolder     = "##JOB_NAME##"
	namespaceHolder   = "##NAMESPACE##"
	sslProtocolHolder = "##SSL_PROTOCOL##"

	// Roles for use in qualified resource relations
	scraperRole = "scraper"
	sourceRole  = "source"

	// SSL protocol scrape parameters for Istio enabled MTLS components
	httpsProtocol = `scheme: https
tls_config:
  ca_file: /etc/istio-certs/root-cert.pem  
  cert_file: /etc/istio-certs/cert-chain.pem
  key_file: /etc/istio-certs/key.pem
  insecure_skip_verify: true  # Prometheus does not support Istio security naming, thus skip verifying target pod certificate`
	httpProtocol = "scheme: http"
)

// prometheusScrapeConfigTemplate configuration for general Prometheus scrape target template
// Used to add new scrape config to a Prometheus configmap
const prometheusScrapeConfigTemplate = `job_name: ##JOB_NAME##
##SSL_PROTOCOL##
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

// prometheusWLSScrapeConfigTemplate configuration for WebLogic Prometheus scrape target template
// Used to add new WebLogic scrape config to a Prometheus configmap
const prometheusWLSScrapeConfigTemplate = `job_name: ##JOB_NAME##
##SSL_PROTOCOL##
kubernetes_sd_configs:
- role: pod
  namespaces:
    names:
    - ##NAMESPACE##
relabel_configs:
- action: keep
  source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape,__meta_kubernetes_pod_label_app_oam_dev_name,__meta_kubernetes_pod_label_app_oam_dev_component]
  regex: true;##APP_NAME##;##COMP_NAME##
- action: replace
  source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
  target_label: __metrics_path__
  regex: (.+)
- action: replace
  source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
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
	r.Log.V(1).Info("Reconcile metrics trait", "trait", req.NamespacedName)
	ctx := context.Background()
	var err error

	// Fetch the trait.
	var trait *vzapi.MetricsTrait
	if trait, err = vznav.FetchTrait(ctx, r, r.Log, req.NamespacedName); err != nil || trait == nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if trait.DeletionTimestamp.IsZero() {
		result, supported, err := r.reconcileTraitCreateOrUpdate(ctx, trait)
		if err != nil {
			return result, err
		}
		if !supported {
			// If the workload kind is not supported then delete the trait
			r.Log.V(1).Info(fmt.Sprintf("deleting trait %s because workload is not supported", trait.Name))
			err = r.Client.Delete(context.TODO(), trait, &client.DeleteOptions{})
		}
		return result, err
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
func (r *Reconciler) reconcileTraitCreateOrUpdate(ctx context.Context, trait *vzapi.MetricsTrait) (ctrl.Result, bool, error) {
	var err error

	// Add finalizer if required.
	if err = r.addFinalizerIfRequired(ctx, trait); err != nil {
		return reconcile.Result{}, true, err
	}

	// Fetch workload resource using information from the trait
	var workload *unstructured.Unstructured
	if workload, err = vznav.FetchWorkloadFromTrait(ctx, r, r.Log, trait); err != nil {
		return reconcile.Result{}, true, err
	}

	// Resolve trait defaults from the trait and the workload.
	var traitDefaults *vzapi.MetricsTraitSpec
	var supported bool
	traitDefaults, supported, err = r.fetchTraitDefaults(ctx, workload)
	if err != nil {
		return reconcile.Result{}, supported, err
	}
	if traitDefaults == nil || !supported {
		return reconcile.Result{}, supported, nil
	}

	var scraper *k8sapps.Deployment
	if scraper, err = r.fetchPrometheusDeploymentFromTrait(ctx, trait, traitDefaults); err != nil {
		return reconcile.Result{}, true, err
	}

	// Find the child resources of the workload based on the childResourceKinds from the
	// workload definition, workload uid and the ownerReferences of the children.
	var children []*unstructured.Unstructured
	if children, err = vznav.FetchWorkloadChildren(ctx, r, r.Log, workload); err != nil {
		return reconcile.Result{}, true, err
	}

	// Create or update the related resources of the trait and collect the outcomes.
	status := r.createOrUpdateRelatedResources(ctx, trait, workload, traitDefaults, scraper, children)
	// Delete or update any previously (but no longer) related resources of the trait.
	status = r.deleteOrUpdateObsoleteResources(ctx, trait, status)

	// Update the status of the trait resource using the outcomes of the create or update.
	traitStatus, err := r.updateTraitStatus(ctx, trait, status)
	return traitStatus, true, err
}

// addFinalizerIfRequired adds the finalizer to the trait if required
// The finalizer is only added if the trait is not being deleted and the finalizer has not previously been added
func (r *Reconciler) addFinalizerIfRequired(ctx context.Context, trait *vzapi.MetricsTrait) error {
	if trait.GetDeletionTimestamp().IsZero() && !vzstring.SliceContainsString(trait.Finalizers, finalizerName) {
		traitName := vznav.GetNamespacedNameFromObjectMeta(trait.ObjectMeta).String()
		r.Log.V(1).Info("Adding finalizer from trait", "trait", traitName)
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
	if !trait.DeletionTimestamp.IsZero() && vzstring.SliceContainsString(trait.Finalizers, finalizerName) {
		traitName := vznav.GetNamespacedNameFromObjectMeta(trait.ObjectMeta).String()
		r.Log.Info("Removing finalizer from trait", "trait", traitName)
		trait.Finalizers = vzstring.RemoveStringFromSlice(trait.Finalizers, finalizerName)
		if err := r.Update(ctx, trait); err != nil {
			r.Log.Error(err, "failed to remove finalizer to trait", "trait", traitName)
			return err
		}
	}
	return nil
}

// createOrUpdateRelatedResources creates or updates resources related to this trait
// The related resources are the workload children and the Prometheus config
func (r *Reconciler) createOrUpdateRelatedResources(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, deployment *k8sapps.Deployment, children []*unstructured.Unstructured) *reconcileresults.ReconcileResults {
	status := reconcileresults.ReconcileResults{}
	for _, child := range children {
		switch child.GroupVersionKind() {
		case k8sapps.SchemeGroupVersion.WithKind(deploymentKind):
			// In the case of VerrazzanoHelidonWorkload, it isn't unwrapped so we need to check to see
			// if the workload is a wrapper kind in addition to checking to see if the owner is a wrapper kind.
			// In the case of a wrapper kind or owner, the status is not being updated here as this is handled by the
			// wrapper owner which is the corresponding Verrazzano wrapper resource/controller.
			if !vznav.IsOwnedByVerrazzanoWorkloadKind(workload) && !vznav.IsVerrazzanoWorkloadKind(workload) {
				status.RecordOutcome(r.updateRelatedDeployment(ctx, trait, workload, traitDefaults, child))
			}
		case k8sapps.SchemeGroupVersion.WithKind(statefulSetKind):
			// In the case of a workload having an owner that is a wrapper kind, the status is not being updated here
			// as this is handled by the wrapper owner which is the corresponding Verrazzano wrapper resource/controller.
			if !vznav.IsOwnedByVerrazzanoWorkloadKind(workload) {
				status.RecordOutcome(r.updateRelatedStatefulSet(ctx, trait, workload, traitDefaults, child))
			}
		case k8score.SchemeGroupVersion.WithKind(podKind):
			// In the case of a workload having an owner that is a wrapper kind, the status is not being updated here
			// as this is handled by the wrapper owner which is the corresponding Verrazzano wrapper resource/controller.
			if !vznav.IsOwnedByVerrazzanoWorkloadKind(workload) {
				status.RecordOutcome(r.updateRelatedPod(ctx, trait, workload, traitDefaults, child))
			}
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
				// Don't record an outcome for unknown role relations.
				r.Log.Info("Skip delete or update of unknown resource role", "role", rel.Role)
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
		// Return a NotFoundError to cause removal the resource relation from the status.
		r.Log.Info("Skip delete or update of metrics source of unknown kind", "kind", rel.Kind)
		return rel, controllerutil.OperationResultNone, apierrors.NewNotFound(schema.GroupResource{Group: rel.APIVersion, Resource: rel.Kind}, rel.Name)
	}
}

// deleteOrUpdateScraperConfigMap cleans up a scraper (i.e. Prometheus) configmap.
// The scraper config for the trait is removed if present.
func (r *Reconciler) deleteOrUpdateScraperConfigMap(ctx context.Context, trait *vzapi.MetricsTrait, rel vzapi.QualifiedResourceRelation) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	deployment := &k8sapps.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Namespace: rel.Namespace, Name: rel.Name}, deployment)
	if err != nil {
		return rel, controllerutil.OperationResultNone, client.IgnoreNotFound(err)
	}
	return r.updatePrometheusScraperConfigMap(ctx, trait, nil, nil, deployment)
}

// updatePrometheusScraperConfigMap updates the Prometheus scraper configmap.
// This updates only the scrape_configs section of the Prometheus configmap.
// Only the rules for the provided trait will be affected.
// trait - The trait to update scrape_config rules for.
// traitDefaults - Default to use for values not provided in the trait.
// deployment - The Prometheus deployment.
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
			r.Log.V(1).Info("Create Prometheus configmap", "configmap", vznav.GetNamespacedNameFromObjectMeta(configmap.ObjectMeta))
		} else {
			r.Log.V(1).Info("Update Prometheus configmap", "configmap", vznav.GetNamespacedNameFromObjectMeta(configmap.ObjectMeta))
		}
		yamlStr, exists := configmap.Data[prometheusConfigKey]
		if !exists {
			yamlStr = ""
		}
		prometheusConf, err := parseYAMLString(yamlStr)
		if err != nil {
			return err
		}
		prometheusConf, err = mutatePrometheusScrapeConfig(ctx, trait, traitDefaults, prometheusConf, secret, workload, r.Client)
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
	// If the Prometheus configmap was updated, the VMI Prometheus has ConfigReloader sidecar to signal Prometheus to reload config
	if res == controllerutil.OperationResultUpdated {
		return rel, res, nil
	}
	return rel, res, err
}

// fetchPrometheusDeploymentFromTrait fetches the Prometheus deployment from information in the trait.
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
	r.Log.V(1).Info("Found Prometheus deployment", "deployment", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta))
	return deployment, nil
}

// findPrometheusScrapeConfigMapNameFromDeployment finds the Prometheus configmap name from the Prometheus deployment.
func (r *Reconciler) findPrometheusScrapeConfigMapNameFromDeployment(deployment *k8sapps.Deployment) (string, error) {
	volumes := deployment.Spec.Template.Spec.Volumes
	for _, volume := range volumes {
		if volume.Name == "config-volume" && volume.ConfigMap != nil && len(volume.ConfigMap.Name) > 0 {
			name := volume.ConfigMap.Name
			r.Log.V(1).Info("Found Prometheus configmap name", "configmap", name)
			return name, nil
		}
	}
	return "", fmt.Errorf("failed to find Prometheus configmap name from deployment %s", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta))
}

// updateRelatedDeployment updates the labels and annotations of a related workload deployment.
// For example containerized workloads produce related deployments.
func (r *Reconciler) updateRelatedDeployment(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	r.Log.V(1).Info("Update workload deployment", "deployment", vznav.GetNamespacedNameFromUnstructured(child))
	ref := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}
	deployment := &k8sapps.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// If the deployment was not found don't attempt to create or update it.
		if deployment.CreationTimestamp.IsZero() {
			r.Log.Info("Workload child deployment not found")
			return apierrors.NewNotFound(schema.GroupResource{Group: deployment.APIVersion, Resource: deployment.Kind}, deployment.Name)
		}
		deployment.Spec.Template.ObjectMeta.Annotations = MutateAnnotations(trait, workload, traitDefaults, deployment.Spec.Template.ObjectMeta.Annotations)
		deployment.Spec.Template.ObjectMeta.Labels = MutateLabels(trait, workload, deployment.Spec.Template.ObjectMeta.Labels)
		return nil
	})
	if err != nil && !apierrors.IsNotFound(err) {
		r.Log.Info("Failed to update workload child deployment", "child", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta), "error", err)
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
		// If the statefulset was not found don't attempt to create or update it.
		if statefulSet.CreationTimestamp.IsZero() {
			r.Log.Info("Workload child statefulset not found")
			return apierrors.NewNotFound(schema.GroupResource{Group: statefulSet.APIVersion, Resource: statefulSet.Kind}, statefulSet.Name)
		}
		statefulSet.Spec.Template.ObjectMeta.Annotations = MutateAnnotations(trait, workload, traitDefaults, statefulSet.Spec.Template.ObjectMeta.Annotations)
		statefulSet.Spec.Template.ObjectMeta.Labels = MutateLabels(trait, workload, statefulSet.Spec.Template.ObjectMeta.Labels)
		return nil
	})
	if err != nil && !apierrors.IsNotFound(err) {
		r.Log.Info("Failed to update workload child statefulset", "child", vznav.GetNamespacedNameFromObjectMeta(statefulSet.ObjectMeta), "error", err)
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
		// If the pod was not found don't attempt to create or update it.
		if pod.CreationTimestamp.IsZero() {
			r.Log.Info("Workload child pod not found")
			return apierrors.NewNotFound(schema.GroupResource{Group: pod.APIVersion, Resource: pod.Kind}, pod.Name)
		}
		pod.ObjectMeta.Annotations = MutateAnnotations(trait, workload, traitDefaults, pod.ObjectMeta.Annotations)
		pod.ObjectMeta.Labels = MutateLabels(trait, workload, pod.ObjectMeta.Labels)
		return nil
	})
	if err != nil && !apierrors.IsNotFound(err) {
		r.Log.Info("Failed to update workload child pod", "pod", vznav.GetNamespacedNameFromObjectMeta(pod.ObjectMeta), "error", err)
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
	r.Log.V(1).Info("Successfully reconciled metrics trait", "trait", name)
	return reconcile.Result{Requeue: true, RequeueAfter: duration}, nil
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
func (r *Reconciler) fetchTraitDefaults(ctx context.Context, workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, bool, error) {
	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil {
		return nil, true, err
	}
	// Match any version of Group=weblogic.oracle and Kind=Domain
	if matched, _ := regexp.MatchString("^weblogic.oracle/.*\\.Domain$", apiVerKind); matched {
		spec, err := r.NewTraitDefaultsForWLSDomainWorkload(ctx, workload)
		return spec, true, err
	}
	// Match any version of Group=coherence.oracle and Kind=Coherence
	if matched, _ := regexp.MatchString("^coherence.oracle.com/.*\\.Coherence$", apiVerKind); matched {
		spec, err := r.NewTraitDefaultsForCOHWorkload(ctx, workload)
		return spec, true, err
	}

	// Match any version of Group=coherence.oracle and Kind=VerrazzanoHelidonWorkload
	// In the case of Helidon, the workload isn't currently being unwrapped
	if matched, _ := regexp.MatchString("^oam.verrazzano.io/.*\\.VerrazzanoHelidonWorkload$", apiVerKind); matched {
		spec, err := r.NewTraitDefaultsForGenericWorkload()
		return spec, true, err
	}

	// Match any version of Group=core.oam.dev and Kind=ContainerizedWorkload
	if matched, _ := regexp.MatchString("^core.oam.dev/.*\\.ContainerizedWorkload$", apiVerKind); matched {
		spec, err := r.NewTraitDefaultsForGenericWorkload()
		return spec, true, err
	}

	// Match any version of Group=apps and Kind=Deployment
	if matched, _ := regexp.MatchString("^apps/.*\\.Deployment$", apiVerKind); matched {
		spec, err := r.NewTraitDefaultsForGenericWorkload()
		return spec, true, err
	}

	// Log the kind/workload is unsupported and return a nil trait.
	gvk, _ := vznav.GetAPIVersionKindOfUnstructured(workload)
	r.Log.V(1).Info(fmt.Sprintf("unsupported kind %s of workload %s", gvk, vznav.GetNamespacedNameFromUnstructured(workload)))
	return nil, false, nil
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
		Port:    &port,
		Path:    &path,
		Secret:  secret,
		Scraper: &r.Scraper}, nil
}

// NewTraitDefaultsForCOHWorkload creates metrics trait default values for a Coherence workload.
func (r *Reconciler) NewTraitDefaultsForCOHWorkload(ctx context.Context, workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
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

// NewTraitDefaultsForGenericWorkload creates metrics trait default values for a containerized workload.
func (r *Reconciler) NewTraitDefaultsForGenericWorkload() (*vzapi.MetricsTraitSpec, error) {
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

// mutatePrometheusScrapeConfig mutates the Prometheus scrape configuration.
// Scrap configuration rules will be added, updated, deleted depending on the state of the trait.
func mutatePrometheusScrapeConfig(ctx context.Context, trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, prometheusScrapeConfig *gabs.Container, secret *k8score.Secret, workload *unstructured.Unstructured, c client.Client) (*gabs.Container, error) {
	oldScrapeConfigs := prometheusScrapeConfig.Search(prometheusScrapeConfigsLabel).Children()
	prometheusScrapeConfig.Array(prometheusScrapeConfigsLabel) // zero out the array of scrape configs
	newScrapeJob, newScrapeConfig, err := createScrapeConfigFromTrait(ctx, trait, traitDefaults, secret, workload, c)
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
			//    In this case the traitDefaults and newScrapeConfig will be nil.
			if trait.DeletionTimestamp.IsZero() && traitDefaults != nil && newScrapeConfig != nil {
				prometheusScrapeConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
			}
			existingReplaced = true
		} else {
			prometheusScrapeConfig.ArrayAppendP(oldScrapeConfig.Data(), prometheusScrapeConfigsLabel)
		}
	}
	// If an existing config was not replaced and there is new config (i.e. newScrapeConfig != nil) then add the new config.
	if !existingReplaced && newScrapeConfig != nil {
		prometheusScrapeConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
	}
	return prometheusScrapeConfig, nil
}

// MutateAnnotations mutates annotations with values used by the scraper config.
// Annotations are either set or removed depending on the state of the trait.
func MutateAnnotations(trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, annotations map[string]string) map[string]string {
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

// MutateLabels mutates the labels associated with a related resources.
func MutateLabels(trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, labels map[string]string) map[string]string {
	mutated := labels
	// If the trait is not being deleted, copy specific labels from the trait.
	if trait.DeletionTimestamp.IsZero() {
		mutated = copyStringMapEntries(mutated, trait.Labels, appObjectMetaLabel, compObjectMetaLabel)
	}
	return mutated
}

// useHTTPSForScrapeTarget returns true if https with Istio certs should be used for scrape target. Otherwise return false, use http
func useHTTPSForScrapeTarget(ctx context.Context, c client.Client, trait *vzapi.MetricsTrait) (bool, error) {
	if trait.Spec.WorkloadReference.Kind == "VerrazzanoCoherenceWorkload" || trait.Spec.WorkloadReference.Kind == "Coherence" {
		return false, nil
	}
	// Get the namespace resource that the MetricsTrait is deployed to
	namespace := &k8score.Namespace{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: trait.Namespace}, namespace); err != nil {
		return false, err
	}
	istioInjection, hasIstioInjection := namespace.Labels["istio-injection"]
	_, hasIstioRevision := namespace.Labels["istio.io/rev"]
	hasSidecar := (hasIstioInjection && istioInjection == "enabled") || hasIstioRevision
	if hasSidecar {
		return true, nil
	}
	return false, nil
}

// createPrometheusScrapeConfigMapJobName creates a Prometheus scrape configmap job name from a trait.
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

// createScrapeConfigFromTrait creates Prometheus scrape config for a trait.
// This populates the Prometheus scrape config template.
// The job name is returned.
// The YAML container populated from the Prometheus scrape config template is returned.
func createScrapeConfigFromTrait(ctx context.Context, trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, secret *k8score.Secret, workload *unstructured.Unstructured, c client.Client) (string, *gabs.Container, error) {

	job, err := createPrometheusScrapeConfigMapJobName(trait)
	if err != nil {
		return "", nil, err
	}

	// If workload is nil then the trait is being deleted so no config is required
	if workload != nil {
		// Populate the Prometheus scrape config template
		context := map[string]string{
			appNameHolder:     trait.Labels[appObjectMetaLabel],
			compNameHolder:    trait.Labels[compObjectMetaLabel],
			jobNameHolder:     job,
			namespaceHolder:   trait.Namespace,
			sslProtocolHolder: httpProtocol}

		var configTemplate string
		https, err := useHTTPSForScrapeTarget(ctx, c, trait)
		if err != nil {
			return "", nil, err
		}

		if https {
			context[sslProtocolHolder] = httpsProtocol
		}
		configTemplate = prometheusScrapeConfigTemplate
		apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
		if err != nil {
			return "", nil, err
		}
		// Match any version of APIVersion=weblogic.oracle and Kind=Domain
		if matched, _ := regexp.MatchString("^weblogic.oracle/.*\\.Domain$", apiVerKind); matched {
			configTemplate = prometheusWLSScrapeConfigTemplate
		}

		// Populate the Prometheus scrape config template
		template := mergeTemplateWithContext(configTemplate, context)

		// Parse the populate the Prometheus scrape config template.
		config, err := parseYAMLString(template)
		if err != nil {
			return job, nil, fmt.Errorf("failed to parse built-in Prometheus scrape config template: %w", err)
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

	// If the trait is being deleted (i.e. workload==nil) then no config is required.
	return job, nil, nil
}
