// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	clusterscontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/clusters"
	vzcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/internal/certificate"
	config2 "github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/log"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		err := client.Get(context.TODO(), types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: constants.MCAdminNamespace}, &secret)
		if err != nil {
			time.Sleep(60 * time.Second)
		} else {
			err := validateClusterSecret(&secret)
			if err != nil {
				log.Error("Secret validation failed: %v", err)
			} else {
				break
			}
		}
	}

	// The cluster secret exists
	clusterName := string(secret.Data["cluster-name"])
	log.Infof("Found secret named %s in namespace %s for cluster named %q", secret.Name, secret.Namespace, clusterName)

	// Create the client for accessing the managed cluster
	k8sClient, err := getMCClient(&secret)
	if err != nil {
		log.Errorf("Failed to get the Kubernetes client for cluster %q: %v", clusterName, err)
		return
	}

	// Test listing a resource
	_, err = k8sClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Failed to list namespaces: %v", err)
	}
	log.Info("Successfully listed namespaces")

}

// Validate the cluster secret
func validateClusterSecret(secret *corev1.Secret) error {
	// The secret must contain a cluster-name
	_, ok := secret.Data["cluster-name"]
	if !ok {
		return errors.New(fmt.Sprintf("The secret named %s in namespace %s is missing the required field cluster-name", secret.Name, secret.Namespace))
	}

	// The secret must contain a kubeconfig
	_, ok = secret.Data["kubeconfig"]
	if !ok {
		return errors.New(fmt.Sprintf("The secret named %s in namespace %s is missing the required field kubeconfig", secret.Name, secret.Namespace))
	}

	return nil
}

// Get the Kubernetes client for accessing the managed cluster
func getMCClient(secret *corev1.Secret) (*kubernetes.Clientset, error) {
	// Create a temp file that contains the kubeconfig
	tmpFile, err := ioutil.TempFile("/tmp", "kubeconfig")
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(tmpFile.Name(), secret.Data["kubeconfig"], 0777)
	defer os.Remove(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.BuildConfigFromFlags("", tmpFile.Name())
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
