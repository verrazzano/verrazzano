// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"context"
	"encoding/base64"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/yaml"
)

// createConfigMap creates a ConfigMap with the scenario data
func (m Manager) createConfigMap(scenario Scenario) (*corev1.ConfigMap, error) {
	y, err := yaml.Marshal(scenario)
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to marshal scenario to YAML: %v", err)
	}
	// convert to base64
	encoded := base64.StdEncoding.EncodeToString(y)

	cmIn := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildConfigmapName(scenario.ID),
			Namespace: m.Namespace,
			Labels: map[string]string{
				LabelScenario:   "true",
				LabelScenarioID: scenario.ScenarioManifest.ID},
		},
		Data: map[string]string{"scenario": encoded},
	}

	cmNew, err := m.Client.ConfigMaps(m.Namespace).Create(context.TODO(), &cmIn, metav1.CreateOptions{})
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to create scenario ConfigMap %s/%s: %v", cmIn.Namespace, cmIn.Name, err)
	}
	return cmNew, nil
}

// deleteConfigMap deletes a ConfigMap
func (m Manager) deleteConfigMap(cm *corev1.ConfigMap) error {
	err := m.Client.ConfigMaps(cm.Namespace).Delete(context.TODO(), cm.Name, metav1.DeleteOptions{})
	if err != nil {
		return m.Log.ErrorfNewErr("Failed to delete scenario ConfigMap %s/%s: %v", cm.Namespace, cm.Name, err)
	}
	return nil
}

// getAllConfigMaps gets all the configmaps that have scenario information
func (m Manager) getAllConfigMaps() ([]corev1.ConfigMap, error) {
	req, _ := labels.NewRequirement(LabelScenario, selection.Exists, nil)
	return m.getConfigMapsByLabels(*req)
}

// getConfigMapByID gets the configmap that matches a specific Scenario ID
func (m Manager) getConfigMapByID(ID string) (*corev1.ConfigMap, error) {
	// Find the scenario configmaps in the cluster
	req1, _ := labels.NewRequirement(LabelScenario, selection.Exists, nil)
	req2, _ := labels.NewRequirement(LabelScenarioID, selection.Equals, []string{ID})
	cms, err := m.getConfigMapsByLabels(*req1, *req2)
	if err != nil {
		return nil, err
	}
	if len(cms) == 0 {
		return nil, fmt.Errorf("Failed to find ConfigMap for scenario with ID %s", ID)
	}
	return &cms[0], nil
}

// getConfigMapsByLabels gets the configmaps by label
func (m Manager) getConfigMapsByLabels(requirements ...labels.Requirement) ([]corev1.ConfigMap, error) {
	// Find the scenario configmaps in the cluster
	selector := labels.NewSelector()
	for _, req := range requirements {
		selector = selector.Add(req)
	}
	cms, err := m.Client.ConfigMaps(m.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to find scenario ConfigMaps: %v", err)
	}
	return cms.Items, nil
}

// getScenarioFromConfigmap gets the Scenario from the ConfigMap
func (m Manager) getScenarioFromConfigmap(cm *corev1.ConfigMap) (*Scenario, error) {
	// Load the scenario from the base64
	decoded, err := base64.StdEncoding.DecodeString(cm.Data[DataScenarioKey])
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to decode configmap %s/%s data at key %s: %v", cm.Namespace, cm.Name, DataScenarioKey, err)
	}
	var sc Scenario
	if err := yaml.Unmarshal(decoded, &sc); err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to unmarshal Scenario in configmap %s/%s: %v", cm.Namespace, cm.Name, err)
	}
	return &sc, nil
}

func buildConfigmapName(ID string) string {
	return fmt.Sprintf("%s%s%s", PsrPrefix, "-", ID)
}
