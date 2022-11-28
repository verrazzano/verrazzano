// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/cluster-operator/controllers/vmc"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/certificate"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"go.uber.org/zap"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/rancher"
	// +kubebuilder:scaffold:imports
)

const (
	clusterSelectorFilePath = "/var/syncRancherClusters/selector.yaml"
	syncClustersEnvVarName  = "RANCHER_CLUSTER_SYNC_ENABLED"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(clustersv1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enableWebhooks bool
	var certDir string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableWebhooks, "enable-webhooks", true,
		"Enable webhooks")
	flag.StringVar(&certDir, "cert-dir", "/etc/certs/", "The directory containing tls.crt and tls.key.")

	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	kzap.UseFlagOptions(&opts)
	vzlog.InitLogs(opts)

	log := zap.S()

	ctrl.SetLogger(kzap.New(kzap.UseFlagOptions(&opts)))

	syncEnabled, clusterSelector, err := shouldSyncRancherClusters(clusterSelectorFilePath)
	if err != nil {
		log.Error(err, "error processing cluster sync config")
		os.Exit(1)
	}

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "42d5ea87.verrazzano.io",
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&rancher.RancherClusterReconciler{
		Client:             mgr.GetClient(),
		ClusterSyncEnabled: syncEnabled,
		ClusterSelector:    clusterSelector,
		Log:                log,
		Scheme:             mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create Rancher cluster controller: %v", err)
		os.Exit(1)
	}

	// Set up the reconciler for VerrazzanoManagedCluster objects
	if err = (&vmc.VerrazzanoManagedClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller VerrazzanoManagedCluster")
		os.Exit(1)
	}

	if enableWebhooks {
		mgr.GetWebhookServer().CertDir = certDir
		config, err := ctrl.GetConfig()
		if err != nil {
			log.Errorf("Failed to get kubeconfig: %v", err)
			os.Exit(1)
		}

		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Errorf("Failed to get clientset", err)
			os.Exit(1)
		}

		log.Debug("Setting up certificates for webhook")
		caCert, err := certificate.SetupCertificates(certDir)
		if err != nil {
			log.Errorf("Failed to setup certificates for webhook: %v", err)
			os.Exit(1)
		}
		log.Debug("Updating webhook configuration")
		// VMC validating webhook
		err = certificate.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificate.VerrazzanoManagedClusterValidatingWebhookName)
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

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// shouldSyncRancherClusters returns true if Rancher cluster synchronization is enabled. An optional
// user-specified label selector can be used to filter the Rancher clusters. If sync is enabled and
// the label selector is nil, we will sync all Rancher clusters.
func shouldSyncRancherClusters(clusterSelectorFile string) (bool, *metav1.LabelSelector, error) {
	enabled := os.Getenv(syncClustersEnvVarName)
	if enabled == "" || strings.ToLower(enabled) != "true" {
		return false, nil, nil
	}

	f, err := os.Stat(clusterSelectorFile)
	if err != nil || f.Size() == 0 {
		return true, nil, nil
	}

	b, err := os.ReadFile(clusterSelectorFile)
	if err != nil {
		return true, nil, err
	}

	selector := &metav1.LabelSelector{}
	err = yaml.Unmarshal(b, selector)
	if err != nil {
		return true, nil, err
	}

	return true, selector, err
}
