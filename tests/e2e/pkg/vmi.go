// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"

	"gopkg.in/yaml.v2"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	v1 "k8s.io/api/core/v1"
)

// GetSystemVMICredentials - Obtain VMI system credentials
func GetSystemVMICredentials() (*UsernamePassword, error) {
	secret, err := GetSecret("verrazzano-system", "verrazzano")
	if err != nil {
		return nil, err
	}

	username := secret.Data["username"]
	password := secret.Data["password"]
	if username == nil || password == nil {
		return nil, fmt.Errorf("username and password fields required in secret %v", secret)
	}

	return &UsernamePassword{
		Username: string(username),
		Password: string(password),
	}, nil
}

// GetPrometheusConfig - Returns the Prometehus Configmap, Marshalled prometehus.yml and the scrape config list
func GetPrometheusConfig() (*v1.ConfigMap, []interface{}, map[interface{}]interface{}, error) {
	configMap, err := GetConfigMap(vzconst.VmiPromConfigName, vzconst.VerrazzanoSystemNamespace)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed getting configmap: %v", err))
		return nil, nil, nil, err
	}

	prometheusConfig := configMap.Data["prometheus.yml"]
	var configYaml map[interface{}]interface{}
	err = yaml.Unmarshal([]byte(prometheusConfig), &configYaml)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed getting configmap yaml: %v", err))
		return nil, nil, nil, err
	}

	scrapeConfigsData := configYaml["scrape_configs"]
	scrapeConfigs := scrapeConfigsData.([]interface{})
	return configMap, scrapeConfigs, configYaml, nil
}
