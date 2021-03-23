// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/report"
	"go.uber.org/zap"
)

// TBD: Overall the intention/design is that we could execute analysis in parallel if we want to do that in the
//      future. So in general analyzers are independent of each other and thread safe, and not expecting to
//      be executed in a particular order.
//      However, there may be special cases where we want an analysis to be done and information gleaned
//      from that analysis to be available to other analyzers. For example, the analysis of the state
//      of Verrazzano is something that is likely to fall into that category. It will make a high level
//      determination of where in the lifecycle we are at, and other analyzers may need to easily get that
//      information to give better guidance on the issues/actions.
//
//      The current implementation is calling the analyzers serially in order.
//      If we do decide to handle analysis in a parallel fashion later, we likely will need to have some
//      analyzers called deterministically in exact order before we fire off other analyzers in parallel.
//      So we may break this into 2 lists in the future: serial analysis functions, parallel analysis functions
//      Analyzers that may fall into this category should be annotated, with a comment, there currently is only
//      one that may require that.
var clusterAnalysisFunctions = map[string]func(log *zap.SugaredLogger, directory string) (err error){
	"Verrazzano Status":  AnalyzeVerrazzano, // Execute first, this may share data other analyzers can use
	"Pod Related Issues": AnalyzePodIssues,
}

// RunAnalysis is the main entry analysis function
func RunAnalysis(log *zap.SugaredLogger, rootDirectory string) (err error) {
	log.Debugf("Cluster Analyzer runAnalysis on %s", rootDirectory)
	clusterRoots, err := files.GetMatchingDirectories(log, rootDirectory, ".*/cluster-dump$")
	if err != nil {
		log.Debugf("Cluster Analyzer runAnalysis failed examining directories for %s", rootDirectory, err)
		return fmt.Errorf("Cluster Analyzer runAnalysis failed examining directories for %s", rootDirectory)
	}
	if len(clusterRoots) == 0 {
		log.Debugf("Cluster Analyzer runAnalysis didn't find any clusters to analyze for %s", rootDirectory)
		return fmt.Errorf("Cluster Analyzer runAnalysis didn't find any clusters to analyze for %s", rootDirectory)
	}

	for _, clusterRoot := range clusterRoots {
		analyzeCluster(log, clusterRoot)
	}

	return nil
}

func analyzeCluster(log *zap.SugaredLogger, clusterRoot string) (err error) {
	log.Debugf("analyzeCluster called for %s", clusterRoot)
	report.AddSourceAnalyzed(clusterRoot)

	for functionName, function := range clusterAnalysisFunctions {
		err := function(log, clusterRoot)
		if err != nil {
			// Log the error and continue on
			log.Errorf("Error processing analysis function %s", functionName, err)
		}
	}

	return nil
}
