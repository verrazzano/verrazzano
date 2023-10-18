// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/capi"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/ociocne"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/oke"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/rancher"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/vmc"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"go.uber.org/zap"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

const (
	clusterSelectorFilePath = "/var/syncClusters/selector.yaml"
	syncClustersEnvVarName  = "CLUSTER_SYNC_ENABLED"
	cattleClustersCRDName   = "clusters.management.cattle.io"
	capiClustersCRDName     = "clusters.cluster.x-k8s.io"
)

type Properties struct {
	Scheme                         *runtime.Scheme
	CertificateDir                 string
	MetricsAddress                 string
	ProbeAddress                   string
	IngressHost                    string
	EnableLeaderElection           bool
	EnableQuickCreate              bool
	DisableCAPIRancherRegistration bool
}

// StartClusterOperator Cluster operator execution entry point
func StartClusterOperator(log *zap.SugaredLogger, props Properties) error {
	options := ctrl.Options{
		Scheme:                 props.Scheme,
		MetricsBindAddress:     props.MetricsAddress,
		Port:                   9443,
		HealthProbeBindAddress: props.ProbeAddress,
		LeaderElection:         props.EnableLeaderElection,
		LeaderElectionID:       "42d5ea87.verrazzano.io",
	}

	ingressNGINXNamespace, err := nginxutil.DetermineNamespaceForIngressNGINX(vzlog.DefaultLogger())
	if err != nil {
		return err
	}
	nginxutil.SetIngressNGINXNamespace(ingressNGINXNamespace)

	ctrlConfig := k8sutil.GetConfigOrDieFromController()
	mgr, err := ctrl.NewManager(ctrlConfig, options)
	if err != nil {
		return errors.Wrapf(err, "Failed to setup controller manager")
	}

	apiextv1Client := apiextv1.NewForConfigOrDie(ctrlConfig)
	crdInstalled, err := isCRDInstalled(apiextv1Client, cattleClustersCRDName)
	if err != nil {
		log.Error(err, "unable to determine if cattle CRD is installed")
		os.Exit(1)
	}

	// only start the Rancher cluster sync controller if the cattle clusters CRD is installed
	if crdInstalled {
		syncEnabled, clusterSelector, err := shouldSyncClusters(clusterSelectorFilePath)
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

	capiCrdInstalled, err := isCRDInstalled(apiextv1Client, capiClustersCRDName)
	if err != nil {
		log.Error(err, "unable to determine if CAPI CRD is installed")
		os.Exit(1)
	}

	if props.IngressHost == "" {
		props.IngressHost = rancherutil.DefaultRancherIngressHostPrefix + nginxutil.IngressNGINXNamespace()
	}

	// only start the CAPI cluster controller if the clusters CRD is installed and the controller is enabled
	if capiCrdInstalled && !props.DisableCAPIRancherRegistration {
		log.Infof("Starting CAPI Cluster controller")
		if err = (&capi.CAPIClusterReconciler{
			Client:             mgr.GetClient(),
			Log:                log,
			Scheme:             mgr.GetScheme(),
			RancherIngressHost: props.IngressHost,
			RancherEnabled:     crdInstalled,
		}).SetupWithManager(mgr); err != nil {
			log.Errorf("Failed to create CAPI cluster controller: %v", err)
			os.Exit(1)
		}
	}

	// Set up the reconciler for VerrazzanoManagedCluster objects
	if err = (&vmc.VerrazzanoManagedClusterReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		RancherIngressHost: props.IngressHost,
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller VerrazzanoManagedCluster")
		os.Exit(1)
	}
	if props.EnableQuickCreate {
		if err = (&ociocne.ClusterReconciler{
			Base: &controller.Base{
				Client: mgr.GetClient(),
			},
			Scheme:            mgr.GetScheme(),
			CredentialsLoader: oci.CredentialsLoaderImpl{},
			OCIClientGetter: func(credentials *oci.Credentials) (oci.Client, error) {
				return oci.NewClient(credentials)
			},
		}).SetupWithManager(mgr); err != nil {
			log.Error(err, "Failed to setup controller OCNEOCIQuickCreate")
			os.Exit(1)
		}
		if err = (&oke.ClusterReconciler{
			Base: &controller.Base{
				Client: mgr.GetClient(),
			},
			CredentialsLoader: oci.CredentialsLoaderImpl{},
			OCIClientGetter: func(credentials *oci.Credentials) (oci.Client, error) {
				return oci.NewClient(credentials)
			},
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			log.Error(err, "Failed to setup controller OKEQuickCreate")
			os.Exit(1)
		}
	}

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

	go startMetricsServer(log)

	log.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
	return nil
}

// shouldSyncClusters returns true if Rancher cluster synchronization is enabled. An optional
// user-specified label selector can be used to filter the Rancher clusters. If sync is enabled and
// the label selector is nil, we will sync all Rancher clusters.
func shouldSyncClusters(clusterSelectorFile string) (bool, *metav1.LabelSelector, error) {
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

// isCRDInstalled returns true if the clusters.management.cattle.io CRD is installed
func isCRDInstalled(client apiextv1.ApiextensionsV1Interface, crdName string) (bool, error) {
	_, err := client.CustomResourceDefinitions().Get(context.TODO(), crdName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
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
		installed, err := isCRDInstalled(client, cattleClustersCRDName)
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

// startMetricsServer initializes the HTTP listener for the metrics server
func startMetricsServer(log *zap.SugaredLogger) {
	// Start up the Prometheus Metrics Exporter server to emit operator metrics
	http.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         ":9100",
	}
	for err := server.ListenAndServe(); err != nil; err = server.ListenAndServe() {
		log.Debugf("Failed to start the metrics server on port 9100: %v", err)
		time.Sleep(10 * time.Second)
	}
}
