// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"
	"sync"
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	cmapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	clusterscontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/clusters"
	configmapcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps"
	secretscontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/secrets"
	vzcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/validator"
	internalconfig "github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/netpolicy"
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
	// +kubebuilder:scaffold:imports
)
// A new scheme is created when main.go runs
// A scheme is a struct, it defines methods for 
var scheme = runtime.NewScheme()

func init() {
	//Add to Scheme applies all of the stored function to a schheme
	// A scheme helps with API versioning and converting
	//Ask about kubebuilder things
	_ = clientgoscheme.AddToScheme(scheme)
	_ = vmov1.AddToScheme(scheme)
	_ = installv1alpha1.AddToScheme(scheme)
	_ = clustersv1alpha1.AddToScheme(scheme)

	_ = istioclinet.AddToScheme(scheme)
	_ = istioclisec.AddToScheme(scheme)

	_ = oam.AddToScheme(scheme)

	_ = vzapp.AddToScheme(scheme)

	// Add cert-manager components to the scheme
	_ = cmapiv1.AddToScheme(scheme)

	// Add the Prometheus Operator resources to the scheme
	_ = promoperapi.AddToScheme(scheme)

	// +kubebuilder:scaffold:scheme
}

func main() {

	go func(){
		http.Handle("/metrics", prometheus.Handler())
        http.ListenAndServe(":9100", nil)
	}

	// config will hold the entire operator config
	//GC the operator config is a singleton with specified fields
	//GC goes to another file in internal to get the internal config?
	config := internalconfig.Get()
	var bomOverride string
	//GC Appear to use flags to overset default config values if possible
	//GC the Operator Config has a metrics address 

	flag.StringVar(&config.MetricsAddr, "metrics-addr", config.MetricsAddr, "The address the metric endpoint binds to.")
	// GC appears to be another metrics endpoint
	flag.BoolVar(&config.LeaderElectionEnabled, "enable-leader-election", config.LeaderElectionEnabled,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	// GC only appears to be on active controller at a time
	flag.StringVar(&config.CertDir, "cert-dir", config.CertDir, "The directory containing tls.crt and tls.key.")
	flag.BoolVar(&config.WebhooksEnabled, "enable-webhooks", config.WebhooksEnabled,
		"Enable webhooks for the operator")
	flag.BoolVar(&config.DryRun, "dry-run", config.DryRun, "Run operator in dry run mode.")
	flag.BoolVar(&config.WebhookValidationEnabled, "enable-webhook-validation", config.WebhookValidationEnabled,
		"Enable webhooks validation for the operator")
	flag.BoolVar(&config.InitWebhooks, "init-webhooks", config.InitWebhooks,
		"Initialize webhooks for the operator")
	flag.StringVar(&config.VerrazzanoRootDir, "vz-root-dir", config.VerrazzanoRootDir,
		"Specify the root directory of Verrazzano (used for development)")
	flag.StringVar(&bomOverride, "bom-path", "", "BOM file location")
	flag.BoolVar(&helm.Debug, "helm-debug", helm.Debug, "Add the --debug flag to helm commands")

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	vzlog.InitLogs(opts)
	//GC at this part in the code, the flags are in for logging and the config and it is intialized

	// Save the config as immutable from this point on.
	// GC After this point in the code, the config can not be changed
	internalconfig.Set(config)
	log := zap.S()
	//GC VPO started - maybe put metric here 

	log.Info("Starting Verrazzano Platform Operator")
// GC what does the bom path point to 
	// Set the BOM file path for the operator?
	if len(bomOverride) > 0 {
		log.Infof("Using BOM override file %s", bomOverride)
		internalconfig.SetDefaultBomFilePath(bomOverride)
	}
//GC what is an initContainer?
	// initWebhooks flag is set when called from an initContainer.  This allows the certs to be setup for the
	// validatingWebhookConfiguration resource before the operator container runs.
	if config.InitWebhooks {
		// GC if you are making webhooks this is what sets up the configuration of the certificates
		log.Debug("Creating certificates used by webhooks")
		caCert, err := certificate.CreateWebhookCertificates(config.CertDir)
		if err != nil {
			log.Errorf("Failed to create certificates used by webhooks: %v", err)
			os.Exit(1)
		}
// GC config is redefined to a kubeconfig file 
//GC GetConfig returns a pointer to a kubeconfig
//GC a Kubeconfig file is a file that tells your computer the certificates, server info, it needs to work with a cluster
		config, err := ctrl.GetConfig()
		if err != nil {
			log.Errorf("Failed to get kubeconfig: %v", err)
			os.Exit(1)
		}
// GC A client set, called kubeclient is created 
//GC A kubeclient is what enables communication with the Kuberentes API server (does translation)
		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Errorf("Failed to get clientset: %v", err)
			os.Exit(1)
		}
//GC using the client that we create and the certificate, we tell k8 to update thh webhook configuration
		log.Debug("Updating webhook configuration")
		err = certificate.UpdateValidatingnWebhookConfiguration(kubeClient, caCert)
		if err != nil {
			log.Errorf("Failed to update validation webhook configuration: %v", err)
			os.Exit(1)
		}

//GC returns a new controller runtime client
// GC This controller runtime client takes in the kubeconfig enables reading and writinh 
// GC ASK WHY THERE ARE TWO CLIENTS
//GC Scheme be used to look up versions and such for given types (Scheme is like a dictionary for the client)
//
		client, err := client.New(config, client.Options{})
		if err != nil {
			log.Errorf("Failed to get controller-runtime client: %v", err)
			os.Exit(1)
		}
// GC The network policies for the cluster are created and updated
		log.Debug("Creating or updating network policies")
		_, err = netpolicy.CreateOrUpdateNetworkPolicies(kubeClient, client)
		if err != nil {
			log.Errorf("Failed to create or update network policies: %v", err)
			os.Exit(1)
		}
		//GC Does it end when this thing ends?
		return
	}
//GC Creates a controller runtime manager and ask why a controller-runtme client and manager are needed
//GC Maybe a controller client is just for controller, already apppears to be sending metrics but on a different port
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
// GC What is valid web hook wise
	installv1alpha1.SetComponentValidator(validator.ComponentValidatorImpl{})

	// Setup the reconciler
	reconciler := vzcontroller.Reconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		DryRun:            config.DryRun,
		WatchedComponents: map[string]bool{},
		WatchMutex:        &sync.RWMutex{},
	}
	//GC This is where te Verrazzano controller appears to be defined and setup if it fails, put a metric prehaps
	if err = reconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller", vzlog.FieldController, "Verrazzano")
		os.Exit(1)
	}

	// Setup the validation webhook
	if config.WebhooksEnabled {
		log.Debug("Setting up Verrazzano webhook with manager")
		if err = (&installv1alpha1.Verrazzano{}).SetupWebhookWithManager(mgr, log); err != nil {
			log.Errorf("Failed to setup webhook with manager: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().CertDir = config.CertDir
	}

	// Setup the reconciler for VerrazzanoManagedCluster objects
	if err = (&clusterscontroller.VerrazzanoManagedClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller", vzlog.FieldController, "VerrazzanoManagedCluster")
		os.Exit(1)
	}

	// Setup the validation webhook
	if config.WebhooksEnabled {
		log.Debug("Setting up VerrazzanoManagedCluster webhook with manager")
		if err = (&clustersv1alpha1.VerrazzanoManagedCluster{}).SetupWebhookWithManager(mgr); err != nil {
			log.Errorf("Failed to setup webhook with manager: %v", err)
			os.Exit(1)
		}
		mgr.GetWebhookServer().CertDir = config.CertDir
	}

	// Setup secrets reconciler
	if err = (&secretscontroller.VerrazzanoSecretsReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "Failed to setup controller", vzlog.FieldController, "VerrazzanoSecrets")
		os.Exit(1)
	}

	// Setup configMaps reconciler
	if err = (&configmapcontroller.VerrazzanoConfigMapsReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
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
