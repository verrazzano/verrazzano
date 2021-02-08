// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"context"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchTrait attempts to get a trait given a namespaced name.
// Will return nil for the trait and no error if the trait does not exist.
func FetchTrait(ctx context.Context, cli client.Reader, log logr.Logger, name types.NamespacedName) (*vzapi.MetricsTrait, error) {
	var trait vzapi.MetricsTrait
	log.Info("Fetch trait", "trait", name)
	if err := cli.Get(ctx, name, &trait); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Trait has been deleted", "trait", name)
			return nil, nil
		}
		log.Info("Failed to fetch trait", "trait", name)
		return nil, err
	}
	return &trait, nil
}

// FetchWorkloadFromTrait fetches a workload resource using data from a trait resource.
// The trait's workload reference is populated by the OAM runtime when the trait resource
// is created.  This provides a way for the trait's controller to locate the workload resource
// that was generated from the common applicationconfiguration resource.
func FetchWorkloadFromTrait(ctx context.Context, cli client.Reader, log logr.Logger, trait oam.Trait) (*unstructured.Unstructured, error) {
	var workload = &unstructured.Unstructured{}
	workload.SetAPIVersion(trait.GetWorkloadReference().APIVersion)
	workload.SetKind(trait.GetWorkloadReference().Kind)
	workloadKey := client.ObjectKey{Name: trait.GetWorkloadReference().Name, Namespace: trait.GetNamespace()}
	var err error
	log.Info("Fetch workload", "workload", workloadKey)
	if err = cli.Get(ctx, workloadKey, workload); err != nil {
		log.Error(err, "Failed to fetch workload", "workload", workloadKey)
		return nil, err
	}

	if IsVerrazzanoWorkloadKind(workload) {
		// this is one of our wrapper workloads so we need to unwrap and pull out the real workload
		workload, err = FetchContainedWorkload(ctx, cli, workload)
		if err != nil {
			log.Error(err, "Failed to fetch contained workload", "workload", workload)
			return nil, err
		}
	}

	return workload, nil
}
