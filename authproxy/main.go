// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	"github.com/verrazzano/verrazzano/authproxy/src/config"
	"github.com/verrazzano/verrazzano/authproxy/src/proxy"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var proxyPort int

func main() {
	handleFlags()
	log := zap.S()

	config.InitConfiguration(log)

	log.Info("Initializing the proxy server")
	authproxy := proxy.InitializeProxy(proxyPort)

	log.Info("Configuring the proxy Kubernetes API client")
	err := proxy.ConfigureKubernetesAPIProxy(authproxy, log)
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
