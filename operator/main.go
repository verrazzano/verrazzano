// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	"github.com/verrazzano/verrazzano/operator/internal/certificates"

	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/v1alpha1"
	"github.com/verrazzano/verrazzano/operator/controllers"
	"github.com/verrazzano/verrazzano/operator/internal/util/log"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = installv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var certDir string
	var enableWebhooks bool

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&certDir, "cert-dir", "/etc/webhook/certs/", "The directory containing tls.crt and tls.key.")
	flag.BoolVar(&enableWebhooks, "enable-webhooks", true,
		"Enable webhooks for the operator")

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	log.InitLogs(opts)

	setupLog := zap.S().Named("operator").Named("setup")

	certificates.CreateCertificates()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "3ec4d290.verrazzano.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup the reconciler
	reconciler := controllers.VerrazzanoReconciler{
		Client: mgr.GetClient(),
		Log:    zap.S().Named("operator"),
		Scheme: mgr.GetScheme(),
	}
	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	// Watch for the secondary resource (Job).
	if err := reconciler.Controller.Watch(&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestForOwner{OwnerType: &installv1alpha1.Verrazzano{}, IsController: true}); err != nil {
		setupLog.Error(err, "unable to set watch for Job resource")
		os.Exit(1)
	}

	// Setup the validation webhook
	if enableWebhooks {
		if err = (&installv1alpha1.Verrazzano{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create validation webhook")
			os.Exit(1)
		}

		mgr.GetWebhookServer().CertDir = certDir
		//		mgr.GetWebhookServer().Register("/validate", &webhook.Admission{Handler: &webhooks.CompomentDefaulter{Log: setupLog}})
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
