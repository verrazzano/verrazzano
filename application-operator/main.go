// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"log"
	"os"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	wls "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterapplicationconfiguration"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclustercomponent"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterconfigmap"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterloggingscope"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclustersecret"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/verrazzanoproject"
	"github.com/verrazzano/verrazzano/application-operator/controllers/cohworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/helidonworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/ingresstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/webhooks"
	"github.com/verrazzano/verrazzano/application-operator/controllers/wlsworkload"
	"github.com/verrazzano/verrazzano/application-operator/internal/certificates"
	"github.com/verrazzano/verrazzano/application-operator/mcagent"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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
	_ = platformopclusters.AddToScheme(scheme)
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
	metricsReconciler := &metricstrait.Reconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("controllers").WithName("MetricsTrait"),
		Scheme:  mgr.GetScheme(),
		Scraper: defaultMetricsScraper,
	}

	if err = metricsReconciler.SetupWithManager(mgr); err != nil {
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

		setupLog.Info("Updating webhook configurations")
		err = certificates.UpdateAppConfigMutatingWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			setupLog.Error(err, "unable to update appconfig mutating webhook configuration")
			os.Exit(1)
		}
		err = certificates.UpdateIstioMutatingWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			setupLog.Error(err, "unable to update pod mutating webhook configuration")
			os.Exit(1)
		}

		// IngressTrait validating webhook
		err = certificates.UpdateIngressTraitValidatingWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			setupLog.Error(err, "unable to update ingresstrait validation webhook configuration")
			os.Exit(1)
		}
		if err = (&vzapi.IngressTrait{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "IngressTrait")
			os.Exit(1)
		}

		// VerrazzanoProject validating webhook
		err = certificates.UpdateVerrazzanoProjectValidatingWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			setupLog.Error(err, "unable to update verrazzanoproject validation webhook configuration")
			os.Exit(1)
		}
		if err = (&clustersv1alpha1.VerrazzanoProject{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "VerrazzanoProject")
			os.Exit(1)
		}

		mgr.GetWebhookServer().CertDir = certDir
		appconfigWebhook := &webhooks.AppConfigWebhook{Client: mgr.GetClient(), Defaulters: []webhooks.AppConfigDefaulter{
			&webhooks.MetricsTraitDefaulter{},
			&webhooks.LoggingScopeDefaulter{Client: mgr.GetClient()},
		}}
		mgr.GetWebhookServer().Register(webhooks.AppConfigDefaulterPath, &webhook.Admission{Handler: appconfigWebhook})

		// Get a Kubernetes dynamic client.
		restConfig, err := clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			setupLog.Error(err, "unable to build kube config")
			os.Exit(1)
		}
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			setupLog.Error(err, "unable to create Kubernetes dynamic client")
			os.Exit(1)
		}

		restConfig, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			setupLog.Error(err, "unable to build kube config")
			os.Exit(1)
		}

		clientSet, err := istioversionedclient.NewForConfig(restConfig)
		if err != nil {
			log.Fatalf("Failed to create istio client: %s", err)
		}

		// Register a webhook that listens on pods that are running in a istio enabled namespace.
		mgr.GetWebhookServer().Register(
			webhooks.IstioDefaulterPath,
			&webhook.Admission{
				Handler: &webhooks.IstioWebhook{
					KubeClient:    kubeClient,
					DynamicClient: dynamicClient,
					IstioClient:   clientSet,
				},
			},
		)
	}

	logReconciler := loggingscope.NewReconciler(
		mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("LoggingScope"),
		mgr.GetScheme())
	if err = logReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LoggingScope")
		os.Exit(1)
	}
	if err = (&cohworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("controllers").WithName("VerrazzanoCoherenceWorkload"),
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VerrazzanoCoherenceWorkload")
		os.Exit(1)
	}
	wlsWorkloadReconciler := &wlsworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("controllers").WithName("VerrazzanoWebLogicWorkload"),
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}
	if err = wlsWorkloadReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VerrazzanoWebLogicWorkload")
		os.Exit(1)
	}
	if err = (&helidonworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("controllers").WithName("VerrazzanoHelidonWorkload"),
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VerrazzanoHelidonWorkload")
		os.Exit(1)
	}

	// Create a buffered channel of size 10 for the multi cluster agent to receive messages
	agentChannel := make(chan clusters.StatusUpdateMessage, constants.StatusUpdateChannelBufferSize)

	if err = (&multiclustersecret.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("MultiClusterSecret"),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultiClusterSecret")
		os.Exit(1)
	}
	if err = (&multiclustercomponent.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("MultiClusterComponent"),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultiClusterComponent")
		os.Exit(1)
	}
	if err = (&multiclusterconfigmap.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("MultiClusterConfigMap"),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultiClusterConfigMap")
		os.Exit(1)
	}
	if err = (&multiclusterloggingscope.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("MultiClusterLoggingScope"),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultiClusterLoggingScope")
		os.Exit(1)
	}
	if err = (&multiclusterapplicationconfiguration.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("MultiClusterApplicationConfiguration"),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultiClusterApplicationConfiguration")
		os.Exit(1)
	}
	if err = (&verrazzanoproject.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("VerrazzanoProject"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VerrazzanoProject")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("Starting agent for syncing multi-cluster objects")
	go mcagent.StartAgent(mgr.GetClient(), agentChannel, ctrl.Log.WithName("multi-cluster agent"))

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
