// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/json"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	"os"
)

// AnalyzeKontainerDrivers handles the checking af the status of KontainerDriver resource.
func AnalyzeKontainerDrivers(log *zap.SugaredLogger, clusterRoot string) error {
	log.Debugf("AnalyzeKontainerDrivers called for %s", clusterRoot)

	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}

	return analyzeKontainerDrivers(log, clusterRoot, &issueReporter)
}

// analyzeKontainerDrivers handles the checking af the status of KontainerDriver resource.
func analyzeKontainerDrivers(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) error {

	kontainerDriverPath := files.FindFileInClusterRoot(clusterRoot, "default/kontainerdriver.json")
	_, err := os.Stat(kontainerDriverPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		log.Errorf("failed to access file %s: %s", kontainerDriverPath, err.Error())
		return err
	}

	// Get the JSON from the kontainerdriver.json file
	jsonData, err := json.GetJSONDataFromFile(log, kontainerDriverPath)
	if err != nil {
		log.Errorf("failed to get JSON data from file %s: %s", kontainerDriverPath, err.Error())
		return err
	}

	// Get the list of kontainer drivers
	drivers, err := json.GetJSONValue(log, jsonData, "items")
	if err != nil {
		log.Errorf("failed to get the list of kontainer drivers: %s", err.Error())
		return err
	}

	for _, driver := range drivers.([]interface{}) {
		err = reportKontainerDriverIssue(log, clusterRoot, driver, issueReporter)
		if err != nil {
			return err
		}
	}

	issueReporter.Contribute(log, clusterRoot)

	return nil
}

// reportKontainerDriverIssue will check the ociocneengine and oraclecontainerengine KontainerDriver resources and
// report any issues that are found with them
func reportKontainerDriverIssue(log *zap.SugaredLogger, clusterRoot string, driver interface{}, issueReporter *report.IssueReporter) error {
	name, err := json.GetJSONValue(log, driver, "metadata.name")
	if err != nil {
		log.Errorf("failed to get metadata.name of a kontainer driver: %s", err.Error())
		return err
	}

	var messages []string
	if name != nil && name == "ociocneengine" || name == "oraclecontainerengine" {
		conditions, err := json.GetJSONValue(log, driver, "status.conditions")
		if err != nil {
			log.Errorf("failed to get the list of status conditions for kontainer driver %s: %s", name, err.Error())
			return err
		}
		for _, condition := range conditions.([]interface{}) {
			condType, err := json.GetJSONValue(log, condition, "type")
			if err != nil {
				log.Errorf("failed to get the condition type of a kontainer driver %s: %s", name, err.Error())
				return err
			}
			if condType != nil {
				var condStatus interface{}
				condTypeValue := condType.(string)
				switch condTypeValue {
				case "Active", "Downloaded", "Installed":
					condStatus, err = json.GetJSONValue(log, condition, "status")
					if err != nil {
						log.Errorf("failed to get the condition status of a kontainer driver %s: %s", name, err.Error())
						return err
					}
				}
				if condStatus != nil {
					condStatusValue := condStatus.(string)
					if condStatusValue != "" && condStatusValue != "True" {
						messages = append(messages, fmt.Sprintf(" - condition type \"%s\" has a status of \"%s\"", condTypeValue, condStatusValue))
					}
				}
			}
		}

		if len(messages) != 0 {
			messages = append([]string{fmt.Sprintf("KontainerDriver resource \"%s\" is not ready per it's resource status", name)}, messages...)
			issueReporter.AddKnownIssueMessagesFiles(report.KontainerDriverNotReady, clusterRoot, messages, []string{})
		}
	}

	return nil
}
