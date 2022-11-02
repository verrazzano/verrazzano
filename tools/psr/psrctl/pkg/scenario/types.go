// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

//name: OpenSearch-S1
//ID: ops-s1
//description: |
//This is a scenario that writes logs to STDOUT and gets logs from OpenSearch. |
//The purpose of the scenario is to test a simulataneous read/write load on OpenSearch
//logging records.
//usecases:
//- usecasePath: opensearch/getlogs/getlogs.yaml
//scenarioOverride: getlogs-slow.yaml
//description: getlogs from Opensearch with long iteration delay
//- usecasePath: opensearch/getlogs/getlogs.yaml
//scenarioOverride: getlogs-fast.yaml
//description: getlogs from Opensearch with short iteration delay
//- usecasePath: opensearch/writelogs/writelogs.yaml
//scenarioOverride: writelogs.yaml
//description: write logs to STDOUT at default rate
//

type Usecase struct {
	UsecasePath  string
	OverrideFile string
	Description  string
}
type Scenario struct {
	Name        string
	ID          string
	Description string
	Usecases    []Usecase
}
