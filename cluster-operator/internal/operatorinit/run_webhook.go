// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/certificate"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// WebhookInit Webhook init container entry point
func WebhookInit(log *zap.SugaredLogger, props Properties) error {
	log.Debug("Creating certificates used by webhooks")

	conf, err := k8sutil.GetConfigFromController()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return err
	}

	// Create the webhook certificates and secrets
	if err := certificate.CreateWebhookCertificates(log, kubeClient, props.CertificateDir); err != nil {
		return err
	}

	return nil
}

func StartWebhookServer(log *zap.SugaredLogger, props Properties) error {
	config, err := k8sutil.GetConfigFromController()
	if err != nil {
		log.Errorf("Failed to get kubeconfig: %v", err)
	}

	options := ctrl.Options{
		Scheme:                 props.Scheme,
		MetricsBindAddress:     props.MetricsAddress,
		Port:                   9443,
		CertDir:                props.CertificateDir,
		HealthProbeBindAddress: props.ProbeAddress,
		LeaderElection:         props.EnableLeaderElection,
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
	// Cluster Operator validating webhook
	err = updateValidatingWebhookConfiguration(kubeClient, certificate.WebhookName)
	if err != nil {
		log.Errorf("Failed to update VerrazzanoManagedCluster validation webhook configuration: %v", err)
		os.Exit(1)
	}
	// Set up VMC Webhook Listener
	log.Debug("Setting up VerrazzanoManagedCluster webhook with manager")
	if err := (&clustersv1alpha1.VerrazzanoManagedCluster{}).SetupWebhookWithManager(mgr); err != nil {
		log.Errorf("Failed to setup VerrazzanoManagedCluster webhook with manager: %v", err)
		os.Exit(1)
	}
	// Set up OCNEOCIQuickCreate Webhook Listener
	log.Debug("Setting up OCNEOCIQuickCreate webhook with manager")
	if err := (&clustersv1alpha1.OCNEOCIQuickCreate{}).SetupWebhookWithManager(mgr); err != nil {
		log.Errorf("Failed to setup OCNEOCIQuickCreate webhook with manager: %v", err)
		os.Exit(1)
	}
	// Set up OCNEOCIQuickCreate Webhook Listener
	log.Debug("Setting up OKEQuickCreate webhook with manager")
	if err := (&clustersv1alpha1.OKEQuickCreate{}).SetupWebhookWithManager(mgr); err != nil {
		log.Errorf("Failed to setup OKEQuickCreate webhook with manager: %v", err)
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
