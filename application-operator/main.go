// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"github.com/verrazzano/verrazzano/application-operator/controllers/wlsworkload"
	"os"

	"github.com/verrazzano/verrazzano/application-operator/internal/certificates"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	wls "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	clusterscontroller "github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/controllers/cohworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/ingresstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/webhooks"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	// Add core oam types to scheme
	_ = core.AddToScheme(scheme)

	// Add ingress trait to scheme
	_ = vzapi.AddToScheme(scheme)
	_ = istioclinet.AddToScheme(scheme)
	_ = wls.AddToScheme(scheme)

	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = certapiv1alpha2.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

const defaultScraperName = "verrazzano-system/vmi-system-prometheus-0"

var (
	metricsAddr           string
	defaultMetricsScraper string
	certDir               string
	enableLeaderElection  bool
	enableWebhooks        bool
)

func main() {
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&defaultMetricsScraper, "default-metrics-scraper", defaultScraperName,
		"The namespace/deploymentName of the prometheus deployment to be used as the default metrics scraper")
	flag.StringVar(&certDir, "cert-dir", "/etc/certs/", "The directory containing tls.crt and tls.key.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableWebhooks, "enable-webhooks", true,
		"Enable access-controller webhooks")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "5df248b3.verrazzano.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&ingresstrait.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("IngressTrait"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IngressTrait")
		os.Exit(1)
	}
	if err = (&metricstrait.Reconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("controllers").WithName("MetricsTrait"),
		Scheme:  mgr.GetScheme(),
		Scraper: defaultMetricsScraper,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MetricsTrait")
		os.Exit(1)
	}
	if enableWebhooks {
		setupLog.Info("Setting up certificates for webhook")
		caCert, err := certificates.SetupCertificates(certDir)
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
		err = certificates.UpdateMutatingWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			setupLog.Error(err, "unable to update mutating webhook configuration")
			os.Exit(1)
		}
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			setupLog.Error(err, "unable to update validation webhook configuration")
			os.Exit(1)
		}

		if err = (&vzapi.IngressTrait{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "IngressTrait")
			os.Exit(1)
		}
		mgr.GetWebhookServer().CertDir = certDir
		appconfigWebhook := &webhooks.AppConfigWebhook{Defaulters: []webhooks.AppConfigDefaulter{
			&webhooks.MetricsTraitDefaulter{},
			&webhooks.LoggingScopeDefaulter{Client: mgr.GetClient()},
		}}
		mgr.GetWebhookServer().Register(webhooks.AppConfigDefaulterPath, &webhook.Admission{Handler: appconfigWebhook})

	}
	reconciler := loggingscope.NewReconciler(
		mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("LoggingScope"),
		mgr.GetScheme())
	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LoggingScope")
		os.Exit(1)
	}
	if err = (&cohworkload.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("VerrazzanoCoherenceWorkload"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VerrazzanoCoherenceWorkload")
		os.Exit(1)
	}
	if err = (&wlsworkload.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("VerrazzanoWebLogicWorkload"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VerrazzanoWebLogicWorkload")
		os.Exit(1)
	}
	if err = (&clusterscontroller.MultiClusterSecretReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("MultiClusterSecret"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultiClusterSecret")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
