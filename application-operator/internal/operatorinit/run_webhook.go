package operatorinit

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/webhooks"
	"github.com/verrazzano/verrazzano/application-operator/internal/certificates"
	"go.uber.org/zap"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// WebhookInit Webhook init container entry point
func WebhookInit(certDir string, log *zap.SugaredLogger) error {
	log.Debug("Creating certificates used by webhooks")

	conf, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return err
	}

	// Create the webhook certificates and secrets
	if err := certificates.CreateWebhookCertificates(log, kubeClient, certDir); err != nil {
		return err
	}

	return nil
}

func StartWebhookServer(metricsAddr string, log *zap.SugaredLogger, enableLeaderElection bool, certDir string, scheme *runtime.Scheme) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "5df248b3.verrazzano.io",
	})
	if err != nil {
		log.Errorf("Failed to start manager: %v", err)
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

	log.Debug("Setting up certificates for webhook")
	err = updateValidatingWebhookConfiguration(kubeClient, certificates.IngressTraitValidatingWebhookName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.IngressTraitValidatingWebhookName, err)
	}

	err = updateValidatingWebhookConfiguration(kubeClient, certificates.MultiClusterSecretName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.MultiClusterSecretName, err)
	}

	err = updateValidatingWebhookConfiguration(kubeClient, certificates.MultiClusterComponentName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.MultiClusterComponentName, err)
	}

	err = updateValidatingWebhookConfiguration(kubeClient, certificates.MultiClusterConfigMapName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.MultiClusterConfigMapName, err)
	}

	err = updateValidatingWebhookConfiguration(kubeClient, certificates.MultiClusterApplicationConfigurationName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.MultiClusterApplicationConfigurationName, err)
	}

	err = updateValidatingWebhookConfiguration(kubeClient, certificates.VerrazzanoProjectValidatingWebhookName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.VerrazzanoProjectValidatingWebhookName, err)
	}

	err = updateMutatingWebhookConfiguration(kubeClient, certificates.IstioMutatingWebhookName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.IstioMutatingWebhookName, err)
	}

	err = updateMutatingWebhookConfiguration(kubeClient, certificates.AppConfigMutatingWebhookName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.AppConfigMutatingWebhookName, err)
	}

	err = updateMutatingWebhookConfiguration(kubeClient, certificates.MetricsBindingWebhookName)
	if err != nil {
		log.Errorf("Failed to update %s: %v", certificates.MetricsBindingWebhookName, err)
	}

	if err = (&vzapi.IngressTrait{}).SetupWebhookWithManager(mgr); err != nil {
		log.Errorf("Failed to create IngressTrait webhook: %v", err)
	}

	// VerrazzanoProject validating webhook
	mgr.GetWebhookServer().Register(
		"/validate-clusters-verrazzano-io-v1alpha1-verrazzanoproject",
		&webhook.Admission{Handler: &webhooks.VerrazzanoProjectValidator{}})

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Failed to create Kubernetes dynamic client: %v", err)
	}

	istioClientSet, err := istioversionedclient.NewForConfig(config)
	if err != nil {
		log.Errorf("Failed to create istio client: %v", err)
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
	mgr.GetWebhookServer().Register(
		"/validate-clusters-verrazzano-io-v1alpha1-multiclusterapplicationconfiguration",
		&webhook.Admission{Handler: &webhooks.MultiClusterApplicationConfigurationValidator{}})

	// MultiClusterComponent validating webhook
	mgr.GetWebhookServer().Register(
		"/validate-clusters-verrazzano-io-v1alpha1-multiclustercomponent",
		&webhook.Admission{Handler: &webhooks.MultiClusterComponentValidator{}})

	// MultiClusterConfigMap validating webhook
	mgr.GetWebhookServer().Register(
		"/validate-clusters-verrazzano-io-v1alpha1-multiclusterconfigmap",
		&webhook.Admission{Handler: &webhooks.MultiClusterConfigmapValidator{}})

	// MultiClusterSecret validating webhook
	mgr.GetWebhookServer().Register(
		"/validate-clusters-verrazzano-io-v1alpha1-multiclustersecret",
		&webhook.Admission{Handler: &webhooks.MultiClusterSecretValidator{}})

	// +kubebuilder:scaffold:builder

	log.Info("Starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Errorf("Failed to run manager: %v", err)
	}

	return err
}
