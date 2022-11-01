// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	cmapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	internalconfig "github.com/verrazzano/verrazzano/platform-operator/internal/config"
	operatorinit "github.com/verrazzano/verrazzano/platform-operator/internal/init"
	"go.uber.org/zap"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
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
	flag.BoolVar(&config.WebhooksEnabled, "webhooks-enabled", config.WebhooksEnabled,
		"Runs in webhook mode; if false, runs the main operator reconcile loop")
	flag.BoolVar(&config.RunWebhookInit, "run-webhook-init", config.RunWebhookInit,
		"Runs the webhook initialization code")
	flag.StringVar(&config.VerrazzanoRootDir, "vz-root-dir", config.VerrazzanoRootDir,
		"Specify the root directory of Verrazzano (used for development)")
	flag.StringVar(&bomOverride, "bom-path", "", "BOM file location")
	flag.BoolVar(&helm.Debug, "helm-debug", helm.Debug, "Add the --debug flag to helm commands")
	flag.Int64Var(&config.HealthCheckPeriodSeconds, "health-check-period", config.HealthCheckPeriodSeconds,
		"Health check period seconds; set to 0 to disable health checks")

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
	//This allows separation of webhooks and operator
	var exitErr error
	if config.RunWebhookInit {
		exitErr = operatorinit.WebhookInit(config, log)
	} else if config.WebhooksEnabled {
		exitErr = operatorinit.StartWebhookServers(config, log, scheme)
	} else {
		exitErr = operatorinit.StartPlatformOperator(config, log, scheme)
	}
	if exitErr != nil {
		log.Errorf("Error occurred during execution: %v", exitErr)
		os.Exit(1)
	}
}
