// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterissuers

import (
	"context"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/secrets"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	verrazzanoClusterIssuerName = "verrazzano-cluster-issuer"
)

type Reconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	log           vzlog.VerrazzanoLogger
	StatusUpdater vzstatus.Updater
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	clusterIssuer := &unstructured.Unstructured{}
	clusterIssuer.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "ClusterIssuer",
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(clusterIssuer).
		Complete(r)
}

// Reconcile the Verrazzano ClusterIssuer
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	zap.S().Infof("ClusterIssuersReconciler to reconcile %s", req.Name)

	// Only reconcile the Verrazzano ClusterIssuer
	if req.Name != verrazzanoClusterIssuerName {
		return ctrl.Result{}, nil
	}
	if ctx == nil {
		ctx = context.TODO()
	}

	// Get the Verrazzano custom resource
	vzList := &installv1alpha1.VerrazzanoList{}
	err := r.List(ctx, vzList)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Verrazzano resource: %v", err)
		return newRequeueWithDelay(), err
	}
	if vzList != nil && len(vzList.Items) > 0 {
		vz := &vzList.Items[0]
		// Nothing to do if the vz resource is being deleted, or if the ClusterIssuer is empty
		if vz.DeletionTimestamp != nil || vz.Spec.Components.ClusterIssuer == nil {
			return ctrl.Result{}, nil
		}

		// Get the effective CR
		effectiveCR, err := transform.GetEffectiveCR(vz)
		if err != nil {
			r.log.Errorf("Failed to get the effective CR for %s/%s: %s", vz.Namespace, vz.Name, err.Error())
			return newRequeueWithDelay(), err
		}

		// Run the secrets-controller reconcile on the ClusterIssuer secret.  The following sequence of events would not
		// be caught by the secrets controller:
		//   - Start with self-signed certs
		//   - Create a custom CA secret
		//   - Update the ClusterIssuer configuration to use the custom CA
		//   - The secrets controller will not be notified about a change in the custom CA since the change was before it
		//     was configured as a ClusterIssuer secret.
		clusterIssuer := effectiveCR.Spec.Components.ClusterIssuer
		isCA, err := clusterIssuer.IsCAIssuer()
		if err == nil && isCA {
			clusterIssuerSecret := types.NamespacedName{Namespace: clusterIssuer.ClusterResourceNamespace, Name: clusterIssuer.CA.SecretName}
			secretsReconciler := &secrets.VerrazzanoSecretsReconciler{
				Client:        r.Client,
				Log:           r.log,
				Scheme:        r.Scheme,
				StatusUpdater: r.StatusUpdater,
			}
			return secretsReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: clusterIssuerSecret})
		}
	}

	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}
