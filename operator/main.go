// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/operator/controllers"
	"github.com/verrazzano/verrazzano/operator/internal/certificates"
	"github.com/verrazzano/verrazzano/operator/internal/util/log"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
	var initWebhooks bool

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&certDir, "cert-dir", "/etc/webhook/certs", "The directory containing tls.crt and tls.key.")
	flag.BoolVar(&enableWebhooks, "enable-webhooks", true,
		"Enable webhooks for the operator")
	flag.BoolVar(&initWebhooks, "init-webhooks", false,
		"Initialize webhooks for the operator")

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	log.InitLogs(opts)

	setupLog := zap.S()

	// initWebhooks flag is set when called from an initContainer.  This allows the certs to be setup for the
	// validatingWebhookConfiguration resource before the operator container runs.
	if initWebhooks {
		setupLog.Info("Setting up certificates for webhook")
		caCert, err := certificates.CreateCertificates(certDir)
		if err != nil {
			setupLog.Error(err, "unable to setup certificates for webhook")
			os.Exit(1)
		}

		config, err := ctrl.GetConfig()
		if err != nil {
			setupLog.Error(err, "unable to get kubeconfig")
			os.Exit(1)
		}

		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			setupLog.Error(err, "unable to get clientset")
			os.Exit(1)
		}

		setupLog.Info("Updating webhook configuration")
		err = certificates.UpdateValidatingnWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			setupLog.Error(err, "unable to update validation webhook configuration")
			os.Exit(1)
		}

		return
	}

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
	_, dryRun := os.LookupEnv("VZ_DRY_RUN") // If this var is set, the install jobs are no-ops
	reconciler := controllers.VerrazzanoReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		DryRun: dryRun,
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
		setupLog.Info("Setting up webhook with manager")
		if err = (&installv1alpha1.Verrazzano{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to setup webhook with manager")
			os.Exit(1)
		}

		mgr.GetWebhookServer().CertDir = certDir
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
