// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package example had an example worker is used as the default worker when the helm chart is run without specifying a worker
// override file.
package example

import (
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"time"
)

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

const (
	msgSize = "PSR_MSG_SIZE"
)

type ExampleWorker struct{}

var _ spi.Worker = ExampleWorker{}

func (w ExampleWorker) GetConfigItems() []config.ConfigItem {
	return []config.ConfigItem{
		{Key: msgSize, DefaultVal: "20", Required: false}}
}

func (w ExampleWorker) Work(config map[string]string, log vzlog.VerrazzanoLogger) {
	for {
		log.Infof("Example Worker Doing Work")
		time.Sleep(10 * time.Second)
	}

}
