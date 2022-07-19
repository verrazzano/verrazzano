// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"strings"
)

const (
	jaegerServiceIndexPrefix = "verrazzano-jaeger-jaeger-service"
	jaegerSpanIndexPrefix    = "verrazzano-jaeger-jaeger-span"
)

func VerifyJaegerSpans(service string) bool {
	return false
}

func IsJaegerInstanceCreated() (bool, error) {
	deployments, err := ListDeployments(constants.VerrazzanoMonitoringNamespace)
	if err != nil {
		return false, err
	}
	if len(deployments.Items) > 0 {
		return true, nil
	}
	return false, nil
}

func GetJaegerIndicesInElasticSearch(kubeconfigPath string) []string {
	jaegerIndices := []string{}
	for _, indexName := range listSystemElasticSearchIndices(kubeconfigPath) {
		if strings.HasPrefix(indexName, jaegerServiceIndexPrefix) ||
			strings.HasPrefix(indexName, jaegerSpanIndexPrefix) {
			jaegerIndices = append(jaegerIndices, indexName)
		}
	}
	return jaegerIndices
}
