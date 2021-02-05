// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingresstrait

import (
	"context"
	"errors"
	"fmt"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"reflect"
	"strings"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	pluralize "github.com/gertd/go-pluralize"
	"github.com/go-logr/logr"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"github.com/verrazzano/verrazzano/application-operator/controllers/reconcileresults"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1beta1"
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
	certificateAPIVersion    = "cert-manager.io/v1alpha2"
	certificateKind          = "Certificate"
	serviceAPIVersion        = "v1"
	serviceKind              = "Service"
	clusterIPNone            = "None"
	verrazzanoClusterIssuer  = "verrazzano-cluster-issuer"
)

// Reconciler is used to reconcile an IngressTrait object
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

// Reconcile reconciles an IngressTrait with other related resources required for ingress.
// This also results in the status of the ingress trait resource being updated.
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
		r.createOrUpdateGatewayCertificate(ctx, trait, &status)
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

func (r *Reconciler) createOrUpdateGatewayCertificate(ctx context.Context, trait *vzapi.IngressTrait, status *reconcileresults.ReconcileResults) error {
	const istioNamespace = "istio-system"

	// derive certificate name and see if it exists, in ready state, and has appropriate SAN value(s)
	certificate := certapiv1alpha2.Certificate{}
	certName, err := buildCertificateNameFromIngressTrait(trait)
	if err != nil {
		return err
	}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: certName, Namespace: istioNamespace}, &certificate); err != nil {
		if k8serrors.IsNotFound(err) {
			// proceed with certificate creation
			certificate = certapiv1alpha2.Certificate{
				TypeMeta: metav1.TypeMeta{
					Kind:       certificateKind,
					APIVersion: certificateAPIVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      certName,
					Namespace: istioNamespace,
				},
				Spec: certapiv1alpha2.CertificateSpec{
					DNSNames:   []string{fmt.Sprintf("*.%s", buildAppDomainName(r, trait))},
					SecretName: fmt.Sprintf("secret-%s", buildCertificateNameFromIngressTrait(trait)),
					IssuerRef: certv1.ObjectReference{
						Name: verrazzanoClusterIssuer,
						Kind: "ClusterIssuer",
					},
				},
			}
		} else {
			return err
		}
	} else {
		// certificate already exists
		r.Log.Info("Certificate exists - nothing to do")
	}

	return nil
}

func buildCertificateNameFromIngressTrait(trait *vzapi.IngressTrait) (string, error) {
	appName, ok := trait.Labels[oam.LabelAppName]
	if !ok {
		return "", errors.New("OAM app name label missing from metadata, unable to add ingress trait")
	}
	return fmt.Sprintf("%s-%s-cert", trait.Namespace, appName), nil
}

// createOrUpdateGateway creates or updates the Gateway child resource of the trait.
// Results are added to the status object.
func (r *Reconciler) createOrUpdateGateway(ctx context.Context, trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, status *reconcileresults.ReconcileResults) *istioclient.Gateway {
	// Create a gateway populating only name metadata.
	// This is used as default if the gateway needs to be created.
	gateway := &istioclient.Gateway{
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
func (r *Reconciler) mutateGateway(gateway *istioclient.Gateway, trait *vzapi.IngressTrait, rule vzapi.IngressRule) error {
	hosts, err := createHostsFromIngressTraitRule(r, rule, trait)
	if err != nil {
		return err
	}

	// Set the spec content.
	gateway.Spec.Selector = map[string]string{"istio": "ingressgateway"}
	gateway.Spec.Servers = []*istionet.Server{{
		Hosts: hosts,
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
func (r *Reconciler) createOrUpdateVirtualService(ctx context.Context, trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, service *corev1.Service, gateway *istioclient.Gateway, status *reconcileresults.ReconcileResults) {
	// Create a virtual service populating only name metadata.
	// This is used as default if the virtual service needs to be created.
	virtualService := &istioclient.VirtualService{
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
func (r *Reconciler) mutateVirtualService(virtualService *istioclient.VirtualService, trait *vzapi.IngressTrait, rule vzapi.IngressRule, service *corev1.Service, gateway *istioclient.Gateway) error {
	// Set the spec content.
	var err error
	virtualService.Spec.Gateways = []string{gateway.Name}
	virtualService.Spec.Hosts, err = createHostsFromIngressTraitRule(r, rule, trait)
	if err != nil {
		return err
	}
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
// It filters out wildcard hosts or hosts that are empty. If there are no valid hosts provided,
// then a DNS host name is automatically generated and used.
func createHostsFromIngressTraitRule(cli client.Reader, rule vzapi.IngressRule, trait *vzapi.IngressTrait) ([]string, error) {
	var validHosts []string
	for _, h := range rule.Hosts {
		h = strings.TrimSpace(h)
		// Ignore empty or wildcard hostname
		if len(h) == 0 || strings.Contains(h, "*") {
			continue
		}
		validHosts = append(validHosts, h)
	}
	// Use default hostname if none of the user specified hosts were valid
	if len(validHosts) == 0 {
		hostName, err := buildAppFullyQualifiedHostName(cli, trait)
		if err != nil {
			return nil, err
		}
		validHosts = []string{hostName}
	}
	return validHosts, nil
}

// fetchServiceFromTrait traverses from an ingress trait resource to the related service resource and returns it.
// This is done by first finding the workload related to the trait.
// Then the child resources of the workload are founds.
// Finally those child resources are scanned to find a Service resource which is returned.
func (r *Reconciler) fetchServiceFromTrait(ctx context.Context, trait *vzapi.IngressTrait) (*corev1.Service, error) {
	var err error

	// Fetch workload resource
	var workload *unstructured.Unstructured
	if workload, err = vznav.FetchWorkloadFromTrait(ctx, r.Client, r.Log, trait); err != nil {
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

// buildAppFullyQualifiedHostName generates a DNS host name for the application using the following structure:
// <app>.<namespace>.<dns-subdomain>  where
//   app is the OAM application name
//   namespace is the namespace of the OAM application
//   dns-subdomain is The DNS subdomain name
// For example: sales.cars.example.com
func buildAppFullyQualifiedHostName(cli client.Reader, trait *vzapi.IngressTrait) (string, error) {
	appName, ok := trait.Labels[oam.LabelAppName]
	if !ok {
		return "", errors.New("OAM app name label missing from metadata, unable to add ingress trait")
	}
	domainName, err := buildAppDomainName(cli, trait)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", appName, domainName), nil
}

// buildAppDomainName generates a domain name for the application using the following structure:
// <namespace>.<dns-subdomain>  where
//   namespace is the namespace of the OAM application
//   dns-subdomain is The DNS subdomain name
// For example: cars.example.com
func buildAppDomainName(cli client.Reader, trait *vzapi.IngressTrait) (string, error) {
	const authRealmKey = "nginx.ingress.kubernetes.io/auth-realm"
	const rancherIngress = "rancher"
	const rancherNamespace = "cattle-system"

	// Extract the domain name from the Rancher ingress
	ingress := k8net.Ingress{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: rancherIngress, Namespace: rancherNamespace}, &ingress)
	if err != nil {
		return "", err
	}
	authRealmAnno, ok := ingress.Annotations[authRealmKey]
	if !ok || len(authRealmAnno) == 0 {
		return "", fmt.Errorf("Annotation %s missing from Rancher ingress, unable to generate DNS name", authRealmKey)
	}
	segs := strings.Split(strings.TrimSpace(authRealmAnno), " ")
	domain := strings.TrimSpace(segs[0])

	// If this is xip.io then build the domain name using Istio info
	if strings.HasSuffix(domain, "xip.io") {
		domain, err = buildDomainNameForXIPIO(cli, trait)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s.%s", trait.Namespace, domain), nil
}

// buildDomainNameForXIPIO generates a domain name in the format of "<IP>.xip.io"
// Get the IP from Istio resources
func buildDomainNameForXIPIO(cli client.Reader, trait *vzapi.IngressTrait) (string, error) {
	const istioIngressGateway = "istio-ingressgateway"
	const istioNamespace = "istio-system"

	istio := corev1.Service{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: istioIngressGateway, Namespace: istioNamespace}, &istio)
	if err != nil {
		return "", err
	}
	var IP string
	if istio.Spec.Type == corev1.ServiceTypeLoadBalancer {
		istioIngress := istio.Status.LoadBalancer.Ingress
		if len(istioIngress) == 0 {
			return "", fmt.Errorf("%s is missing loadbalancer IP", istioIngressGateway)
		}
		IP = istioIngress[0].IP
	} else if istio.Spec.Type == corev1.ServiceTypeNodePort {
		// Do the equiv of the following command to get the IP
		// kubectl -n istio-system get pods --selector app=istio-ingressgateway,istio=ingressgateway -o jsonpath='{.items[0].status.hostIP}'
		podList := corev1.PodList{}
		listOptions := client.MatchingLabels{"app": "istio-ingressgateway", "istio": "ingressgateway"}
		err := cli.List(context.TODO(), &podList, listOptions)
		if err != nil {
			return "", err
		}
		if len(podList.Items) == 0 {
			return "", errors.New("Unable to find Istio ingressway pod")
		}
		IP = podList.Items[0].Status.HostIP
	} else {
		return "", fmt.Errorf("Unsupported service type %s for istio_ingress", string(istio.Spec.Type))
	}
	domain := IP + "." + "xip.io"
	return domain, nil
}
