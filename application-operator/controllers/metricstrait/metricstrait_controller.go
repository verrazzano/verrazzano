// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"github.com/verrazzano/verrazzano/application-operator/controllers/reconcileresults"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	vzlog2 "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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
	prometheusClusterNameLabel   = "verrazzano_cluster"

	// Annotation names for metrics read by the controller
	prometheusPortAnnotation = "prometheus.io/port"
	prometheusPathAnnotation = "prometheus.io/path"

	// Annotation names for metrics set by the controller
	verrazzanoMetricsAnnotationPrefix  = "verrazzano.io/metrics"
	verrazzanoMetricsPortAnnotation    = "verrazzano.io/metricsPort%s"
	verrazzanoMetricsPathAnnotation    = "verrazzano.io/metricsPath%s"
	verrazzanoMetricsEnabledAnnotation = "verrazzano.io/metricsEnabled%s"

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
	appNameHolder       = "##APP_NAME##"
	compNameHolder      = "##COMP_NAME##"
	jobNameHolder       = "##JOB_NAME##"
	portOrderHolder     = "##PORT_ORDER##"
	namespaceHolder     = "##NAMESPACE##"
	sslProtocolHolder   = "##SSL_PROTOCOL##"
	vzClusterNameHolder = "##VERRAZZANO_CLUSTER_NAME##"

	// Roles for use in qualified resource relations
	scraperRole = "scraper"
	sourceRole  = "source"
	ownerRole   = "owner"

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
- action: replace
  source_labels: null
  target_label: ` + prometheusClusterNameLabel + `
  replacement: ##VERRAZZANO_CLUSTER_NAME##
- action: keep
  source_labels: [__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled##PORT_ORDER##,__meta_kubernetes_pod_label_app_oam_dev_name,__meta_kubernetes_pod_label_app_oam_dev_component]
  regex: true;##APP_NAME##;##COMP_NAME##
- action: replace
  source_labels: [__meta_kubernetes_pod_annotation_verrazzano_io_metricsPath##PORT_ORDER##]
  target_label: __metrics_path__
  regex: (.+)
- action: replace
  source_labels: [__address__, __meta_kubernetes_pod_annotation_verrazzano_io_metricsPort##PORT_ORDER##]
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
- action: replace
  source_labels: null
  target_label: ` + prometheusClusterNameLabel + `
  replacement: ##VERRAZZANO_CLUSTER_NAME##
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
	Log     *zap.SugaredLogger
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
	ctx := context.Background()
	// Fetch the trait.
	var err error
	var trait *vzapi.MetricsTrait
	if trait, err = vznav.FetchTrait(ctx, r, zap.S(), req.NamespacedName); err != nil || trait == nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}

	log, err := clusters.GetResourceLogger("metricstrait", req.NamespacedName, trait)
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for metrics trait", err)
		return clusters.NewRequeueWithDelay(), nil
	}
	log.Oncef("Reconciling metrics trait resource %v, generation %v", req.NamespacedName, trait.Generation)

	res, err := r.doReconcile(ctx, trait, log)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	if err != nil {
		return clusters.NewRequeueWithDelay(), err
	}

	log.Oncef("Finished reconciling metrics trait %v", req.NamespacedName)

	return ctrl.Result{}, nil
}

// doReconcile performs the reconciliation operations for the metrics trait
func (r *Reconciler) doReconcile(ctx context.Context, trait *vzapi.MetricsTrait, log vzlog2.VerrazzanoLogger) (ctrl.Result, error) {
	if trait.DeletionTimestamp.IsZero() {
		result, supported, err := r.reconcileTraitCreateOrUpdate(ctx, trait, log)
		if err != nil {
			return result, err
		}
		if !supported {
			// If the workload kind is not supported then delete the trait
			log.Debugf("Deleting trait %s because workload is not supported", trait.Name)
			err = r.Client.Delete(context.TODO(), trait, &client.DeleteOptions{})
		}
		return result, err
	}
	return r.reconcileTraitDelete(ctx, trait, log)
}

// reconcileTraitDelete reconciles a metrics trait that is being deleted.
func (r *Reconciler) reconcileTraitDelete(ctx context.Context, trait *vzapi.MetricsTrait, log vzlog2.VerrazzanoLogger) (ctrl.Result, error) {
	status := r.deleteOrUpdateObsoleteResources(ctx, trait, &reconcileresults.ReconcileResults{}, log)
	// Only remove the finalizer if all related resources were successfully updated.
	if !status.ContainsErrors() {
		r.removeFinalizerIfRequired(ctx, trait, log)
	}
	return r.updateTraitStatus(ctx, trait, status, log)
}

// reconcileTraitCreateOrUpdate reconciles a metrics trait that is being created or updated.
func (r *Reconciler) reconcileTraitCreateOrUpdate(ctx context.Context, trait *vzapi.MetricsTrait, log vzlog2.VerrazzanoLogger) (ctrl.Result, bool, error) {
	var err error

	// Add finalizer if required.
	if err = r.addFinalizerIfRequired(ctx, trait, log); err != nil {
		return reconcile.Result{}, true, err
	}

	// Fetch workload resource using information from the trait
	var workload *unstructured.Unstructured
	if workload, err = vznav.FetchWorkloadFromTrait(ctx, r, log, trait); err != nil || workload == nil {
		return reconcile.Result{}, true, err
	}

	// Resolve trait defaults from the trait and the workload.
	var traitDefaults *vzapi.MetricsTraitSpec
	var supported bool
	traitDefaults, supported, err = r.fetchTraitDefaults(ctx, workload, log)
	if err != nil {
		return reconcile.Result{}, supported, err
	}
	if traitDefaults == nil || !supported {
		return reconcile.Result{Requeue: false}, supported, nil
	}

	var scraper *k8sapps.Deployment
	if scraper, err = r.fetchPrometheusDeploymentFromTrait(ctx, trait, traitDefaults, log); err != nil {
		return reconcile.Result{}, true, err
	}

	// Find the child resources of the workload based on the childResourceKinds from the
	// workload definition, workload uid and the ownerReferences of the children.
	var children []*unstructured.Unstructured
	if children, err = vznav.FetchWorkloadChildren(ctx, r, log, workload); err != nil {
		return reconcile.Result{}, true, err
	}

	// Create or update the related resources of the trait and collect the outcomes.
	status := r.createOrUpdateRelatedResources(ctx, trait, workload, traitDefaults, scraper, children, log)
	// Delete or update any previously (but no longer) related resources of the trait.
	status = r.deleteOrUpdateObsoleteResources(ctx, trait, status, log)

	// Update the status of the trait resource using the outcomes of the create or update.
	traitStatus, err := r.updateTraitStatus(ctx, trait, status, log)
	return traitStatus, true, err
}

// addFinalizerIfRequired adds the finalizer to the trait if required
// The finalizer is only added if the trait is not being deleted and the finalizer has not previously been added
func (r *Reconciler) addFinalizerIfRequired(ctx context.Context, trait *vzapi.MetricsTrait, log vzlog2.VerrazzanoLogger) error {
	if trait.GetDeletionTimestamp().IsZero() && !vzstring.SliceContainsString(trait.Finalizers, finalizerName) {
		traitName := vznav.GetNamespacedNameFromObjectMeta(trait.ObjectMeta)
		log.Debugf("Adding finalizer from trait %s", traitName)
		trait.Finalizers = append(trait.Finalizers, finalizerName)
		if err := r.Update(ctx, trait); err != nil {
			log.Errorf("Failed to add finalizer to trait %s: %v", traitName, err)
			return err
		}
	}
	return nil
}

// removeFinalizerIfRequired removes the finalizer from the trait if required
// The finalizer is only removed if the trait is being deleted and the finalizer had been added
func (r *Reconciler) removeFinalizerIfRequired(ctx context.Context, trait *vzapi.MetricsTrait, log vzlog2.VerrazzanoLogger) error {
	if !trait.DeletionTimestamp.IsZero() && vzstring.SliceContainsString(trait.Finalizers, finalizerName) {
		traitName := vznav.GetNamespacedNameFromObjectMeta(trait.ObjectMeta)
		log.Debugf("Removing finalizer from trait %s", traitName)
		trait.Finalizers = vzstring.RemoveStringFromSlice(trait.Finalizers, finalizerName)
		if err := r.Update(ctx, trait); err != nil {
			log.Errorf("Failed to remove finalizer to trait %s: %v", traitName, err)
			return err
		}
	}
	return nil
}

// createOrUpdateRelatedResources creates or updates resources related to this trait
// The related resources are the workload children and the Prometheus config
func (r *Reconciler) createOrUpdateRelatedResources(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, deployment *k8sapps.Deployment, children []*unstructured.Unstructured, log vzlog2.VerrazzanoLogger) *reconcileresults.ReconcileResults {
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
	status.RecordOutcome(r.updatePrometheusScraperConfigMap(ctx, trait, workload, traitDefaults, deployment, log))
	return &status
}

// deleteOrUpdateObsoleteResources deletes or updates resources that should no longer be related to this trait.
// This includes previous scrapers when the scraper has changed.
// This also includes previous workload children that are no longer referenced.
func (r *Reconciler) deleteOrUpdateObsoleteResources(ctx context.Context, trait *vzapi.MetricsTrait, status *reconcileresults.ReconcileResults, log vzlog2.VerrazzanoLogger) *reconcileresults.ReconcileResults {
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
				update.RecordOutcomeIfError(r.deleteOrUpdateScraperConfigMap(ctx, trait, rel, log)) // Need to pass down traitDefaults, current scraper or current scraper deployment
			case sourceRole:
				update.RecordOutcomeIfError(r.deleteOrUpdateMetricSourceResource(ctx, trait, rel, log))
			default:
				// Don't record an outcome for unknown role relations.
				log.Debugf("Skip delete or update of unknown resource role %s", rel.Role)
			}
		}
	}
	// Copy the reconcile outcomes from the current reconcile.
	for i, rel := range status.Relations {
		if !update.ContainsRelation(rel) {
			update.RecordOutcome(status.Relations[i], status.Results[i], status.Errors[i])
		}
	}

	if !trait.DeletionTimestamp.IsZero() && trait.OwnerReferences != nil {
		for i := range trait.OwnerReferences {
			if trait.OwnerReferences[i].Kind == "ApplicationConfiguration" {
				update.RecordOutcome(r.removedTraitReferencesFromOwner(ctx, &trait.OwnerReferences[i], trait, log))
			}
		}
	}

	return &update
}

// deleteOrUpdateMetricSourceResource deletes or updates the related resources that are the source of metrics.
// These are the children of the workloads.  For example for containerized workloads these are deployments.
// For WLS workloads these are pods.
func (r *Reconciler) deleteOrUpdateMetricSourceResource(ctx context.Context, trait *vzapi.MetricsTrait, rel vzapi.QualifiedResourceRelation, log vzlog2.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	child := unstructured.Unstructured{}
	child.SetAPIVersion(rel.APIVersion)
	child.SetKind(rel.Kind)
	child.SetNamespace(rel.Namespace)
	child.SetName(rel.Name)
	switch rel.Kind {
	case "Deployment":
		return r.updateRelatedDeployment(ctx, trait, nil, nil, &child, log)
	case "StatefulSet":
		return r.updateRelatedStatefulSet(ctx, trait, nil, nil, &child, log)
	case "Pod":
		return r.updateRelatedPod(ctx, trait, nil, nil, &child, log)
	default:
		// Return a NotFoundError to cause removal the resource relation from the status.
		log.Debugf("Skip delete or update of metrics source of unknown kind %s", rel.Kind)
		return rel, controllerutil.OperationResultNone, apierrors.NewNotFound(schema.GroupResource{Group: rel.APIVersion, Resource: rel.Kind}, rel.Name)
	}
}

// deleteOrUpdateScraperConfigMap cleans up a scraper (i.e. Prometheus) configmap.
// The scraper config for the trait is removed if present.
func (r *Reconciler) deleteOrUpdateScraperConfigMap(ctx context.Context, trait *vzapi.MetricsTrait, rel vzapi.QualifiedResourceRelation, log vzlog2.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	deployment := &k8sapps.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Namespace: rel.Namespace, Name: rel.Name}, deployment)
	if err != nil {
		return rel, controllerutil.OperationResultNone, client.IgnoreNotFound(err)
	}
	return r.updatePrometheusScraperConfigMap(ctx, trait, nil, nil, deployment, log)
}

// updatePrometheusScraperConfigMap updates the Prometheus scraper configmap.
// This updates only the scrape_configs section of the Prometheus configmap.
// Only the rules for the provided trait will be affected.
// trait - The trait to update scrape_config rules for.
// traitDefaults - Default to use for values not provided in the trait.
// deployment - The Prometheus deployment.
func (r *Reconciler) updatePrometheusScraperConfigMap(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, deployment *k8sapps.Deployment, log vzlog2.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	rel := vzapi.QualifiedResourceRelation{APIVersion: deployment.APIVersion, Kind: deployment.Kind, Name: deployment.Name, Namespace: deployment.Namespace, Role: scraperRole}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := r.fetchSourceCredentialsSecretIfRequired(ctx, trait, traitDefaults, workload)
	if err != nil {
		return rel, controllerutil.OperationResultNone, err
	}

	configmapName, err := r.findPrometheusScrapeConfigMapNameFromDeployment(deployment, log)
	if err != nil {
		return rel, controllerutil.OperationResultNone, err
	}

	configmap := &k8score.ConfigMap{}
	err = r.Get(ctx, client.ObjectKey{Namespace: deployment.Namespace, Name: configmapName}, configmap)
	if err != nil {
		// Don't create the config map if it doesn't already exist - that is the sole responsibility of
		// the Verrazzano Monitoring Operator
		return rel, controllerutil.OperationResultNone, client.IgnoreNotFound(err)
	}

	existingConfigmap := configmap.DeepCopyObject()

	if configmap.CreationTimestamp.IsZero() {
		log.Debugf("Create Prometheus configmap %s", vznav.GetNamespacedNameFromObjectMeta(configmap.ObjectMeta))
	} else {
		log.Debugf("Update Prometheus configmap %s", vznav.GetNamespacedNameFromObjectMeta(configmap.ObjectMeta))
	}
	yamlStr, exists := configmap.Data[prometheusConfigKey]
	if !exists {
		yamlStr = ""
	}
	prometheusConf, err := parseYAMLString(yamlStr)
	if err != nil {
		return rel, controllerutil.OperationResultNone, err
	}
	prometheusConf, err = mutatePrometheusScrapeConfig(ctx, trait, traitDefaults, prometheusConf, secret, workload, r.Client)
	if err != nil {
		return rel, controllerutil.OperationResultNone, err
	}
	yamlStr, err = writeYAMLString(prometheusConf)
	if err != nil {
		return rel, controllerutil.OperationResultNone, err
	}
	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}
	configmap.Data[prometheusConfigKey] = yamlStr

	// compare and don't update if unchanged
	if equality.Semantic.DeepEqual(existingConfigmap, configmap) {
		return rel, controllerutil.OperationResultNone, nil
	}

	err = r.Update(ctx, configmap)
	// If the Prometheus configmap was updated, the VMI Prometheus has ConfigReloader sidecar to signal Prometheus to reload config
	if err != nil {
		return rel, controllerutil.OperationResultNone, err
	}
	return rel, controllerutil.OperationResultUpdated, nil
}

// fetchPrometheusDeploymentFromTrait fetches the Prometheus deployment from information in the trait.
func (r *Reconciler) fetchPrometheusDeploymentFromTrait(ctx context.Context, trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, log vzlog2.VerrazzanoLogger) (*k8sapps.Deployment, error) {
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
	log.Debugf("Found Prometheus deployment %s", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta))
	return deployment, nil
}

// findPrometheusScrapeConfigMapNameFromDeployment finds the Prometheus configmap name from the Prometheus deployment.
func (r *Reconciler) findPrometheusScrapeConfigMapNameFromDeployment(deployment *k8sapps.Deployment, log vzlog2.VerrazzanoLogger) (string, error) {
	volumes := deployment.Spec.Template.Spec.Volumes
	for _, volume := range volumes {
		if volume.Name == "config-volume" && volume.ConfigMap != nil && len(volume.ConfigMap.Name) > 0 {
			name := volume.ConfigMap.Name
			log.Debugf("Found Prometheus configmap name %s", name)
			return name, nil
		}
	}
	return "", fmt.Errorf("failed to find Prometheus configmap name from deployment %s", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta))
}

// updateRelatedDeployment updates the labels and annotations of a related workload deployment.
// For example containerized workloads produce related deployments.
func (r *Reconciler) updateRelatedDeployment(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured, log vzlog2.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	log.Debugf("Update workload deployment %s", vznav.GetNamespacedNameFromUnstructured(child))
	ref := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}
	deployment := &k8sapps.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// If the deployment was not found don't attempt to create or update it.
		if deployment.CreationTimestamp.IsZero() {
			log.Debug("Workload child deployment not found")
			return apierrors.NewNotFound(schema.GroupResource{Group: deployment.APIVersion, Resource: deployment.Kind}, deployment.Name)
		}
		deployment.Spec.Template.ObjectMeta.Annotations = MutateAnnotations(trait, traitDefaults, deployment.Spec.Template.ObjectMeta.Annotations)
		deployment.Spec.Template.ObjectMeta.Labels = MutateLabels(trait, workload, deployment.Spec.Template.ObjectMeta.Labels)
		return nil
	})
	if err != nil && !apierrors.IsNotFound(err) {
		_, err = vzlog.IgnoreConflictWithLog(fmt.Sprintf("Failed to update workload child deployment %s: %v", vznav.GetNamespacedNameFromObjectMeta(deployment.ObjectMeta).Name, err),
			err, zap.S())
	}
	return ref, res, err
}

// updateRelatedStatefulSet updates the labels and annotations of a related workload stateful set.
// For example coherence workloads produce related stateful sets.
func (r *Reconciler) updateRelatedStatefulSet(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured, log vzlog2.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	log.Debugf("Update workload stateful set %s", vznav.GetNamespacedNameFromUnstructured(child))
	ref := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}
	statefulSet := &k8sapps.StatefulSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, statefulSet, func() error {
		// If the statefulset was not found don't attempt to create or update it.
		if statefulSet.CreationTimestamp.IsZero() {
			log.Debug("Workload child statefulset not found")
			return apierrors.NewNotFound(schema.GroupResource{Group: statefulSet.APIVersion, Resource: statefulSet.Kind}, statefulSet.Name)
		}
		statefulSet.Spec.Template.ObjectMeta.Annotations = MutateAnnotations(trait, traitDefaults, statefulSet.Spec.Template.ObjectMeta.Annotations)
		statefulSet.Spec.Template.ObjectMeta.Labels = MutateLabels(trait, workload, statefulSet.Spec.Template.ObjectMeta.Labels)
		return nil
	})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Errorf("Failed to update workload child statefulset %s: %v", vznav.GetNamespacedNameFromObjectMeta(statefulSet.ObjectMeta), err)
	}
	return ref, res, err
}

// updateRelatedPod updates the labels and annotations of a related workload pod.
// For example WLS workloads produce related pods.
func (r *Reconciler) updateRelatedPod(ctx context.Context, trait *vzapi.MetricsTrait, workload *unstructured.Unstructured, traitDefaults *vzapi.MetricsTraitSpec, child *unstructured.Unstructured, log vzlog2.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	log.Debug("Update workload pod %s", vznav.GetNamespacedNameFromUnstructured(child))
	rel := vzapi.QualifiedResourceRelation{APIVersion: child.GetAPIVersion(), Kind: child.GetKind(), Namespace: child.GetNamespace(), Name: child.GetName(), Role: sourceRole}
	pod := &k8score.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: child.GetAPIVersion(), Kind: child.GetKind()},
		ObjectMeta: metav1.ObjectMeta{Namespace: child.GetNamespace(), Name: child.GetName()},
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, pod, func() error {
		// If the pod was not found don't attempt to create or update it.
		if pod.CreationTimestamp.IsZero() {
			log.Debug("Workload child pod not found")
			return apierrors.NewNotFound(schema.GroupResource{Group: pod.APIVersion, Resource: pod.Kind}, pod.Name)
		}
		pod.ObjectMeta.Annotations = MutateAnnotations(trait, traitDefaults, pod.ObjectMeta.Annotations)
		pod.ObjectMeta.Labels = MutateLabels(trait, workload, pod.ObjectMeta.Labels)
		return nil
	})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Errorf("Failed to update workload child pod %s: %v", vznav.GetNamespacedNameFromObjectMeta(pod.ObjectMeta), err)
	}
	return rel, res, err
}

// updateTraitStatus updates the trait's status conditions and resources if they have changed.
// The return value can be used as the result of the Reconcile method.
func (r *Reconciler) updateTraitStatus(ctx context.Context, trait *vzapi.MetricsTrait, results *reconcileresults.ReconcileResults, log vzlog2.VerrazzanoLogger) (reconcile.Result, error) {
	name := vznav.GetNamespacedNameFromObjectMeta(trait.ObjectMeta)

	// If the status content has changed persist the updated status.
	if trait.DeletionTimestamp.IsZero() && updateStatusIfRequired(&trait.Status, results) {
		err := r.Status().Update(ctx, trait)
		if err != nil {
			return vzlog.IgnoreConflictWithLog(fmt.Sprintf("Failed to update metrics trait %s status", name.Name), err, zap.S())
		}
		log.Debugf("Updated metrics trait %s status", name.Name)
	}

	// If the results contained errors then requeue immediately.
	if results.ContainsErrors() {
		vzlog.ResultErrorsWithLog(fmt.Sprintf("Failed to reconcile metrics trait %s", name), results.Errors, zap.S())
		return reconcile.Result{Requeue: true}, nil
	}

	// If the status has not change and there are no errors
	// requeue with a jittered delay to account for situations where a workload
	// changes but without necessarily updating the trait spec.
	var seconds = rand.IntnRange(45, 90)
	var duration = time.Duration(seconds) * time.Second
	log.Debugf("Reconciled metrics trait %s successfully", name.Name)
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
func (r *Reconciler) fetchTraitDefaults(ctx context.Context, workload *unstructured.Unstructured, log vzlog2.VerrazzanoLogger) (*vzapi.MetricsTraitSpec, bool, error) {
	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil {
		return nil, true, err
	}

	workloadType := GetSupportedWorkloadType(apiVerKind)
	switch workloadType {
	case constants.WorkloadTypeWeblogic:
		spec, err := r.NewTraitDefaultsForWLSDomainWorkload(ctx, workload)
		return spec, true, err
	case constants.WorkloadTypeCoherence:
		spec, err := r.NewTraitDefaultsForCOHWorkload(ctx, workload)
		return spec, true, err
	case constants.WorkloadTypeGeneric:
		spec, err := r.NewTraitDefaultsForGenericWorkload()
		return spec, true, err
	default:
		// Log the kind/workload is unsupported and return a nil trait.
		log.Debugf("unsupported kind %s of workload %s", apiVerKind, vznav.GetNamespacedNameFromUnstructured(workload))
		return nil, false, nil

	}

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

	for i := range ports {
		oldScrapeConfigs := prometheusScrapeConfig.Search(prometheusScrapeConfigsLabel).Children()
		prometheusScrapeConfig.Array(prometheusScrapeConfigsLabel) // zero out the array of scrape configs
		newScrapeJob, newScrapeConfig, err := createScrapeConfigFromTrait(ctx, trait, i, secret, workload, c)
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
	}
	return prometheusScrapeConfig, nil
}

// MutateAnnotations mutates annotations with values used by the scraper config.
// Annotations are either set or removed depending on the state of the trait.
func MutateAnnotations(trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, annotations map[string]string) map[string]string {
	mutated := annotations

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

	// If the trait is being deleted, remove the annotations.
	if !trait.DeletionTimestamp.IsZero() {
		for k := range mutated {
			if strings.HasPrefix(k, verrazzanoMetricsAnnotationPrefix) {
				delete(mutated, k)
			}
		}
		return mutated
	}

	// Merge trait, default and existing value.
	var found bool
	var port string
	for i, portSpec := range ports {

		mutated = updateStringMap(mutated, formatMetric(verrazzanoMetricsEnabledAnnotation, i), strconv.FormatBool(true))

		if portSpec.Port != nil {
			port = strconv.Itoa(*portSpec.Port)
		} else {
			port, found = annotations[prometheusPortAnnotation]
			if !found {
				port = strconv.Itoa(*traitDefaults.Ports[0].Port)
			}
		}
		mutated = updateStringMap(mutated, formatMetric(verrazzanoMetricsPortAnnotation, i), port)

		// Merge trait, default and existing value.
		var path string
		if portSpec.Path != nil {
			path = *portSpec.Path
		} else {
			path, found = annotations[prometheusPathAnnotation]
			if !found {
				if traitDefaults.Ports[0].Path != nil {
					path = *traitDefaults.Ports[0].Path
				}
			}
		}
		mutated = updateStringMap(mutated, formatMetric(verrazzanoMetricsPathAnnotation, i), path)
	}

	return mutated
}

func formatMetric(format string, i int) string {
	suffix := ""
	if i > 0 {
		suffix = strconv.Itoa(i)
	}
	return fmt.Sprintf(format, suffix)
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
	value, ok := namespace.Labels["istio-injection"]
	if ok && value == "enabled" {
		return true, nil
	}
	return false, nil
}

// createPrometheusScrapeConfigMapJobName creates a Prometheus scrape configmap job name from a trait.
// Format is {oam_app}_{cluster}_{namespace}_{oam_comp}
func createPrometheusScrapeConfigMapJobName(trait *vzapi.MetricsTrait, portNum int) (string, error) {
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
	portStr := ""
	if portNum > 0 {
		portStr = fmt.Sprintf("_%d", portNum)
	}
	return fmt.Sprintf("%s_%s_%s_%s%s", app, cluster, namespace, comp, portStr), nil
}

// createScrapeConfigFromTrait creates Prometheus scrape config for a trait.
// This populates the Prometheus scrape config template.
// The job name is returned.
// The YAML container populated from the Prometheus scrape config template is returned.
func createScrapeConfigFromTrait(ctx context.Context, trait *vzapi.MetricsTrait, portIncrement int, secret *k8score.Secret, workload *unstructured.Unstructured, c client.Client) (string, *gabs.Container, error) {

	// TODO: see if we can create a scrape job per port within this method. change name to createScrapeConfigsFromTrait
	job, err := createPrometheusScrapeConfigMapJobName(trait, portIncrement)
	if err != nil {
		return "", nil, err
	}

	// If workload is nil then the trait is being deleted so no config is required
	if workload != nil {
		// Populate the Prometheus scrape config template
		portOrderStr := ""
		if portIncrement > 0 {
			portOrderStr = strconv.Itoa(portIncrement)
		}
		context := map[string]string{
			appNameHolder:       trait.Labels[appObjectMetaLabel],
			compNameHolder:      trait.Labels[compObjectMetaLabel],
			jobNameHolder:       job,
			portOrderHolder:     portOrderStr,
			namespaceHolder:     trait.Namespace,
			sslProtocolHolder:   httpProtocol,
			vzClusterNameHolder: clusters.GetClusterName(ctx, c)}

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

// removedTraitReferencesFromOwner removes traits from components of owner ApplicationConfiguration.
func (r *Reconciler) removedTraitReferencesFromOwner(ctx context.Context, ownerRef *metav1.OwnerReference, trait *vzapi.MetricsTrait, log vzlog2.VerrazzanoLogger) (vzapi.QualifiedResourceRelation, controllerutil.OperationResult, error) {
	rel := vzapi.QualifiedResourceRelation{APIVersion: "core.oam.dev/v1alpha2", Kind: "ApplicationConfiguration", Namespace: trait.GetNamespace(), Name: ownerRef.Name, Role: ownerRole}
	var appConfig oamv1.ApplicationConfiguration
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: trait.GetNamespace(), Name: ownerRef.Name}, &appConfig)
	if err != nil {
		log.Debugf("Unable to fetch ApplicationConfiguration %s/%s, error: %v", trait.GetNamespace(), ownerRef.Name, err)
		return rel, controllerutil.OperationResultNone, err
	}

	if appConfig.Spec.Components != nil {
		traitsRemoved := false
		for i := range appConfig.Spec.Components {
			component := &appConfig.Spec.Components[i]
			if component.Traits != nil {
				remainingTraits := []oamv1.ComponentTrait{}
				for _, componentTrait := range component.Traits {
					remainingTraits = append(remainingTraits, componentTrait)
					componentTraitUnstructured, err := vznav.ConvertRawExtensionToUnstructured(&componentTrait.Trait)
					if err != nil || componentTraitUnstructured == nil {
						log.Debugf("Unable to convert trait for component: %s of application configuration: %s/%s, error: %v", component.ComponentName, appConfig.GetNamespace(), appConfig.GetName(), err)
					} else {
						if componentTraitUnstructured.GetAPIVersion() == trait.APIVersion && componentTraitUnstructured.GetKind() == trait.Kind {
							if compName, ok := trait.Labels[compObjectMetaLabel]; ok && compName == component.ComponentName {
								log.Infof("Removing trait %s/%s for component: %s of application configuration: %s/%s", componentTraitUnstructured.GetAPIVersion(), componentTraitUnstructured.GetKind(), component.ComponentName, appConfig.GetNamespace(), appConfig.GetName())
								remainingTraits = remainingTraits[:len(remainingTraits)-1]
							}
						}
					}
				}
				if len(remainingTraits) < len(component.Traits) {
					component.Traits = remainingTraits
					traitsRemoved = true
				}
			}
		}
		if traitsRemoved {
			log.Infof("Updating ApplicationConfiguration %s/%s", trait.GetNamespace(), ownerRef.Name)
			err = r.Client.Update(ctx, &appConfig)
			if err != nil {
				log.Infof("Unable to update ApplicationConfiguration %s/%s, error: %v", trait.GetNamespace(), ownerRef.Name, err)
				return rel, controllerutil.OperationResultNone, err
			}

			return rel, controllerutil.OperationResultUpdated, err
		}
	}
	return rel, controllerutil.OperationResultNone, nil
}
