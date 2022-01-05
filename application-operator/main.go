// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
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
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstemplate"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/webhooks"
	"github.com/verrazzano/verrazzano/application-operator/controllers/wlsworkload"
	"github.com/verrazzano/verrazzano/application-operator/internal/certificates"
	"github.com/verrazzano/verrazzano/application-operator/mcagent"
	"github.com/verrazzano/verrazzano/pkg/log"
	vmcclient "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned/scheme"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	k8sapiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
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

	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = certapiv1alpha2.AddToScheme(scheme)
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

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	log.InitLogs(opts)

	setupLog := ctrl.Log.WithName("operator").WithName("setup")

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

	if enableWebhooks {
		setupLog.Info("Setting up certificates for webhook")
		caCert, err := certificates.SetupCertificates(certDir)
		if err != nil {
			setupLog.Error(err, "unable to setup certificates for webhook")
			os.Exit(1)
		}

		setupLog.Info("Updating webhook configurations")
		err = certificates.UpdateMutatingWebhookConfiguration(kubeClient, caCert, certificates.AppConfigMutatingWebhookName)
		if err != nil {
			setupLog.Error(err, "unable to update appconfig mutating webhook configuration")
			os.Exit(1)
		}
		err = certificates.UpdateMutatingWebhookConfiguration(kubeClient, caCert, certificates.IstioMutatingWebhookName)
		if err != nil {
			setupLog.Error(err, "unable to update pod mutating webhook configuration")
			os.Exit(1)
		}

		// IngressTrait validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.IngressTraitValidatingWebhookName)
		if err != nil {
			setupLog.Error(err, "unable to update ingresstrait validation webhook configuration")
			os.Exit(1)
		}
		if err = (&vzapi.IngressTrait{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "IngressTrait")
			os.Exit(1)
		}

		// VerrazzanoProject validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.VerrazzanoProjectValidatingWebhookName)
		if err != nil {
			setupLog.Error(err, "unable to update verrazzanoproject validation webhook configuration")
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-verrazzanoproject",
			&webhook.Admission{Handler: &webhooks.VerrazzanoProjectValidator{}})

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

		istioClientSet, err := istioversionedclient.NewForConfig(restConfig)
		if err != nil {
			setupLog.Error(err, "Failed to create istio client")
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

		// Register the mutating webhook for plain old kubernetes objects workloads when the object exists
		_, err = kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), certificates.ScrapeGeneratorWebhookName, metav1.GetOptions{})
		if err == nil {
			mgr.GetWebhookServer().Register(
				webhooks.ScrapeGeneratorLoadPath,
				&webhook.Admission{
					Handler: &webhooks.ScrapeGeneratorWebhook{
						Client:     mgr.GetClient(),
						KubeClient: kubeClient,
					},
				},
			)
			err = certificates.UpdateMutatingWebhookConfiguration(kubeClient, caCert, certificates.ScrapeGeneratorWebhookName)
			if err != nil {
				setupLog.Error(err, fmt.Sprintf("unable to update %s mutating webhook configuration", certificates.ScrapeGeneratorWebhookName))
				os.Exit(1)
			}
		}

		mgr.GetWebhookServer().CertDir = certDir
		appconfigWebhook := &webhooks.AppConfigWebhook{
			Client:      mgr.GetClient(),
			KubeClient:  kubeClient,
			IstioClient: istioClientSet,
			Defaulters: []webhooks.AppConfigDefaulter{
				&webhooks.MetricsTraitDefaulter{},
				&webhooks.NetPolicyDefaulter{
					Client:          mgr.GetClient(),
					NamespaceClient: kubeClient.CoreV1().Namespaces(),
				},
			},
		}
		mgr.GetWebhookServer().Register(webhooks.AppConfigDefaulterPath, &webhook.Admission{Handler: appconfigWebhook})

		// MultiClusterApplicationConfiguration validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.MultiClusterApplicationConfigurationName)
		if err != nil {
			setupLog.Error(err, "unable to update multiclusterapplicationconfiguration validation webhook configuration")
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-multiclusterapplicationconfiguration",
			&webhook.Admission{Handler: &webhooks.MultiClusterApplicationConfigurationValidator{}})

		// MultiClusterComponent validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.MultiClusterComponentName)
		if err != nil {
			setupLog.Error(err, "unable to update multiclusterapplicationconfiguration validation webhook configuration")
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-multiclustercomponent",
			&webhook.Admission{Handler: &webhooks.MultiClusterComponentValidator{}})

		// MultiClusterConfigMap validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.MultiClusterConfigMapName)
		if err != nil {
			setupLog.Error(err, "unable to update multiclusterconfigmap validation webhook configuration")
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-multiclusterconfigmap",
			&webhook.Admission{Handler: &webhooks.MultiClusterConfigmapValidator{}})

		// MultiClusterSecret validating webhook
		err = certificates.UpdateValidatingWebhookConfiguration(kubeClient, caCert, certificates.MultiClusterSecretName)
		if err != nil {
			setupLog.Error(err, "unable to update multiclustersecret validation webhook configuration")
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/validate-clusters-verrazzano-io-v1alpha1-multiclustersecret",
			&webhook.Admission{Handler: &webhooks.MultiClusterSecretValidator{}})
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
		Log:          ctrl.Log.WithName("controllers").WithName(clustersv1alpha1.MultiClusterSecretKind),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", clustersv1alpha1.MultiClusterSecretKind)
		os.Exit(1)
	}
	if err = (&multiclustercomponent.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName(clustersv1alpha1.MultiClusterComponentKind),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", clustersv1alpha1.MultiClusterComponentKind)
		os.Exit(1)
	}
	if err = (&multiclusterconfigmap.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName(clustersv1alpha1.MultiClusterConfigMapKind),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", clustersv1alpha1.MultiClusterConfigMapKind)
		os.Exit(1)
	}
	if err = (&multiclusterapplicationconfiguration.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName(clustersv1alpha1.MultiClusterAppConfigKind),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", clustersv1alpha1.MultiClusterAppConfigKind)
		os.Exit(1)
	}
	scheme := mgr.GetScheme()
	vmcclient.AddToScheme(scheme)
	if err = (&verrazzanoproject.Reconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName(clustersv1alpha1.VerrazzanoProjectKind),
		Scheme:       scheme,
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", clustersv1alpha1.VerrazzanoProjectKind)
		os.Exit(1)
	}
	if err = (&loggingtrait.LoggingTraitReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("LoggingTrait"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LoggingTrait")
		os.Exit(1)
	}
	if err = (&appconfig.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ApplicationConfiguration"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApplicationConfiguration")
		os.Exit(1)
	}
	if err = (&containerizedworkload.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ContainerizedWorkload"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ContainerizedWorkload")
		os.Exit(1)
	}
	// Register the metrics workload controller
	_, err = kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), certificates.ScrapeGeneratorWebhookName, metav1.GetOptions{})
	if err == nil {
		if err = (&metricstemplate.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("MetricsTemplate"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "MetricsTemplate")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("Starting agent for syncing multi-cluster objects")
	go mcagent.StartAgent(mgr.GetClient(), agentChannel, ctrl.Log.WithName("multi-cluster").WithName("agent"))

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
