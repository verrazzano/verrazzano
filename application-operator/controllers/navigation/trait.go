// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"context"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"reflect"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchTrait attempts to get a trait given a namespaced name.
// Will return nil for the trait and no error if the trait does not exist.
func FetchTrait(ctx context.Context, cli client.Reader, log *zap.SugaredLogger, name types.NamespacedName) (*vzapi.MetricsTrait, error) {
	var trait vzapi.MetricsTrait
	log.Debugf("Fetch trait %s", name.Name)
	if err := cli.Get(ctx, name, &trait); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debugf("Trait %s has been deleted", name.Name)
			return nil, nil
		}
		log.Errorf("Failed to fetch trait %s: %v", name, err)
		return nil, err
	}
	return &trait, nil
}

// FetchWorkloadFromTrait fetches a workload resource using data from a trait resource.
// The trait's workload reference is populated by the OAM runtime when the trait resource
// is created.  This provides a way for the trait's controller to locate the workload resource
// that was generated from the common applicationconfiguration resource.
func FetchWorkloadFromTrait(ctx context.Context, cli client.Reader, log vzlog.VerrazzanoLogger, trait oam.Trait) (*unstructured.Unstructured, error) {
	var workload = &unstructured.Unstructured{}
	workload.SetAPIVersion(trait.GetWorkloadReference().APIVersion)
	workload.SetKind(trait.GetWorkloadReference().Kind)
	workloadKey := client.ObjectKey{Name: trait.GetWorkloadReference().Name, Namespace: trait.GetNamespace()}
	var err error
	log.Debugf("Fetch workload %s", workloadKey)
	if err = cli.Get(ctx, workloadKey, workload); err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("Failed to fetch workload %s: %v", workloadKey, err)
		}
		return nil, err
	}

	return FetchWorkloadResource(ctx, cli, log, workload)
}

// IsWeblogicWorkloadKind returns true if the trait references a Verrazzano WebLogic workload kind
// (VerrazzanoWebLogicWorkload), false otherwise.
func IsWeblogicWorkloadKind(trait oam.Trait) bool {
	kind := trait.GetWorkloadReference().Kind
	return kind == vzconst.VerrazzanoWebLogicWorkloadKind
}

// FetchWorkloadResource fetches the underlying resource created by the workload.
func FetchWorkloadResource(ctx context.Context, cli client.Reader, log vzlog.VerrazzanoLogger, workload *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// Getting kind of helidon workload i.e. "VerrazzanoHelidonWorkload"
	helidonWorkloadKind := reflect.TypeOf(vzapi.VerrazzanoHelidonWorkload{}).Name()
	// If the workload does not wrap unstructured data
	if !IsVerrazzanoWorkloadKind(workload) || (helidonWorkloadKind == workload.GetKind()) {
		return workload, nil
	}

	// this is one of our wrapper workloads so we need to unwrap and pull out the real workload
	resource, err := FetchContainedWorkload(ctx, cli, workload)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("Failed to fetch contained workload %s: %v", client.ObjectKey{Name: workload.GetName(), Namespace: workload.GetNamespace()}, err)
		}
		return nil, err
	}

	return resource, nil
}
