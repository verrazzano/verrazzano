// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"time"
)

const (
	msgSize = "PSR_MSG_SIZE"
)

type LogGenerator struct{}

var _ spi.Worker = LogGenerator{}

func (w LogGenerator) GetConfigItems() []config.ConfigItem {
	return []config.ConfigItem{
		{Key: msgSize, DefaultVal: "20", Required: false}}
}

func (w LogGenerator) Work(config map[string]string, log vzlog.VerrazzanoLogger) {
	for {
		log.Infof("Log Generator Doing Work")
		time.Sleep(10 * time.Second)
	}

}
