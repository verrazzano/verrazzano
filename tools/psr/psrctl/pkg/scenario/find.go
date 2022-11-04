// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/yaml"
)

const (
	PsrPrefix = "psr"
)

// FindRunningScenarios returns the list of Scenarios that are running in the cluster.
// Scenario manifests
func (m Manager) FindRunningScenarios() ([]Scenario, error) {
	scenarios := []Scenario{}

	client, err := k8sutil.GetCoreV1Client(m.Log)
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to get CoreV1 client: %v", err)
	}

	// Find the scenario configmaps in the cluster
	req, _ := labels.NewRequirement(LabelScenario, selection.Exists, nil)
	selector := labels.NewSelector()
	selector = selector.Add(*req)

	cms, err := client.ConfigMaps(m.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to find scenario ConfigMaps: %v", err)
	}

	for _, cm := range cms.Items {
		// Load the scenario from the base64
		decoded, err := base64.StdEncoding.DecodeString(cm.Data[DataScenarioKey])
		if err != nil {
			return nil, m.Log.ErrorfNewErr("Failed to decode configmap %s/%s data at key %s: %v", cm.Namespace, cm.Name, DataScenarioKey, err)
		}
		var sc Scenario
		if err := yaml.Unmarshal(decoded, &sc); err != nil {
			return nil, m.Log.ErrorfNewErr("Failed to unmarshal Scenario in configmap %s/%s: %v", cm.Namespace, cm.Name, err)
		}
		scenarios = append(scenarios, sc)
	}

	return scenarios, nil
}

func (m Manager) saveScenario(scenario Scenario) (*corev1.ConfigMap, error) {
	client, err := k8sutil.GetCoreV1Client(m.Log)
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to get CoreV1 client: %v", err)
	}

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
				LabelScenarioId: scenario.ScenarioManifest.ID},
		},
		Data: map[string]string{"scenario": encoded},
	}

	cmNew, err := client.ConfigMaps(m.Namespace).Create(context.TODO(), &cmIn, metav1.CreateOptions{})
	if err != nil {
		return nil, m.Log.ErrorfNewErr("Failed to create scenario ConfigMap %s/%s: %v", cmIn.Namespace, cmIn.Name, err)
	}
	return cmNew, nil
}

func buildConfigmapName(name string) string {
	return fmt.Sprintf("%s%s%s", PsrPrefix, "-", name)
}
