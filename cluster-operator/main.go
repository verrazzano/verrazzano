// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/vmc"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/certificate"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	// +kubebuilder:scaffold:imports
)

const (
	clusterSelectorFilePath = "/var/syncRancherClusters/selector.yaml"
	syncClustersEnvVarName  = "RANCHER_CLUSTER_SYNC_ENABLED"
	cattleClustersCRDName   = "clusters.management.cattle.io"
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

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "42d5ea87.verrazzano.io",
	}

	config := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(config, options)
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	apiextv1Client := apiextv1.NewForConfigOrDie(config)
	crdInstalled, err := isCattleClustersCRDInstalled(apiextv1Client)
	if err != nil {
		log.Error(err, "unable to determine if cattle CRD is installed")
		os.Exit(1)
	}

	// only start the Rancher cluster sync controller if the cattle clusters CRD is installed
	if crdInstalled {
		syncEnabled, clusterSelector, err := shouldSyncRancherClusters(clusterSelectorFilePath)
		if err != nil {
			log.Error(err, "error processing cluster sync config")
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

	// wrap the controller context with a new context so we can cancel the context if we detect
	// a change in the clusters.management.cattle.io CRD installation
	ctx, cancel := context.WithCancel(ctrl.SetupSignalHandler())
	go watchCattleClustersCRD(cancel, apiextv1Client, crdInstalled, log)

	log.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
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

// isCattleClustersCRDInstalled returns true if the clusters.management.cattle.io CRD is installed
func isCattleClustersCRDInstalled(client apiextv1.ApiextensionsV1Interface) (bool, error) {
	_, err := client.CustomResourceDefinitions().Get(context.TODO(), cattleClustersCRDName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// watchCattleClustersCRD periodically checks to see if the clusters.management.cattle.io CRD is installed. If it detects a change
// it will call the context cancel function which will cause the operator to gracefully shut down. The operator will then be
// restarted by Kubernetes and it will start the cattle clusters sync controller if the CRD is installed.
func watchCattleClustersCRD(cancel context.CancelFunc, client apiextv1.ApiextensionsV1Interface, crdInstalled bool, log *zap.SugaredLogger) {
	log.Infof("Watching for CRD %s to be installed or uninstalled", cattleClustersCRDName)
	for {
		installed, err := isCattleClustersCRDInstalled(client)
		if err != nil {
			log.Debugf("Unable to determine if CRD %s is installed: %v", cattleClustersCRDName, err)
			continue
		}
		if installed != crdInstalled {
			log.Infof("Detected CRD %s was installed or uninstalled, shutting down operator", cattleClustersCRDName)
			cancel()
			return
		}
		time.Sleep(10 * time.Second)
	}
}
