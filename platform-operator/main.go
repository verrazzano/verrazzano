// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/constants"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	clusterscontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/clusters"
	vzcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/internal/certificate"
	config2 "github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/log"
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
	_ = clustersv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {

	// config will hold the entire operator config
	config := config2.Get()

	flag.StringVar(&config.MetricsAddr, "metrics-addr", config.MetricsAddr, "The address the metric endpoint binds to.")
	flag.BoolVar(&config.LeaderElectionEnabled, "enable-leader-election", config.LeaderElectionEnabled,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&config.CertDir, "cert-dir", config.CertDir, "The directory containing tls.crt and tls.key.")
	flag.BoolVar(&config.WebhooksEnabled, "enable-webhooks", config.WebhooksEnabled,
		"Enable webhooks for the operator")
	flag.BoolVar(&config.WebhookValidationEnabled, "enable-webhook-validation", config.WebhookValidationEnabled,
		"Enable webhooks validation for the operator")
	flag.BoolVar(&config.InitWebhooks, "init-webhooks", config.InitWebhooks,
		"Initialize webhooks for the operator")
	flag.StringVar(&config.VerrazzanoInstallDir, "vz-install-dir", config.VerrazzanoInstallDir,
		"Specify the install directory of verrazzano (used for development)")
	flag.StringVar(&config.ThirdpartyChartsDir, "thirdparty-charts-dir", config.ThirdpartyChartsDir,
		"Specify the thirdparty helm charts directory (used for development)")
	flag.StringVar(&config.HelmConfigDir, "helm-config-dir", config.HelmConfigDir,
		"Specify the helm config directory (used for development)")

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	log.InitLogs(opts)

	// Save the config as immutable from this point on.
	config2.Set(config)

	setupLog := zap.S()

	// initWebhooks flag is set when called from an initContainer.  This allows the certs to be setup for the
	// validatingWebhookConfiguration resource before the operator container runs.
	if config.InitWebhooks {
		setupLog.Info("Setting up certificates for webhook")
		caCert, err := certificate.CreateWebhookCertificates(config.CertDir)
		if err != nil {
			setupLog.Errorf("unable to setup certificates for webhook: %v", err)
			os.Exit(1)
		}

		config, err := ctrl.GetConfig()
		if err != nil {
			setupLog.Errorf("unable to get kubeconfig: %v", err)
			os.Exit(1)
		}

		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			setupLog.Errorf("unable to get client: %v", err)
			os.Exit(1)
		}

		setupLog.Info("Updating webhook configuration")
		err = certificate.UpdateValidatingnWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			setupLog.Errorf("unable to update validation webhook configuration: %v", err)
			os.Exit(1)
		}

		return
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: config.MetricsAddr,
		Port:               9443,
		LeaderElection:     config.LeaderElectionEnabled,
		LeaderElectionID:   "3ec4d290.verrazzano.io",
	})
	if err != nil {
		setupLog.Errorf("unable to start manager: %v", err)
		os.Exit(1)
	}

	// Setup the reconciler
	_, dryRun := os.LookupEnv("VZ_DRY_RUN") // If this var is set, the install jobs are no-ops
	reconciler := vzcontroller.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		DryRun: dryRun,
	}
	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Errorf("unable to create controller: %v", err)
		os.Exit(1)
	}

	// Watch for the secondary resource (Job).
	if err := reconciler.Controller.Watch(&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestForOwner{OwnerType: &installv1alpha1.Verrazzano{}, IsController: true}); err != nil {
		setupLog.Errorf("unable to set watch for Job resource: %v", err)
		os.Exit(1)
	}

	// Setup the validation webhook
	if config.WebhooksEnabled {
		setupLog.Info("Setting up Verrazzano webhook with manager")
		if err = (&installv1alpha1.Verrazzano{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Errorf("unable to setup webhook with manager: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().CertDir = config.CertDir
	}

	// Setup the reconciler for VerrazzanoManagedCluster objects
	if err = (&clusterscontroller.VerrazzanoManagedClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VerrazzanoManagedCluster")
		os.Exit(1)
	}

	// Setup the validation webhook
	if config.WebhooksEnabled {
		setupLog.Info("Setting up VerrazzanoManagedCluster webhook with manager")
		if err = (&clustersv1alpha1.VerrazzanoManagedCluster{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Errorf("unable to setup webhook with manager: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().CertDir = config.CertDir
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("Starting thread for syncing multi-cluster objects")
	go mcThread(mgr.GetClient(), setupLog)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Errorf("problem running manager: %v", err)
		os.Exit(1)
	}

}

func mcThread(client client.Client, log *zap.SugaredLogger) {
	// Wait for the existence of the verrazzano-cluster secret.  It contains the credentials
	// for connecting to a managed cluster.
	secret := corev1.Secret{}

	for {
		log.Info("looking for secret")
		err := client.Get(context.TODO(), types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: constants.MCAdminNamespace}, &secret)
		if err != nil {
			time.Sleep(60 * time.Second)
		} else {
			log.Info("found secret")
			break
		}
	}

	// The secret exists
	log.Infof("Found secret named %s in namespace %s", secret.Name, secret.Namespace)

}
