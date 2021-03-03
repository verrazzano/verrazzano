// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
)

const (
	configMapAPIVersion = "v1"
	configMapKind       = "ConfigMap"
)

// Handler abstracts the FLUENTD integration for components
type Handler interface {
	Apply(ctx context.Context, resource vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope) (*ctrl.Result, error)
	Remove(ctx context.Context, resource vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope) (bool, error)
}

// Reconciler reconciles a LoggingScope object
type Reconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Handlers map[string]Handler
}

// NewReconciler creates a new Logging Scope reconciler
func NewReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *Reconciler {
	handlers := map[string]Handler{
		wlsWorkloadKey:     &wlsHandler{Client: client, Log: log},
		helidonWorkloadKey: &HelidonHandler{Client: client, Log: log},
	}
	return &Reconciler{
		Client:   client,
		Log:      log,
		Scheme:   scheme,
		Handlers: handlers,
	}
}

// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=loggingscopes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=loggingscopes/status,verbs=get;update;patch

// Reconcile reconciles a LoggingScope.
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("scope", req.NamespacedName)
	log.Info("Reconcile logging scope")

	// Fetch the scope.
	scope, err := r.fetchScope(ctx, req.NamespacedName)
	if scope == nil || err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	applyDefaults(r.Client, log, scope)

	var errors []string
	var resources []vzapi.QualifiedResourceRelation
	workloads, _ := fetchWorkloadsFromScope(ctx, r, r.Log, scope)
	for _, workload := range workloads {
		resource := toResource(workload)
		resources = append(resources, resource)

		handler := r.Handlers[handlerKey(resource)]
		if handler == nil {
			// if this is one of our wrapper resources, we expect to not find a handler since logging is
			// added at resource creation time
			if vznav.IsVerrazzanoWorkloadKind(workload) {
				log.Info("Skipping logging scope processing for Verrazzano workload kind", "resource", resource)
			} else {
				log.Error(nil, "Unknown Resource Kind encountered in Logging Scope Controller", "resource", resource)
			}
			continue
		}
		result, err := handler.Apply(ctx, resource, scope)
		if result != nil {
			return *result, nil
		}
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	// check for existing resources which aren't included in workloads
	for _, existingResource := range scope.Status.Resources {
		workloadFound := false
		for _, workload := range workloads {
			if existingResource.Kind == workload.GetKind() &&
				existingResource.Name == workload.GetName() &&
				existingResource.Namespace == workload.GetNamespace() &&
				existingResource.APIVersion == workload.GetAPIVersion() {
				workloadFound = true
				break
			}
		}
		if !workloadFound {
			handler := r.Handlers[handlerKey(existingResource)]
			if handler != nil {
				deleteConfirmed, err := handler.Remove(ctx, existingResource, scope)
				if err != nil {
					errors = append(errors, err.Error())
				}

				if !deleteConfirmed {
					// Add the resource to the scope status until we confirm the remove
					resources = append(resources, existingResource)
				}
			}
		}
	}
	err = r.updateScopeStatus(ctx, resources, scope)
	if err != nil {
		log.Error(err, "Unable to persist resources to scope", "scope", scope)
	}

	if errors != nil {
		return ctrl.Result{}, fmt.Errorf(strings.Join(errors, "\n"))
	}

	return ctrl.Result{}, err
}

func handlerKey(workload vzapi.QualifiedResourceRelation) string {
	return fmt.Sprintf("%s/%s", workload.APIVersion, workload.Kind)
}

// fetchScope attempts to get a scope given a namespaced name.
// Will return nil for the scope and no error if the scope does not exist.
func (r *Reconciler) fetchScope(ctx context.Context, name types.NamespacedName) (*vzapi.LoggingScope, error) {
	var scope vzapi.LoggingScope
	r.Log.Info("Fetch scope", "name", name)
	if err := r.Get(ctx, name, &scope); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("Scope has been deleted")
			return nil, nil
		}
		r.Log.Info("Failed to fetch scope")
		return nil, err
	}

	return &scope, nil
}

// fetchWorkloadsFromScope fetches workload resources using data from a scope resource.
// The scope's workload references are populated by the OAM runtime when the scope resource
// is created.  This provides a way for the scope's controller to locate the workload resources
// that were generated from the common applicationconfiguration resource.
func fetchWorkloadsFromScope(ctx context.Context, cli client.Reader, log logr.Logger, scope oam.Scope) ([]*unstructured.Unstructured, error) {
	workloadLen := len(scope.GetWorkloadReferences())
	if workloadLen == 0 {
		return []*unstructured.Unstructured{}, nil
	}

	workloads := make([]*unstructured.Unstructured, workloadLen)
	for i, workloadRef := range scope.GetWorkloadReferences() {
		var workload unstructured.Unstructured
		workload.SetAPIVersion(workloadRef.APIVersion)
		workload.SetKind(workloadRef.Kind)
		workloadKey := client.ObjectKey{Name: workloadRef.Name, Namespace: scope.GetNamespace()}
		log.Info("Fetch workload", "workload", workloadKey)
		if err := cli.Get(ctx, workloadKey, &workload); err != nil {
			log.Error(err, "Failed to fetch workload", "workload", workloadKey)
			return nil, err
		}
		workloads[i] = &workload
	}
	return workloads, nil
}

// updateScopeStatus the loging scope status with the provided resources
func (r *Reconciler) updateScopeStatus(ctx context.Context, resources []vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope) error {
	scope.Status.Resources = resources
	return r.Status().Update(ctx, scope)
}

// SetupWithManager creates a controller and adds it to the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&oamv1alpha1.LoggingScope{}).
		Complete(r)
}

// toResource creates a QualifiedResourceRelation instance from a workload
func toResource(workload *unstructured.Unstructured) vzapi.QualifiedResourceRelation {
	return vzapi.QualifiedResourceRelation{
		APIVersion: workload.GetAPIVersion(),
		Name:       workload.GetName(),
		Namespace:  workload.GetNamespace(),
		Kind:       workload.GetKind(),
	}
}

// FromWorkloadLabels returns the LoggingScope object associated with the workload or nil if
// there is no associated logging scope. The workload lookup is done using the OAM labels from the workload metadata.
func FromWorkloadLabels(ctx context.Context, cli client.Reader, log logr.Logger, namespace string, labels map[string]string) (*vzapi.LoggingScope, error) {
	component, err := vznav.ComponentFromWorkloadLabels(ctx, cli, namespace, labels)
	if err != nil {
		return nil, err
	}

	// fetch the first logging scope - do we need to handle multiple logging scopes?
	for _, s := range component.Scopes {
		if s.ScopeReference.Kind == vzapi.LoggingScopeKind {
			scope := vzapi.LoggingScope{}
			name := types.NamespacedName{
				Namespace: namespace,
				Name:      s.ScopeReference.Name,
			}
			err = cli.Get(ctx, name, &scope)
			if err != nil {
				return nil, err
			}

			applyDefaults(cli, log, &scope)
			return &scope, nil
		}
	}

	return nil, nil
}

// applyDefaults fills in any empty fields in the logging scope - also handle the case
// where we are running in a managed cluster
func applyDefaults(cli client.Reader, log logr.Logger, scope *vzapi.LoggingScope) {
	if scope.Spec.ElasticSearchURL == "" && scope.Spec.SecretName == "" {
		// if we're running in a managed cluster, use the multicluster ES URL and secret, and if we're
		// not the fields will be empty and we will set these fields to defaults below
		elasticSearchDetails := clusters.FetchManagedClusterElasticSearchDetails(context.TODO(), cli)
		if elasticSearchDetails.URL != "" && elasticSearchDetails.SecretName != "" {
			scope.Spec.ElasticSearchURL = elasticSearchDetails.URL
			scope.Spec.SecretName = elasticSearchDetails.SecretName
		} else if elasticSearchDetails.SecretName != "" && elasticSearchDetails.URL == "" {
			// Elasticsearch secret is misconfigured, log it and set defaults below
			log.Info("Unable to set defaults in logging scope for managed cluster because URL is empty")
		}
	}

	if scope.Spec.FluentdImage == "" {
		scope.Spec.FluentdImage = DefaultFluentdImage
	}
	if scope.Spec.ElasticSearchURL == "" {
		scope.Spec.ElasticSearchURL = DefaultElasticSearchURL
	}
	if scope.Spec.SecretName == "" {
		scope.Spec.SecretName = DefaultSecretName
	}
}
