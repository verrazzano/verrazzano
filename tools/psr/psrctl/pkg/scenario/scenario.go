// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"context"
	"encoding/base64"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/yaml"
)

// FindRunningScenarios returns the list of Scenarios that are running in the cluster.
// Scenario manifests
func (m Manager) FindRunningScenarios() ([]Scenario, error) {
	scenarios := []Scenario{}

	client, err := k8sutil.GetCoreV1Client(m.Log)
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to get CoreV1 client: %v", err)
	}

	// Get the scenario configmaps in the cluster
	req, _ := labels.NewRequirement(LabelScenarioKey, selection.Exists, nil)
	selector := labels.NewSelector()
	selector = selector.Add(*req)

	cms, err := client.ConfigMaps(m.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to find scenario ConfigMaps: %v", err)
	}

	for _, cm := range cms.Items {
		// Load the scenario from the base64
		decoded, err := base64.StdEncoding.DecodeString(cm.Data[LabelScenarioIdKey])
		if err != nil {
			return nil, m.Log.ErrorfNewErr("Failed to decode configmap %s/%s data at key %s: %v", cm.Namespace, cm.Name, LabelScenarioIdKey, err)
		}
		var sc Scenario
		if err := yaml.Unmarshal(decoded, &sc); err != nil {
			return nil, m.Log.ErrorfNewErr("Failed to unmarshal Scenario in configmap %s/%s: %v", cm.Namespace, cm.Name, err)
		}
		scenarios = append(scenarios, sc)
	}

	return scenarios, nil
}
