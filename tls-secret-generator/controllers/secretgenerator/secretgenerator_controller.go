// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secretgenerator

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/rand"
	"os"
	"time"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8scontroller "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a metrics workload object
type Reconciler struct {
	k8sclient.Client
	Log     *zap.SugaredLogger
	Scheme  *k8sruntime.Scheme
	Scraper string
}

const (
	controllerName = "secretgenerator"

	istioTLSSecret = "istio-certs"
	rootCertFile   = "root-cert.pem"
	certChainFile  = "cert-chain.pem"
	certKeyFile    = "key.pem"
)

// SetupWithManager creates controller for the MetricsBinding
func (r *Reconciler) SetupWithManager(mgr k8scontroller.Manager) error {
	return k8scontroller.NewControllerManagedBy(mgr).For(&promoperapi.Prometheus{}).Complete(r)
}

// Reconcile continuously attempts to generate a secret once it detects a Prometheus
// This will not update the Prometheus, but expects that the secret will be picked up by the created pod
func (r *Reconciler) Reconcile(ctx context.Context, req k8scontroller.Request) (k8scontroller.Result, error) {
	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	if req.Namespace == constants.KubeSystem {
		log := zap.S().With(vzlogInit.FieldResourceNamespace, req.Namespace, vzlogInit.FieldResourceName, req.Name, vzlogInit.FieldController, controllerName)
		log.Infof("Metrics binding resource %v should not be reconciled in kube-system namespace, ignoring", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	certDir := "/etc/istio-certs"
	rootCert, err := os.ReadFile(fmt.Sprintf("%s/root-cert.pem", certDir))
	if err != nil {
		r.Log.Errorf("Failed to read the root certificate file: %v", err)
		return reconcile.Result{}, err
	}
	certChain, err := os.ReadFile(fmt.Sprintf("%s/cert-chain.pem", certDir))
	if err != nil {
		r.Log.Errorf("Failed to read the certificate chain file: %v", err)
		return reconcile.Result{}, err
	}
	key, err := os.ReadFile(fmt.Sprintf("%s/key.pem", certDir))
	if err != nil {
		r.Log.Errorf("Failed to read the certificate key file: %v", err)
		return reconcile.Result{}, err
	}

	tlsSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      istioTLSSecret,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &tlsSecret, func() error {
		tlsSecret.Data = map[string][]byte{
			rootCertFile:  rootCert,
			certChainFile: certChain,
			certKeyFile:   key,
		}
		return nil
	})
	if err != nil {
		r.Log.Errorf("Failed to update the secret: %v", err)
		return reconcile.Result{}, err
	}

	// Requeue often to account for certificate rotation
	seconds := rand.IntnRange(40, 80)
	duration := time.Duration(seconds) * time.Second
	return reconcile.Result{Requeue: true, RequeueAfter: duration}, nil
}
