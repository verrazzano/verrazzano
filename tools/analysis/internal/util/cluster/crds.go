// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/json"
	"go.uber.org/zap"
)

// GetCustomResource gets a customResource list
func GetCustomResources(log *zap.SugaredLogger, path string) (customResources interface{}, err error) {
	return json.GetJSONDataFromFile(log, path)
}
