// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// ScenarioMananger contains the information needed to manage a Scenario
type ScenarioMananger struct {
	Namespace     string
	DryRun        bool
	Verbose       bool
	Log           vzlog.VerrazzanoLogger
	Client        corev1.CoreV1Interface
	HelmOverrides []helm.HelmOverrides
}

// NewManager returns a scenario ScenarioMananger
func NewManager(namespace string, helmOverrides ...helm.HelmOverrides) (ScenarioMananger, error) {
	log := vzlog.DefaultLogger()
	client, err := k8sutil.GetCoreV1Func(log)
	if err != nil {
		return ScenarioMananger{}, fmt.Errorf("Failed to get CoreV1 client: %v", err)
	}
	m := ScenarioMananger{
		Namespace:     namespace,
		Log:           vzlog.DefaultLogger(),
		HelmOverrides: helmOverrides,
		Client:        client,
		Verbose:       true,
	}
	return m, nil
}

// FindRunningScenarios returns the list of Scenarios that are running in the cluster.
func (m ScenarioMananger) FindRunningScenarios() ([]Scenario, error) {
	scenarios := []Scenario{}

	cms, err := m.getAllConfigMaps()
	if err != nil {
		return nil, err
	}

	for i := range cms {
		sc, err := m.getScenarioFromConfigmap(&cms[i])
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, *sc)
	}

	return scenarios, nil
}

// FindRunningScenarioByID returns the Scenario with the specified Scenario ID
func (m ScenarioMananger) FindRunningScenarioByID(ID string) (*Scenario, error) {
	cm, err := m.getConfigMapByID(ID)
	if err != nil {
		return nil, err
	}
	sc, err := m.getScenarioFromConfigmap(cm)
	if err != nil {
		return nil, err
	}
	return sc, nil
}
