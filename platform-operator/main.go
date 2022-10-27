// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	vzcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/reconcile"
	"os"
	"sync"
	"time"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/webhooks"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/status"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/validator"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	cmapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"

	"go.uber.org/zap"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/verrazzano/verrazzano/pkg/helm"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	clusterscontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/clusters"
	configmapcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps"
	secretscontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/secrets"
	internalconfig "github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/netpolicy"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	// +kubebuilder:scaffold:imports
)

var scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = vmov1.AddToScheme(scheme)
	_ = installv1alpha1.AddToScheme(scheme)
	_ = installv1beta1.AddToScheme(scheme)
	_ = clustersv1alpha1.AddToScheme(scheme)

	_ = istioclinet.AddToScheme(scheme)
	_ = istioclisec.AddToScheme(scheme)

	_ = oam.AddToScheme(scheme)

	_ = vzapp.AddToScheme(scheme)
	_ = vzappclusters.AddToScheme(scheme)

	// Add cert-manager components to the scheme
	_ = cmapiv1.AddToScheme(scheme)

	// Add the Prometheus Operator resources to the scheme
	_ = promoperapi.AddToScheme(scheme)

	// Add K8S api-extensions so that we can list CustomResourceDefinitions during uninstall of VZ
	_ = v1.AddToScheme(scheme)
	utilruntime.Must(installv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

var healthCheckPeriodSeconds int64

func main() {

	// config will hold the entire operator config
	config := internalconfig.Get()
	var bomOverride string

	flag.StringVar(&config.MetricsAddr, "metrics-addr", config.MetricsAddr, "The address the metric endpoint binds to.")
	flag.BoolVar(&config.LeaderElectionEnabled, "enable-leader-election", config.LeaderElectionEnabled,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&config.CertDir, "cert-dir", config.CertDir, "The directory containing tls.crt and tls.key.")
	flag.BoolVar(&config.DryRun, "dry-run", config.DryRun, "Run operator in dry run mode.")
	flag.BoolVar(&config.WebhookValidationEnabled, "enable-webhook-validation", config.WebhookValidationEnabled,
		"Enable webhooks validation for the operator")
	flag.BoolVar(&config.InitWebhooks, "init-webhooks", config.InitWebhooks,
		"Initialize webhooks for the operator")
	flag.StringVar(&config.VerrazzanoRootDir, "vz-root-dir", config.VerrazzanoRootDir,
		"Specify the root directory of Verrazzano (used for development)")
	flag.StringVar(&bomOverride, "bom-path", "", "BOM file location")
	flag.BoolVar(&helm.Debug, "helm-debug", helm.Debug, "Add the --debug flag to helm commands")
	// Set to 0 to disable the health check routine.
	flag.Int64Var(&healthCheckPeriodSeconds, "health-check-period", 60, "Health check period seconds")

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	vzlog.InitLogs(opts)

	// Save the config as immutable from this point on.
	internalconfig.Set(config)
	log := zap.S()

	log.Info("Starting Verrazzano Platform Operator")

	// Set the BOM file path for the operator
	if len(bomOverride) > 0 {
		log.Infof("Using BOM override file %s", bomOverride)
		internalconfig.SetDefaultBomFilePath(bomOverride)
	}

	if !validators.IsKubernetesVersionSupported() {
		os.Exit(1)
	}

	// Log the Verrazzano version
	version, err := validators.GetCurrentBomVersion()
	if err == nil {
		log.Infof("Verrazzano version: %s", version.ToString())
	} else {
		log.Errorf("Failed to get the Verrazzano version from the BOM: %v", err)
	}

	registry.InitRegistry()
	// initWebhooks flag is set when called from an webhook deployment.  This allows separation of webhooks and operator
	if config.InitWebhooks {
		startWebhookServers(config, log)
	} else {
		reconcilePlatformOperator(config, log)
	}
}

// startWebhookServers uses the same image as operator, but configured only to run Webhooks for the platform.
func startWebhookServers(config internalconfig.OperatorConfig, log *zap.SugaredLogger) {
	log.Debug("Creating certificates used by webhooks")
	caCert, err := certificate.CreateWebhookCertificates(config.CertDir)
	if err != nil {
		log.Errorf("Failed to create certificates used by webhooks: %v", err)
		os.Exit(1)
	}

	conf, err := ctrl.GetConfig()
	if err != nil {
		log.Errorf("Failed to get kubeconfig: %v", err)
		os.Exit(1)
	}

	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Errorf("Failed to get clientset: %v", err)
		os.Exit(1)
	}

	dynamicClient, err := dynamic.NewForConfig(conf)
	if err != nil {
		log.Errorf("Failed to create Kubernetes dynamic client: %v", err)
		os.Exit(1)
	}

	log.Debug("Delete old VPO webhook configuration")
	err = certificate.DeleteValidatingWebhookConfiguration(kubeClient, certificate.OperatorName)
	if err != nil {
		log.Errorf("Failed to delete old webhook configuration: %v", err)
		os.Exit(1)
	}

	log.Debug("Updating VPO webhook configuration")
	err = certificate.UpdateValidatingnWebhookConfiguration(kubeClient, caCert, certificate.OperatorName)
	if err != nil {
		log.Errorf("Failed to update validation webhook configuration: %v", err)
		os.Exit(1)
	}

	log.Debug("Updating conversion webhook")
	apixClient, err := apiextensionsv1client.NewForConfig(conf)
	if err != nil {
		log.Errorf("Failed to get apix clientset: %v", err)
		os.Exit(1)
	}
	err = certificate.UpdateConversionWebhookConfiguration(apixClient, caCert)
	if err != nil {
		log.Errorf("Failed to update conversion webhook: %v", err)
		os.Exit(1)
	}

	err = certificate.UpdateMutatingWebhookConfiguration(kubeClient, caCert, constants.MysqlBackupMutatingWebhookName)
	if err != nil {
		log.Errorf("Failed to update pod mutating webhook configuration: %v", err)
		os.Exit(1)
	}

	log.Debug("Updating MySQL install values webhook configuration")
	err = certificate.UpdateValidatingnWebhookConfiguration(kubeClient, caCert, webhooks.MysqlInstallValuesWebhook)
	if err != nil {
		log.Errorf("Failed to update validation webhook configuration: %v", err)
		os.Exit(1)
	}

	c, err := client.New(conf, client.Options{})
	if err != nil {
		log.Errorf("Failed to get controller-runtime client: %v", err)
		os.Exit(1)
	}

	log.Debug("Creating or updating network policies")
	var errors []error
	_, errors = netpolicy.CreateOrUpdateNetworkPolicies(kubeClient, c)
	if len(errors) < 0 {
		log.Errorf("Failed to create or update network policies: %v", err)
		os.Exit(1)
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: config.MetricsAddr,
		Port:               9443,
		LeaderElection:     config.LeaderElectionEnabled,
		LeaderElectionID:   "3ec4d295.verrazzano.io",
	})
	if err != nil {
		log.Errorf("Failed to create a controller-runtime manager: %v", err)
		os.Exit(1)
	}

	installv1alpha1.SetComponentValidator(validator.ComponentValidatorImpl{})
	installv1beta1.SetComponentValidator(validator.ComponentValidatorImpl{})

	// Setup the validation webhook
	log.Debug("Setting up Verrazzano webhook with manager")
	if err = (&installv1alpha1.Verrazzano{}).SetupWebhookWithManager(mgr, log); err != nil {
		log.Errorf("Failed to setup install.v1alpha1.Verrazzano webhook with manager: %v", err)
		os.Exit(1)
	}
	if err = (&installv1beta1.Verrazzano{}).SetupWebhookWithManager(mgr, log); err != nil {
		log.Errorf("Failed to setup install.v1beta1.Verrazzano webhook with manager: %v", err)
		os.Exit(1)
	}

	// register MySQL backup job mutating webhook
	mgr.GetWebhookServer().Register(
		constants.MysqlBackupMutatingWebhookPath,
		&webhook.Admission{
			Handler: &webhooks.MySQLBackupJobWebhook{
				Client:        mgr.GetClient(),
				KubeClient:    kubeClient,
				DynamicClient: dynamicClient,
				Defaulters:    []webhooks.MySQLDefaulter{},
			},
		},
	)

	// register MySQL install values webhooks
	mgr.GetWebhookServer().Register(webhooks.MysqlInstallValuesV1beta1path, &webhook.Admission{Handler: &webhooks.MysqlValuesValidatorV1beta1{}})
	mgr.GetWebhookServer().Register(webhooks.MysqlInstallValuesV1alpha1path, &webhook.Admission{Handler: &webhooks.MysqlValuesValidatorV1alpha1{}})

	// Set up the validation webhook for VMC
	log.Debug("Setting up VerrazzanoManagedCluster webhook with manager")
	if err = (&clustersv1alpha1.VerrazzanoManagedCluster{}).SetupWebhookWithManager(mgr); err != nil {
		log.Errorf("Failed to setup webhook with manager: %v", err)
		os.Exit(1)
	}
	mgr.GetWebhookServer().CertDir = config.CertDir

	// +kubebuilder:scaffold:builder
	log.Info("Starting controller-runtime manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Errorf("Failed starting controller-runtime manager: %v", err)
		os.Exit(1)
	}

}

// reconcilePlatformOperator runs the Verrazzano Platform Operator without running webhooks servers
func reconcilePlatformOperator(config internalconfig.OperatorConfig, log *zap.SugaredLogger) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: config.MetricsAddr,
		Port:               9443,
		LeaderElection:     config.LeaderElectionEnabled,
		LeaderElectionID:   "3ec4d290.verrazzano.io",
	})
	if err != nil {
		log.Errorf("Failed to create a controller-runtime manager: %v", err)
		os.Exit(1)
	}

	metricsexporter.StartMetricsServer(log)

	// Set up the reconciler
	statusUpdater := vzstatus.NewStatusUpdater(mgr.GetClient())
	healthCheck := vzstatus.NewHealthChecker(statusUpdater, mgr.GetClient(), time.Duration(healthCheckPeriodSeconds)*time.Second)
	reconciler := vzcontroller.Reconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		DryRun:            config.DryRun,
		WatchedComponents: map[string]bool{},
		WatchMutex:        &sync.RWMutex{},
		StatusUpdater:     statusUpdater,
	}
	if err = reconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller", vzlog.FieldController, "Verrazzano")
		os.Exit(1)
	}
	if healthCheckPeriodSeconds > 0 {
		healthCheck.Start()
	}

	// Set up the reconciler for VerrazzanoManagedCluster objects
	if err = (&clusterscontroller.VerrazzanoManagedClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller", vzlog.FieldController, "VerrazzanoManagedCluster")
		os.Exit(1)
	}

	// Start the goroutine to sync Rancher clusters and VerrazzanoManagedCluster objects
	rancherClusterSyncer := &clusterscontroller.RancherClusterSyncer{
		Client: mgr.GetClient(),
		Log:    log,
	}
	go rancherClusterSyncer.StartSyncing()

	// Setup secrets reconciler
	if err = (&secretscontroller.VerrazzanoSecretsReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		StatusUpdater: statusUpdater,
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller", vzlog.FieldController, "VerrazzanoSecrets")
		os.Exit(1)
	}

	// Setup configMaps reconciler
	if err = (&configmapcontroller.VerrazzanoConfigMapsReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		StatusUpdater: statusUpdater,
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller", vzlog.FieldController, "VerrazzanoConfigMaps")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder
	log.Info("Starting controller-runtime manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Errorf("Failed starting controller-runtime manager: %v", err)
		os.Exit(1)
	}
}
