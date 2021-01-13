// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	"github.com/verrazzano/verrazzano/oam-application-operator/internal/certificates"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	vzapi "github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/oam-application-operator/controllers/ingresstrait"
	"github.com/verrazzano/verrazzano/oam-application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/oam-application-operator/controllers/webhooks"
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

	// +kubebuilder:scaffold:scheme
}

const defaultScraperName = "istio-system/prometheus"

var (
	metricsAddr          string
	metricsScraper       string
	certDir              string
	enableLeaderElection bool
	enableWebhooks       bool
)

func main() {
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&metricsScraper, "metrics-scraper", defaultScraperName, "The metrics scraper")
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
		Scraper: metricsScraper,
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
	// +kubebuilder:scaffold:builder
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
