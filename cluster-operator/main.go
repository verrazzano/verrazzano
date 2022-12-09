// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/operatorinit"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()

	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string
	runWebhooks          bool
	runWebhookInit       bool
	certDir              string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(clustersv1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	handleFlags()
	log := zap.S()

	if runWebhookInit {
		err := operatorinit.WebhookInit(certDir, log)
		if err != nil {
			os.Exit(1)
		}
	} else if runWebhooks {
		err := operatorinit.StartWebhookServer(metricsAddr, probeAddr, enableLeaderElection, certDir, scheme, log)
		if err != nil {
			os.Exit(1)
		}
	} else {
		err := operatorinit.StartClusterOperator(metricsAddr, enableLeaderElection, probeAddr, log, scheme)
		if err != nil {
			os.Exit(1)
		}
	}
}

// handleFlags sets up the CLI flags, parses them, and initializes loggers
func handleFlags() {
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&runWebhooks, "run-webhooks", false,
		"Runs in webhook mode; if false, runs the main operator reconcile loop")
	flag.BoolVar(&runWebhookInit, "run-webhook-init", false,
		"Runs the webhook initialization code")
	flag.StringVar(&certDir, "cert-dir", "/etc/certs/", "The directory containing tls.crt and tls.key.")

	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	kzap.UseFlagOptions(&opts)
	vzlog.InitLogs(opts)
	ctrl.SetLogger(kzap.New(kzap.UseFlagOptions(&opts)))
}
