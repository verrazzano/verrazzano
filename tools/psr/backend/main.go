// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	vzlog2 "github.com/verrazzano/verrazzano/pkg/log"
	vzlog "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/workmanager"
)

func main() {
	vzlog2.InitLogs(kzap.Options{})
	log := vzlog.DefaultLogger()
	log.Info("Starting PSR backend")

	// Run the worker forever or until it quits
	err := workmanager.StartWorkerRunners(log)
	if err != nil {
		log.Error("Failed running worker: %v", err)
	}
	log.Info("Stopping PSR backend")
}
