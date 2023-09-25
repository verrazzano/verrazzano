// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"context"
	"fmt"

	constants2 "github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Create a registration secret in the verrazzano-system namespace, with the managed cluster information.
// This secret will be used on the admin cluster to get information about itself, like the cluster name,
// so that the admin cluster can manage multi-cluster resources without the need of a VMC.  In the
// case of a cluster that is used as a managed cluster, this secret will still be created, but not used, and
// ultimately replaced when the user applies the managed cluster YAML which has the replacement secret.
func (r *Reconciler) syncLocalRegistrationSecret() error {

	// If the agent secret exists in verrazzano-system namespace then that means this
	// is a managed cluster and the user applied the YAML to create all the secrets.
	// In that case, we do NOT want to create/update this default local secret since
	// it will override the one created by the YAML
	var secret corev1.Secret
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.MCAgentSecret,
	}
	// get the agent secret and return if it exists
	err := r.Get(context.TODO(), nsn, &secret)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("Failed fetching the agent secret %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}

	// create the local registration secret
	_, err = r.createOrUpdateLocalRegistrationSecret(constants.MCLocalRegistrationSecret, constants.VerrazzanoSystemNamespace)
	if err != nil {
		return err
	}

	return nil
}

// Create or update the secret
func (r *Reconciler) createOrUpdateLocalRegistrationSecret(name string, namespace string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = namespace
	secret.Name = name

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateLocalRegistrationSecret(&secret)
		// Verrrazzano resource cannot own this secret since it is in a different namespace
		// The secret will get deleted when verrazzano-system namespace is deleted.
		return nil
	})
}

// Mutate the secret, setting the kubeconfig data
func (r *Reconciler) mutateLocalRegistrationSecret(secret *corev1.Secret) error {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		constants2.ManagedClusterNameKey: []byte(constants.MCLocalCluster),
	}
	return nil
}
