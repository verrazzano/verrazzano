// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingresstrait

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	pluralize "github.com/gertd/go-pluralize"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/oam-application-operator/controllers/reconcileresults"
	istionet "istio.io/api/networking/v1alpha3"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	gatewayAPIVersion        = "networking.istio.io/v1alpha3"
	gatewayKind              = "Gateway"
	virtualServiceAPIVersion = "networking.istio.io/v1alpha3"
	virtualServiceKind       = "VirtualService"
	serviceAPIVersion        = "v1"
	serviceKind              = "Service"
	clusterIPNone            = "None"
)

// Reconciler is used to reconcile a IngressTrait object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager creates a controller and adds it to the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.IngressTrait{}).
		Complete(r)
}

// Reconcile reconciles a IngressTrait to an Ingress object.
// This results in a related Ingress object being created or updated.
// This also results in the Status of the IngressTrait resource being updated.
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=ingresstraits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=ingresstraits/status,verbs=get;update;patch
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var err error
	ctx := context.Background()
	log := r.Log.WithValues("trait", req.NamespacedName)
	log.Info("Reconcile ingress trait")

	// Fetch the trait.
	var trait *vzapi.IngressTrait
	if trait, err = r.fetchTrait(ctx, req.NamespacedName); err != nil {
		return reconcile.Result{}, err
	}

	// If the trait no longer exists or is being deleted then return success.
	if trait == nil || isTraitBeingDeleted(trait) {
		return reconcile.Result{}, nil
	}

	// Find the service associated with the trait in the application configuration.
	var service *corev1.Service
	if service, err = r.fetchServiceFromTrait(ctx, trait); err != nil {
		return reconcile.Result{}, err
	}

	// Create or update the child resources of the trait and collect the outcomes.
	status := r.createOrUpdateChildResources(ctx, trait, service)

	// Update the status of the trait resource using the outcomes of the create or update.
	return r.updateTraitStatus(ctx, trait, status)
}

// createOrUpdateChildResources creates or updates the Gateway and VirtualService resources that
// should be used to setup ingress to the service.
func (r *Reconciler) createOrUpdateChildResources(ctx context.Context, trait *vzapi.IngressTrait, service *corev1.Service) *reconcileresults.ReconcileResults {
	status := reconcileresults.ReconcileResults{}
	rules := trait.Spec.Rules
	// If there are no rules, create a single default rule
	if len(rules) == 0 {
		rules = []vzapi.IngressRule{{}}
	}
	for index, rule := range rules {
		gwName := fmt.Sprintf("%s-rule-%d-gw", trait.Name, index)
		gateway := r.createOrUpdateGateway(ctx, trait, rule, gwName, &status)
		vsName := fmt.Sprintf("%s-rule-%d-vs", trait.Name, index)
		r.createOrUpdateVirtualService(ctx, trait, rule, vsName, service, gateway, &status)
	}
	return &status
}

// updateTraitStatus updates the trait's status conditions and resources if they have changed.
func (r *Reconciler) updateTraitStatus(ctx context.Context, trait *vzapi.IngressTrait, status *reconcileresults.ReconcileResults) (reconcile.Result, error) {
	resources := status.CreateResources()
	if status.ContainsErrors() || !reflect.DeepEqual(trait.Status.Resources, resources) {
		trait.Status = vzapi.IngressTraitStatus{
			ConditionedStatus: status.CreateConditionedStatus(),
			Resources:         resources}
		// Requeue to prevent potential conflict errors being logged.
		return reconcile.Result{Requeue: true}, r.Status().Update(ctx, trait)
	}
	return reconcile.Result{}, nil
}

// isTraitBeingDeleted determines if the trait is in the process of being deleted.
// This is done checking for a non-nil deletion timestamp.
func isTraitBeingDeleted(trait *vzapi.IngressTrait) bool {
	return trait != nil && trait.GetDeletionTimestamp() != nil
}

// fetchTrait attempts to get a trait given a namespaced name.
// Will return nil for the trait and no error if the trait does not exist.
func (r *Reconciler) fetchTrait(ctx context.Context, name types.NamespacedName) (*vzapi.IngressTrait, error) {
	var trait vzapi.IngressTrait
	r.Log.Info("Fetch trait", "trait", name)
	if err := r.Get(ctx, name, &trait); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("Trait has been deleted")
			return nil, nil
		}
		r.Log.Info("Failed to fetch trait")
		return nil, err
	}
	return &trait, nil
}

// fetchWorkloadFromTrait fetches a workload resource using data from a trait resource.
// The trait's workload reference is populated by the OAM runtime when the trait resource
// is created.  This provides a way for the trait's controller to locate the workload resource
// that was generated from the common applicationconfiguration resource.
func (r *Reconciler) fetchWorkloadFromTrait(ctx context.Context, trait oam.Trait) (*unstructured.Unstructured, error) {
	var workload unstructured.Unstructured
	workload.SetAPIVersion(trait.GetWorkloadReference().APIVersion)
	workload.SetKind(trait.GetWorkloadReference().Kind)
	workloadKey := client.ObjectKey{Name: trait.GetWorkloadReference().Name, Namespace: trait.GetNamespace()}
	r.Log.Info("Fetch workload", "workload", workloadKey)
	if err := r.Get(ctx, workloadKey, &workload); err != nil {
		r.Log.Error(err, "Failed to fetch workload", "workload", workloadKey)
		return nil, err
	}
	return &workload, nil
}

// fetchWorkloadDefinition fetches the workload definition of the provided workload.
// The definition is found by converting the workload APIVersion and Kind to a CRD resource name.
// for example core.oam.dev/v1alpha2.ContainerizedWorkload would be converted to
// containerizedworkloads.core.oam.dev.  Workload definitions are always found in the default
// namespace.
func (r *Reconciler) fetchWorkloadDefinition(ctx context.Context, workload *unstructured.Unstructured) (*v1alpha2.WorkloadDefinition, error) {
	workloadAPIVer, _, _ := unstructured.NestedString(workload.Object, "apiVersion")
	workloadKind, _, _ := unstructured.NestedString(workload.Object, "kind")
	workloadName := convertAPIVersionAndKindToNamespacedName(workloadAPIVer, workloadKind)
	workloadDef := v1alpha2.WorkloadDefinition{}
	if err := r.Get(ctx, workloadName, &workloadDef); err != nil {
		r.Log.Error(err, "Failed to fetch workload definition", "name", workloadName)
		return nil, err
	}
	return &workloadDef, nil
}

// fetchWorkloadChildren finds the children resource of a workload resource.
// Both the workload and the returned array of children are unstructured maps of primitives.
// Finding children is done by first looking to the workflow definition of the provided workload.
// The workload definition contains a set of child resource types supported by the workload.
// The namespace of the workload is then searched for child resources of the supported types.
func (r *Reconciler) fetchWorkloadChildren(ctx context.Context, workload *unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	var err error
	var workloadDefinition *v1alpha2.WorkloadDefinition

	// Attempt to fetch workload definition based on the workload GVK.
	if workloadDefinition, err = r.fetchWorkloadDefinition(ctx, workload); err != nil {
		r.Log.Info("Workload definition not found")
	}
	if workloadDefinition != nil {
		// If the workload definition is found then fetch child resources of the declared child types
		var children []*unstructured.Unstructured
		if children, err = r.fetchChildResourcesByAPIVersionKinds(ctx, workload.GetNamespace(), workload.GetUID(), workloadDefinition.Spec.ChildResourceKinds); err != nil {
			return nil, err
		}
		return children, nil
	} else if workload.GetAPIVersion() == appsv1.SchemeGroupVersion.String() {
		// Else if this is a native resource then use the workload itself as the child
		r.Log.Info("Found native workload")
		return []*unstructured.Unstructured{workload}, nil
	} else {
		// Else return an error that the workload type is not supported by this trait.
		r.Log.Info("Workload not supported by trait")
		return nil, fmt.Errorf("Workload not supported by trait")
	}
}

// fetchChildResourcesByAPIVersionKinds find all of the child resource of specific kinds
// having a specific parent UID.  The child kinds are APIVersion and Kind
// (e.g. apps/v1.Deployment or v1.Service).  The objects of these resource kinds are listed
// and the ones having the correct parent UID are collected and accumulated and returned.
// This is used to collect a subset children of a particular parent object.
// ctx - The calling context
// namespace - The namespace to search for children objects
// parentUID - The parent UID a child must have to be included in the result.
// childResKinds - The set of resource kinds a child's resource kind must in to be included in the result.
func (r *Reconciler) fetchChildResourcesByAPIVersionKinds(ctx context.Context, namespace string, parentUID types.UID, childResKinds []v1alpha2.ChildResourceKind) ([]*unstructured.Unstructured, error) {
	var childResources []*unstructured.Unstructured
	r.Log.Info("Fetch child resources")
	for _, childResKind := range childResKinds {
		resources := unstructured.UnstructuredList{}
		resources.SetAPIVersion(childResKind.APIVersion)
		resources.SetKind(childResKind.Kind)
		if err := r.List(ctx, &resources, client.InNamespace(namespace), client.MatchingLabels(childResKind.Selector)); err != nil {
			r.Log.Error(err, "Failed listing child resources")
			return nil, err
		}
		for i, item := range resources.Items {
			for _, owner := range item.GetOwnerReferences() {
				if owner.UID == parentUID {
					r.Log.Info(fmt.Sprintf("Found child %s.%s:%s", item.GetAPIVersion(), item.GetKind(), item.GetName()))
					childResources = append(childResources, &resources.Items[i])
					break
				}
			}
		}
	}
	return childResources, nil
}

// createOrUpdateGateway creates or updates the Gateway child resource of the trait.
// Results are added to the status object.
func (r *Reconciler) createOrUpdateGateway(ctx context.Context, trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, status *reconcileresults.ReconcileResults) *istioclinet.Gateway {
	// Create a gateway populating only name metadata.
	// This is used as default if the gateway needs to be created.
	gateway := &istioclinet.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayAPIVersion,
			Kind:       gatewayKind},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: trait.Namespace,
			Name:      name}}

	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, gateway, func() error {
		return r.mutateGateway(gateway, trait, rule)
	})

	ref := vzapi.QualifiedResourceRelation{APIVersion: gatewayAPIVersion, Kind: gatewayKind, Name: trait.Name, Role: "gateway"}
	status.Relations = append(status.Relations, ref)
	status.Results = append(status.Results, res)
	status.Errors = append(status.Errors, err)

	if err != nil {
		r.Log.Error(err, "Failed to create or update gateway.")
	}

	return gateway
}

// mutateGateway mutates the output Gateway child resource.
func (r *Reconciler) mutateGateway(gateway *istioclinet.Gateway, trait *vzapi.IngressTrait, rule vzapi.IngressRule) error {
	// Set the spec content.
	gateway.Spec.Selector = map[string]string{"istio": "ingressgateway"}
	gateway.Spec.Servers = []*istionet.Server{{
		Hosts: createHostsFromIngressTraitRule(rule),
		Port: &istionet.Port{
			Name:     "http",
			Number:   80,
			Protocol: "HTTP"}}}

	// Set the owner reference.
	controllerutil.SetControllerReference(trait, gateway, r.Scheme)
	return nil
}

// createOrUpdateVirtualService creates or updates the VirtualService child resource of the trait.
// Results are added to the status object.
func (r *Reconciler) createOrUpdateVirtualService(ctx context.Context, trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, service *corev1.Service, gateway *istioclinet.Gateway, status *reconcileresults.ReconcileResults) {
	// Create a virtual service populating only name metadata.
	// This is used as default if the virtual service needs to be created.
	virtualService := &istioclinet.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: virtualServiceAPIVersion,
			Kind:       virtualServiceKind},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: trait.Namespace,
			Name:      name}}

	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, virtualService, func() error {
		return r.mutateVirtualService(virtualService, trait, rule, service, gateway)
	})

	ref := vzapi.QualifiedResourceRelation{APIVersion: virtualServiceAPIVersion, Kind: virtualServiceKind, Name: trait.Name, Role: "virtualservice"}
	status.Relations = append(status.Relations, ref)
	status.Results = append(status.Results, res)
	status.Errors = append(status.Errors, err)

	if err != nil {
		r.Log.Error(err, "Failed to create or update virtual service.")
	}
}

// mutateVirtualService mutates the output virtual service resource
func (r *Reconciler) mutateVirtualService(virtualService *istioclinet.VirtualService, trait *vzapi.IngressTrait, rule vzapi.IngressRule, service *corev1.Service, gateway *istioclinet.Gateway) error {
	// Set the spec content.
	virtualService.Spec.Gateways = []string{gateway.Name}
	virtualService.Spec.Hosts = createHostsFromIngressTraitRule(rule)
	matches := []*istionet.HTTPMatchRequest{}
	paths := getPathsFromRule(rule)
	for _, path := range paths {
		matches = append(matches, &istionet.HTTPMatchRequest{
			Uri: createVirtualServiceMatchURIFromIngressTraitPath(path)})
	}
	dest := createDestinationFromService(service)
	route := istionet.HTTPRoute{
		Match: matches,
		Route: []*istionet.HTTPRouteDestination{&dest}}
	virtualService.Spec.Http = []*istionet.HTTPRoute{&route}

	// Set the owner reference.
	controllerutil.SetControllerReference(trait, virtualService, r.Scheme)
	return nil
}

// getPathsFromRule gets the paths from a trait.
// If the trait has no paths a default path is returned.
func getPathsFromRule(rule vzapi.IngressRule) []vzapi.IngressPath {
	paths := rule.Paths
	// If there are no paths create a default.
	if len(paths) == 0 {
		paths = []vzapi.IngressPath{{Path: "/", PathType: "prefix"}}
	}
	return paths
}

// createDestinationFromService creates a virtual service destination from a Service.
// If the service does not have a port it is not included in the destination.
func createDestinationFromService(service *corev1.Service) istionet.HTTPRouteDestination {
	dest := istionet.HTTPRouteDestination{
		Destination: &istionet.Destination{Host: service.Name}}
	// If the related service declares a port add it to the destination.
	if len(service.Spec.Ports) > 0 {
		dest.Destination.Port = &istionet.PortSelector{Number: uint32(service.Spec.Ports[0].Port)}
	}
	return dest
}

// createVirtualServiceMatchURIFromIngressTraitPath create the virtual service match uri map from an ingress trait path
// This is primarily used to setup defaults when either path or type are not present in the ingress path.
// If the provided ingress path doesn't contain a path it is default to /
// If the provided ingress path doesn't contain a type it is defaulted to prefix if path is / and exact otherwise.
func createVirtualServiceMatchURIFromIngressTraitPath(path vzapi.IngressPath) *istionet.StringMatch {
	// Default path to /
	p := strings.TrimSpace(path.Path)
	if p == "" {
		p = "/"
	}

	// If path is / default type to prefix
	// If path is not / default to exact
	t := strings.ToLower(strings.TrimSpace(path.PathType))
	if t == "" {
		if p == "/" {
			t = "prefix"
		} else {
			t = "exact"
		}
	}

	switch t {
	case "regex":
		return &istionet.StringMatch{MatchType: &istionet.StringMatch_Regex{Regex: p}}
	case "prefix":
		return &istionet.StringMatch{MatchType: &istionet.StringMatch_Prefix{Prefix: p}}
	default:
		return &istionet.StringMatch{MatchType: &istionet.StringMatch_Exact{Exact: p}}
	}
}

// createHostsFromIngressTraitRule creates an array of hosts from an ingress rule.
// It is primarily used to setup defaults when either no hosts are provided or the host is blank.
// If the rule contains an empty host array a host array with a single "*" element is created.
// Otherwise any blank hosts in the input ingress rules are replaced with "*" values.
func createHostsFromIngressTraitRule(rule vzapi.IngressRule) []string {
	hosts := rule.Hosts
	if len(rule.Hosts) == 0 {
		hosts = []string{"*"}
	}
	for i, h := range hosts {
		h = strings.TrimSpace(h)
		if len(h) == 0 {
			h = "*"
		}
		hosts[i] = h
	}
	return hosts
}

// fetchServiceFromTrait traverses from an ingress trait resource to the related service resource and returns it.
// This is done by first finding the workload related to the trait.
// Then the child resources of the workload are founds.
// Finally those child resources are scanned to find a Service resource which is returned.
func (r *Reconciler) fetchServiceFromTrait(ctx context.Context, trait *vzapi.IngressTrait) (*corev1.Service, error) {
	var err error

	// Fetch workload resource
	var workload *unstructured.Unstructured
	if workload, err = r.fetchWorkloadFromTrait(ctx, trait); err != nil {
		return nil, err
	}

	// Fetch workload child resources
	var children []*unstructured.Unstructured
	if children, err = r.fetchWorkloadChildren(ctx, workload); err != nil {
		return nil, err
	}

	// Find the service from within the list of unstructured child resources
	var service *corev1.Service
	service, err = extractServiceFromUnstructuredChildren(children)
	if err != nil {
		return nil, err
	}

	return service, nil
}

// extractServiceFromUnstructuredChildren finds and returns Service in an array of unstructured child service.
// The children array is scanned looking for Service's APIVersion and Kind, selecting the first service with a
// cluster IP. If no service has a cluster IP, choose the first service.
// If found the unstructured data is converted to a Service object and returned.
// children - An array of unstructured children
func extractServiceFromUnstructuredChildren(children []*unstructured.Unstructured) (*corev1.Service, error) {
	var selectedService *corev1.Service

	for _, child := range children {
		if child.GetAPIVersion() == serviceAPIVersion && child.GetKind() == serviceKind {
			var service corev1.Service
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(child.UnstructuredContent(), &service)
			if err != nil {
				// maybe we should continue here and hope that another child can be converted?
				return nil, err
			}

			if selectedService == nil {
				selectedService = &service
			}

			if service.Spec.ClusterIP != clusterIPNone {
				selectedService = &service
				break
			}
		}
	}

	if selectedService != nil {
		return selectedService, nil
	}

	return nil, fmt.Errorf("No child service found")
}

// convertAPIVersionAndKindToNamespacedName converts APIVersion and Kind of CR to a CRD namespaced name.
// For example CR APIVersion.Kind core.oam.dev/v1alpha2.ContainerizedWorkload would be converted
// to containerizedworkloads.core.oam.dev in the default (i.e. "") namespace.
// apiVersion - The CR APIVersion
// kind - The CR Kind
func convertAPIVersionAndKindToNamespacedName(apiVersion string, kind string) types.NamespacedName {
	grp, ver := convertAPIVersionToGroupAndVersion(apiVersion)
	res := pluralize.NewClient().Plural(strings.ToLower(kind))
	grpVerRes := metav1.GroupVersionResource{
		Group:    grp,
		Version:  ver,
		Resource: res,
	}
	name := grpVerRes.Resource + "." + grpVerRes.Group
	return types.NamespacedName{Namespace: "", Name: name}
}

// convertAPIVersionToGroupAndVersion splits APIVersion into API and version parts.
// An APIVersion takes the form api/version (e.g. networking.k8s.io/v1)
// If the input does not contain a / the group is defaulted to the empty string.
// apiVersion - The combined api and version to split
func convertAPIVersionToGroupAndVersion(apiVersion string) (string, string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) < 2 {
		// Use empty group for core types.
		return "", parts[0]
	}
	return parts[0], parts[1]
}
