// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package buildlog is for build log analysis
package buildlog

import (
	"go.uber.org/zap"
)

// RunAnalysis runs the analysis
func RunAnalysis(log *zap.SugaredLogger, rootDirectory string) (err error) {
	log.Debugf("Build Log Analyzer runAnalysis on %s", rootDirectory)
	return nil
}
