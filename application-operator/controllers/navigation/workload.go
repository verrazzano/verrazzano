// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// workloadToContainedGVKMap maps Verrazzano workload GroupVersionKind strings to schema.GroupVersionKind
// structs of the resources that the workloads contain. This is needed because the embedded resources
// do not have API version and kind fields.
var workloadToContainedGVKMap = map[string]schema.GroupVersionKind{
	"oam.verrazzano.io/v1alpha1.VerrazzanoWebLogicWorkload":  {Group: "weblogic.oracle", Version: "v8", Kind: "Domain"},
	"oam.verrazzano.io/v1alpha1.VerrazzanoCoherenceWorkload": {Group: "coherence.oracle.com", Version: "v1", Kind: "Coherence"},
}

// FetchWorkloadDefinition fetches the workload definition of the provided workload.
// The definition is found by converting the workload APIVersion and Kind to a CRD resource name.
// for example core.oam.dev/v1alpha2.ContainerizedWorkload would be converted to
// containerizedworkloads.core.oam.dev.  Workload definitions are always found in the default
// namespace.
func FetchWorkloadDefinition(ctx context.Context, cli client.Reader, log logr.Logger, workload *unstructured.Unstructured) (*v1alpha2.WorkloadDefinition, error) {
	if workload == nil {
		return nil, fmt.Errorf("invalid workload reference")
	}
	workloadAPIVer, _, _ := unstructured.NestedString(workload.Object, "apiVersion")
	workloadKind, _, _ := unstructured.NestedString(workload.Object, "kind")
	workloadName := GetDefinitionOfResource(workloadAPIVer, workloadKind)
	workloadDef := v1alpha2.WorkloadDefinition{}
	if err := cli.Get(ctx, workloadName, &workloadDef); err != nil {
		log.Error(err, "Failed to fetch workload definition", "workload", workloadName)
		return nil, err
	}
	return &workloadDef, nil
}

// FetchWorkloadChildren finds the children resource of a workload resource.
// Both the workload and the returned array of children are unstructured maps of primitives.
// Finding children is done by first looking to the workflow definition of the provided workload.
// The workload definition contains a set of child resource types supported by the workload.
// The namespace of the workload is then searched for child resources of the supported types.
func FetchWorkloadChildren(ctx context.Context, cli client.Reader, log logr.Logger, workload *unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	var err error
	var workloadDefinition *v1alpha2.WorkloadDefinition

	// Attempt to fetch workload definition based on the workload GVK.
	if workloadDefinition, err = FetchWorkloadDefinition(ctx, cli, log, workload); err != nil {
		log.Info("Workload definition not found")
		return nil, err
	}
	// If the workload definition is found then fetch child resources of the declared child types
	var children []*unstructured.Unstructured
	if children, err = FetchUnstructuredChildResourcesByAPIVersionKinds(ctx, cli, log, workload.GetNamespace(), workload.GetUID(), workloadDefinition.Spec.ChildResourceKinds); err != nil {
		return nil, err
	}
	return children, nil
}

// ComponentFromWorkloadLabels returns the OAM component from the application configuration that references
// the workload. The workload lookup is done using the OAM labels from the workload metadata.
func ComponentFromWorkloadLabels(ctx context.Context, cli client.Reader, namespace string, labels map[string]string) (*oamv1.ApplicationConfigurationComponent, error) {
	// look up the OAM application that aggregates this workload
	componentName, ok := labels[oam.LabelAppComponent]
	if !ok {
		return nil, errors.New("OAM component label missing from metadata")
	}
	appName, ok := labels[oam.LabelAppName]
	if !ok {
		return nil, errors.New("OAM app name label missing from metadata")
	}

	appConfig := oamv1.ApplicationConfiguration{}
	name := types.NamespacedName{
		Namespace: namespace,
		Name:      appName,
	}

	if err := cli.Get(ctx, name, &appConfig); err != nil {
		return nil, err
	}

	// find our component in the app config components collection
	for _, c := range appConfig.Spec.Components {
		if c.ComponentName == componentName {
			return &c, nil
		}
	}

	return nil, errors.New("Unable to find application component for workload")
}

// LoggingScopeFromWorkloadLabels returns the LoggingScope object associated with the workload or nil if
// there is no associated logging scope. The workload lookup is done using the OAM labels from the workload metadata.
func LoggingScopeFromWorkloadLabels(ctx context.Context, cli client.Reader, namespace string, labels map[string]string) (*vzapi.LoggingScope, error) {
	component, err := ComponentFromWorkloadLabels(ctx, cli, namespace, labels)
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
			return &scope, nil
		}
	}

	return nil, nil
}

// MetricsTraitFromWorkloadLabels returns the MetricsTrait object associated with the workload or nil if
// there is no associated metrics trait for the workload. If there is an associated metrics trait and the lookup of the
// trait fails, an error is returned and the reconcile should be retried.
func MetricsTraitFromWorkloadLabels(ctx context.Context, cli client.Reader, log logr.Logger, namespace string, workloadMeta v1.ObjectMeta) (*vzapi.MetricsTrait, error) {
	log.Info(fmt.Sprintf("Getting metrics trait from OAM labels: %v", workloadMeta.Labels))
	component, err := ComponentFromWorkloadLabels(ctx, cli, namespace, workloadMeta.Labels)
	if err != nil {
		return nil, err
	}

	hasMetricsTrait := false
	for _, t := range component.Traits {
		u, err := ConvertRawExtensionToUnstructured(&t.Trait)
		if err != nil {
			return nil, err
		}

		if u.GetKind() == vzapi.MetricsTraitKind {
			hasMetricsTrait = true
			metricsTraitList := &vzapi.MetricsTraitList{}
			metricsTraitList.APIVersion = u.GetAPIVersion()
			metricsTraitList.Kind = u.GetKind()

			if err := cli.List(ctx, metricsTraitList, client.InNamespace(namespace)); err != nil {
				return nil, err
			}

			ownerUIDs := make(map[types.UID]struct{}, len(workloadMeta.OwnerReferences))
			for _, owner := range workloadMeta.OwnerReferences {
				ownerUIDs[owner.UID] = struct{}{}
			}
			log.Info(fmt.Sprintf("Workload owner UID's: %v", ownerUIDs))

			for _, item := range metricsTraitList.Items {
				for _, owner := range item.GetOwnerReferences() {
					log.Info(fmt.Sprintf("Comparing metrics trait owner with UID: %s and name: %s", owner.UID, item.Spec.WorkloadReference.Name))
					if _, ok := ownerUIDs[owner.UID]; ok {
						if workloadMeta.Name == item.Spec.WorkloadReference.Name {
							log.Info("Matched Trait")
							return &item, nil
						}
					}
				}
			}
		}
	}

	if hasMetricsTrait {
		log.Info(fmt.Sprintf("Unable to lookup associated MetricTrait for workload %s", workloadMeta.Name))
		return nil, fmt.Errorf("lookup of MetricTrait failed for workload %s", workloadMeta.Name)
	}
	log.Info(fmt.Sprintf("Workload %s has no associated metric trait", workloadMeta.Name))
	return nil, nil
}

// IsVerrazzanoWorkloadKind returns true if the workload is a Verrazzano workload kind
// (e.g. VerrazzanoWebLogicWorkload), false otherwise.
func IsVerrazzanoWorkloadKind(workload *unstructured.Unstructured) bool {
	kind := workload.GetKind()
	return strings.HasPrefix(kind, "Verrazzano") && strings.HasSuffix(kind, "Workload")
}

// IsOwnedByVerrazzanoWorkloadKind returns true if the workloads owner is a Verrazzano workload kind
func IsOwnedByVerrazzanoWorkloadKind(workload *unstructured.Unstructured) bool {
	for _, owner := range workload.GetOwnerReferences() {
		if strings.HasPrefix(owner.Kind, "Verrazzano") && strings.HasSuffix(owner.Kind, "Workload") {
			return true
		}
	}
	return false
}

// APIVersionAndKindToContainedGVK returns the GroupVersionKind of the contained resource
// for the given wrapper resource API version and kind.
func APIVersionAndKindToContainedGVK(apiVersion string, kind string) *schema.GroupVersionKind {
	key := fmt.Sprintf("%s.%s", apiVersion, kind)
	gvk, ok := workloadToContainedGVKMap[key]
	if ok {
		return &gvk
	}
	return nil
}

// WorkloadToContainedGVK returns the GroupVersionKind of the contained resource
// for the type wrapped by the provided Verrazzano workload.
func WorkloadToContainedGVK(workload *unstructured.Unstructured) *schema.GroupVersionKind {
	return APIVersionAndKindToContainedGVK(workload.GetAPIVersion(), workload.GetKind())
}

// GetContainedWorkloadVersionKindName returns the API version, kind, and name of the contained workload
// inside a Verrazzano*Workload.
func GetContainedWorkloadVersionKindName(workload *unstructured.Unstructured) (string, string, string, error) {
	gvk := WorkloadToContainedGVK(workload)
	if gvk == nil {
		return "", "", "", fmt.Errorf("Unable to find contained GroupVersionKind for workload: %v", workload)
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()

	// NOTE: this may need to change if we do not allow the user to set the name or if we do and default it
	// to the workload or component name
	name, found, err := unstructured.NestedString(workload.Object, "spec", "template", "metadata", "name")
	if !found || err != nil {
		return "", "", "", errors.New("Unable to find metadata name in contained workload")
	}

	return apiVersion, kind, name, nil
}

// FetchContainedWorkload takes a Verrazzano workload and fetches the contained workload as unstructured.
func FetchContainedWorkload(ctx context.Context, cli client.Reader, workload *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	apiVersion, kind, name, err := GetContainedWorkloadVersionKindName(workload)
	if err != nil {
		return nil, err
	}

	u := &unstructured.Unstructured{}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)

	err = cli.Get(ctx, client.ObjectKey{Namespace: workload.GetNamespace(), Name: name}, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}
