// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"fmt"
	"helm.sh/helm/v3/pkg/strvals"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
)

// HelmValueFileConstructor creates a YAML file from a set of key value pairs
func HelmValueFileConstructor(kvs []bom.KeyValue) (string, error) {
	strVal := strings.Builder{}
	for _, kv := range kvs {
		strVal.WriteString(fmt.Sprintf("%s=%s,", kv.Key, kv.Value))
	}

	yamlFile, err := strvals.ToYAML(strings.TrimRight(strVal.String(), ","))
	if err != nil {
		return "", err
	}
	return yamlFile, nil
}
