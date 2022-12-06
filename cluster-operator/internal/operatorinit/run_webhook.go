// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"os"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/certificate"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// WebhookInit Webhook init container entry point
func WebhookInit(certDir string, log *zap.SugaredLogger) error {
	log.Debug("Creating certificates used by webhooks")

	conf, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return err
	}

	// Create the webhook certificates and secrets
	if err := certificate.CreateWebhookCertificates(log, kubeClient, certDir); err != nil {
		return err
	}

	return nil
}

func StartWebhookServer(metricsAddr string, probeAddr string, enableLeaderElection bool, certDir string, scheme *runtime.Scheme, log *zap.SugaredLogger) error {
	config, err := ctrl.GetConfig()
	if err != nil {
		log.Errorf("Failed to get kubeconfig: %v", err)
	}

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "42d5ea87.verrazzano.io",
	}

	mgr, err := ctrl.NewManager(config, options)
	if err != nil {
		log.Errorf("Failed to start manager: %v", err)
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Failed to get clientset", err)
		return err
	}

	log.Debug("Updating webhook configuration")
	// VMC validating webhook
	err = updateValidatingWebhookConfiguration(kubeClient, certificate.OperatorName)
	if err != nil {
		log.Errorf("Failed to update VerrazzanoManagedCluster validation webhook configuration: %v", err)
		os.Exit(1)
	}

	// Set up the validation webhook for VMC
	log.Debug("Setting up VerrazzanoManagedCluster webhook with manager")
	if err := (&clustersv1alpha1.VerrazzanoManagedCluster{}).SetupWebhookWithManager(mgr); err != nil {
		log.Errorf("Failed to setup webhook with manager: %v", err)
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	log.Info("Starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Errorf("Failed to run manager: %v", err)
	}

	return err
}
