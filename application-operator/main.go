// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	certapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	wls "github.com/verrazzano/verrazzano/application-operator/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/appconfig"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterapplicationconfiguration"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclustercomponent"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterconfigmap"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclustersecret"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/verrazzanoproject"
	"github.com/verrazzano/verrazzano/application-operator/controllers/cohworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/containerizedworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/helidonworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/ingresstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingtrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricsbinding"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/namespace"
	"github.com/verrazzano/verrazzano/application-operator/controllers/webhooks"
	"github.com/verrazzano/verrazzano/application-operator/controllers/wlsworkload"
	"github.com/verrazzano/verrazzano/application-operator/internal/certificates"
	"github.com/verrazzano/verrazzano/application-operator/mcagent"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	vmcclient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/scheme"
	"go.uber.org/zap"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	k8sapiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = k8sapiext.AddToScheme(scheme)

	// Add core oam types to scheme
	_ = core.AddToScheme(scheme)

	// Add ingress trait to scheme
	_ = vzapi.AddToScheme(scheme)
	_ = vzapp.AddToScheme(scheme)
	_ = istioclinet.AddToScheme(scheme)
	_ = wls.AddToScheme(scheme)
	_ = clisecurity.AddToScheme(scheme)

	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = certapiv1.AddToScheme(scheme)
	_ = promoperapi.AddToScheme(scheme)
}

var (
	metricsAddr           string
	defaultMetricsScraper string
	certDir               string
	enableLeaderElection  bool
	enableWebhooks        bool
)

func main() {
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&defaultMetricsScraper, "default-metrics-scraper", constants.DefaultScraperName,
		"The namespace/deploymentName of the prometheus deployment to be used as the default metrics scraper")
	flag.StringVar(&certDir, "cert-dir", "/etc/certs/", "The directory containing tls.crt and tls.key.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableWebhooks, "enable-webhooks", true,
		"Enable access-controller webhooks")

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	vzlog.InitLogs(opts)

	// Initialize the zap log
	log := zap.S()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "5df248b3.verrazzano.io",
	})
	if err != nil {
		log.Errorf("Failed to start manager: %v", err)
		os.Exit(1)
	}

	if err = (&ingresstrait.Reconciler{
		Client: mgr.GetClient(),
		Log:    log,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create IngressTrait controller: %v", err)
		os.Exit(1)
	}
	metricsReconciler := &metricstrait.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Scraper: defaultMetricsScraper,
	}

	if err = metricsReconciler.SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create MetricsTrait controller: %v", err)
		os.Exit(1)
	}

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

	if enableWebhooks {
		log.Debug("Setting up certificates for webhook")
		caCert, err := certificates.SetupCertificates(certDir)
		if err != nil {
			log.Errorf("Failed to setup certificates for webhook: %v", err)
			os.Exit(1)
		}

		log.Debug("Updating webhook configurations")
		err = certificates.UpdateMutatingWebhookConfiguration(kubeClient, caCert, certificates.AppConfigMutatingWebhookName)
		if err != nil {
			log.Errorf("Failed to update appconfig mutating webhook configuration: %v", err)
			os.Exit(1)
		}
		err = certificates.UpdateMutatingWebhookConfiguration(kubeClient, caCert, certificates.IstioMutatingWebhookName)
		if err != nil {
			log.Errorf("Failed to update pod mutating webhook configuration: %v", err)
			os.Exit(1)
		}

		// IngressTrait validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.IngressTraitValidatingWebhookName)
		if err != nil {
			log.Errorf("Failed to update IngressTrait validation webhook configuration: %v", err)
			os.Exit(1)
		}
		if err = (&vzapi.IngressTrait{}).SetupWebhookWithManager(mgr); err != nil {
			log.Errorf("Failed to create IngressTrait webhook: %v", err)
			os.Exit(1)
		}

		// VerrazzanoProject validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.VerrazzanoProjectValidatingWebhookName)
		if err != nil {
			log.Errorf("Failed to update verrazzanoproject validation webhook configuration: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-verrazzanoproject",
			&webhook.Admission{Handler: &webhooks.VerrazzanoProjectValidator{}})

		dynamicClient, err := dynamic.NewForConfig(config)
		if err != nil {
			log.Errorf("Failed to create Kubernetes dynamic client: %v", err)
			os.Exit(1)
		}

		istioClientSet, err := istioversionedclient.NewForConfig(config)
		if err != nil {
			log.Errorf("Failed to create istio client: %v", err)
			os.Exit(1)
		}

		// Register a webhook that listens on pods that are running in a istio enabled namespace.
		mgr.GetWebhookServer().Register(
			webhooks.IstioDefaulterPath,
			&webhook.Admission{
				Handler: &webhooks.IstioWebhook{
					Client:        mgr.GetClient(),
					KubeClient:    kubeClient,
					DynamicClient: dynamicClient,
					IstioClient:   istioClientSet,
				},
			},
		)

		// Register the metrics binding mutating webhooks for plain old kubernetes objects workloads
		// The webhooks handle legacy metrics template annotations on these workloads - newer
		// workloads should rely on user-created monitor resources.
		mgr.GetWebhookServer().Register(
			webhooks.MetricsBindingGeneratorWorkloadPath,
			&webhook.Admission{
				Handler: &webhooks.WorkloadWebhook{
					Client:     mgr.GetClient(),
					KubeClient: kubeClient,
				},
			},
		)
		mgr.GetWebhookServer().Register(
			webhooks.MetricsBindingLabelerPodPath,
			&webhook.Admission{
				Handler: &webhooks.LabelerPodWebhook{
					Client:        mgr.GetClient(),
					DynamicClient: dynamicClient,
				},
			},
		)
		err = certificates.UpdateMutatingWebhookConfiguration(kubeClient, caCert, certificates.MetricsBindingWebhookName)
		if err != nil {
			log.Errorf("Failed to update %s mutating webhook configuration: %v", certificates.MetricsBindingWebhookName, err)
			os.Exit(1)
		}

		mgr.GetWebhookServer().CertDir = certDir
		appconfigWebhook := &webhooks.AppConfigWebhook{
			Client:      mgr.GetClient(),
			KubeClient:  kubeClient,
			IstioClient: istioClientSet,
			Defaulters: []webhooks.AppConfigDefaulter{
				&webhooks.MetricsTraitDefaulter{
					Client: mgr.GetClient(),
				},
			},
		}
		mgr.GetWebhookServer().Register(webhooks.AppConfigDefaulterPath, &webhook.Admission{Handler: appconfigWebhook})

		// MultiClusterApplicationConfiguration validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.MultiClusterApplicationConfigurationName)
		if err != nil {
			log.Errorf("Failed to update multiclusterapplicationconfiguration validation webhook configuration: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-multiclusterapplicationconfiguration",
			&webhook.Admission{Handler: &webhooks.MultiClusterApplicationConfigurationValidator{}})

		// MultiClusterComponent validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.MultiClusterComponentName)
		if err != nil {
			log.Errorf("Failed to update multiclusterapplicationconfiguration validation webhook configuration: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-multiclustercomponent",
			&webhook.Admission{Handler: &webhooks.MultiClusterComponentValidator{}})

		// MultiClusterConfigMap validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.MultiClusterConfigMapName)
		if err != nil {
			log.Errorf("Failed to update multiclusterconfigmap validation webhook configuration: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-multiclusterconfigmap",
			&webhook.Admission{Handler: &webhooks.MultiClusterConfigmapValidator{}})

		// MultiClusterSecret validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.MultiClusterSecretName)
		if err != nil {
			log.Errorf("Failed to update multiclustersecret validation webhook configuration: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-multiclustersecret",
			&webhook.Admission{Handler: &webhooks.MultiClusterSecretValidator{}})
	}

	logger, err := vzlog.BuildZapLogger(0)
	if err != nil {
		log.Errorf("Failed to create ApplicationConfiguration logger: %v", err)
		os.Exit(1)
	}
	if err = (&cohworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     logger,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create VerrazzanoCoherenceWorkload controller: %v", err)
		os.Exit(1)
	}
	wlsWorkloadReconciler := &wlsworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}
	if err = wlsWorkloadReconciler.SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create VerrazzanoWeblogicWorkload controller %v", err)
		os.Exit(1)
	}
	if err = (&helidonworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create VerrazzanoHelidonWorkload controller: %v", err)
		os.Exit(1)
	}
	// Setup the namespace reconciler
	if _, err := namespace.NewNamespaceController(mgr, log.With("controller", "VerrazzanoNamespaceController")); err != nil {
		log.Errorf("Failed to create VerrazzanoNamespaceController controller: %v", err)
		os.Exit(1)
	}

	// Create a buffered channel of size 10 for the multi cluster agent to receive messages
	agentChannel := make(chan clusters.StatusUpdateMessage, constants.StatusUpdateChannelBufferSize)

	// Initialize the metricsExporter
	if err := metricsexporter.StartMetricsServer(); err != nil {
		log.Errorf("Failed to create metrics exporter: %v", err)
		os.Exit(1)
	}

	if err = (&multiclustersecret.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterSecretKind, err)
		os.Exit(1)
	}
	if err = (&multiclustercomponent.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterComponentKind, err)
		os.Exit(1)
	}
	if err = (&multiclusterconfigmap.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller %v", clustersv1alpha1.MultiClusterConfigMapKind, err)
		os.Exit(1)
	}
	if err = (&multiclusterapplicationconfiguration.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterAppConfigKind, err)
		os.Exit(1)
	}
	scheme := mgr.GetScheme()
	vmcclient.AddToScheme(scheme)
	if err = (&verrazzanoproject.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       scheme,
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller %v", clustersv1alpha1.VerrazzanoProjectKind, err)
		os.Exit(1)
	}
	if err = (&loggingtrait.LoggingTraitReconciler{
		Client: mgr.GetClient(),
		Log:    log,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create LoggingTrait controller: %v", err)
		os.Exit(1)
	}
	if err = (&appconfig.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create ApplicationConfiguration controller: %v", err)
		os.Exit(1)
	}
	if err = (&containerizedworkload.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create ContainerizedWorkload controller: %v", err)
		os.Exit(1)
	}
	// Register the metrics workload controller
	if err = (&metricsbinding.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create MetricsBinding controller: %v", err)
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	log.Debug("Starting agent for syncing multi-cluster objects")
	go mcagent.StartAgent(mgr.GetClient(), agentChannel, log)

	log.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Errorf("Failed to run manager: %v", err)
		os.Exit(1)
	}
}
