// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"github.com/verrazzano/verrazzano/application-operator/internal/operatorinit"
	"os"

	certapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vmc "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	k8sapiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
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
	_ = clisecurity.AddToScheme(scheme)

	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = vmc.AddToScheme(scheme)
	_ = certapiv1.AddToScheme(scheme)
	_ = promoperapi.AddToScheme(scheme)
}

var (
	metricsAddr           string
	defaultMetricsScraper string
	certDir               string
	enableLeaderElection  bool
	enableWebhooks        bool
	runWebhooks           bool
	runWebhookInit        bool
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
	flag.BoolVar(&runWebhooks, "run-webhooks", false,
		"Runs in webhook mode; if false, runs the main operator reconcile loop")
	flag.BoolVar(&runWebhookInit, "run-webhook-init", false,
		"Runs the webhook initialization code")
	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	vzlog.InitLogs(opts)

	// Initialize the zap log
	log := zap.S()

	var exitErr error
	if runWebhookInit {
		exitErr = operatorinit.WebhookInit(certDir, log)
	} else if runWebhooks {
		exitErr = operatorinit.StartWebhookServer(metricsAddr, log, enableLeaderElection, certDir, scheme)
	} else {
		exitErr = operatorinit.StartApplicationOperator(metricsAddr, enableLeaderElection, defaultMetricsScraper, log, scheme)
	}
	if exitErr != nil {
		log.Errorf("Error occurred during execution: %v", exitErr)
		os.Exit(1)
	}
}
