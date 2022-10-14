// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"github.com/verrazzano/verrazzano/tools/psr/worker"
	"github.com/verrazzano/verrazzano/tools/psr/worker/config"
)

const (
	msgSize = "PSR_MSG_SIZE"
)

type LogGenerator struct{}

var _ worker.Worker = LogGenerator{}

func (w LogGenerator) GetConfigItems() []config.ConfigItem {
	return []config.ConfigItem{
		{msgSize, "20", false}}
}

func (w LogGenerator) Work(config map[string]string) {
}
