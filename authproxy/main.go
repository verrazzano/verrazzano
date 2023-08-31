// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	"github.com/verrazzano/verrazzano/authproxy/src/config"
	"github.com/verrazzano/verrazzano/authproxy/src/proxy"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var proxyPort int

func main() {
	handleFlags()
	log := zap.S()

	config.InitConfiguration(log)

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	opts := ctrl.Options{
		Scheme: scheme,
	}
	// create a controller manager in order to create a K8S in-cluster client
	mgr, err := ctrl.NewManager(k8sutil.GetConfigOrDieFromController(), opts)

	log.Info("Initializing the proxy server")
	authproxy := proxy.InitializeProxy(proxyPort, mgr.GetClient())

	log.Info("Configuring the proxy Kubernetes API client")
	err = proxy.ConfigureKubernetesAPIProxy(authproxy, log)
	if err != nil {
		os.Exit(1)
	}

	log.Info("Starting up proxy server to listen for requests")
	err = authproxy.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}

// handleFlags sets up the CLI flags, parses them, and initializes loggers
func handleFlags() {
	flag.IntVar(&proxyPort, "proxy-port", 8777, "Port for incoming request to the Auth Proxy.")

	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	kzap.UseFlagOptions(&opts)
	vzlog.InitLogs(opts)
	ctrl.SetLogger(kzap.New(kzap.UseFlagOptions(&opts)))
}
