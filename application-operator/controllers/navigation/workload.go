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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var workloadPropertyNameMap = map[string]string{
	"oam.verrazzano.io/v1alpha1.VerrazzanoWebLogicWorkload":  "domain",
	"oam.verrazzano.io/v1alpha1.VerrazzanoCoherenceWorkload": "coherence",
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

// IsVerrazzanoWorkloadKind returns true if the workload is a Verrazzano workload kind
// (e.g. VerrazzanoWebLogicWorkload), false otherwise.
func IsVerrazzanoWorkloadKind(workload *unstructured.Unstructured) bool {
	kind := workload.GetKind()
	return strings.HasPrefix(kind, "Verrazzano") && strings.HasSuffix(kind, "Workload")
}

// GetContainedWorkloadVersionKindName returns the api version, kind, and name of the contained workload
// inside a Verrazzano*Workload.
func GetContainedWorkloadVersionKindName(workload *unstructured.Unstructured) (string, string, string, error) {
	// container workloads use different properties to hold the resource, so use a map
	// to get the property name (e.g. VerrazzanoWebLogicWorkload has a "spec" containing a "domain" property)
	key := fmt.Sprintf("%s.%s", workload.GetAPIVersion(), workload.GetKind())
	specProperty, ok := workloadPropertyNameMap[key]
	if !ok {
		return "", "", "", fmt.Errorf("Unable to find spec property for workload type: %s", workload.GetKind())
	}

	apiVersion, found, err := unstructured.NestedString(workload.Object, "spec", specProperty, "apiVersion")
	if !found || err != nil {
		return "", "", "", errors.New("Unable to find api version in contained workload")
	}

	kind, found, err := unstructured.NestedString(workload.Object, "spec", specProperty, "kind")
	if !found || err != nil {
		return "", "", "", errors.New("Unable to find kind in contained workload")
	}

	name, found, err := unstructured.NestedString(workload.Object, "spec", specProperty, "metadata", "name")
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
