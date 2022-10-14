// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/workers/config"
)

type Worker interface {
	GetConfigItems() []config.ConfigItem
	Work(config map[string]string, log vzlog.VerrazzanoLogger)
}
