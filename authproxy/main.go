// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"os"

	"github.com/verrazzano/verrazzano/authproxy/src/proxy"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	kubeconfig string
)

func main() {
	handleFlags()
	log := zap.S()

	authproxy := proxy.InitializeProxy()
	err := proxy.ConfigureKubernetesAPIProxy(authproxy, kubeconfig, log)
	if err != nil {
		os.Exit(1)
	}

	err = authproxy.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// handleFlags sets up the CLI flags, parses them, and initializes loggers
func handleFlags() {
	flag.StringVar(&kubeconfig, "kubeconfig-path", "", "Kubeconfig path for the Verrazzano Authproxy cluster.")

	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	kzap.UseFlagOptions(&opts)
	vzlog.InitLogs(opts)
	ctrl.SetLogger(kzap.New(kzap.UseFlagOptions(&opts)))
}
