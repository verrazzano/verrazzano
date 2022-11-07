// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
)

// StopScenarioByID stops a running scenario specified by the scenario ID
func (m Manager) StopScenarioByID(ID string) (string, error) {
	cm, err := m.getConfigMapByID(ID)
	if err != nil {
		return "", err
	}
	sc, err := m.getScenarioFromConfigmap(cm)
	if err != nil {
		return "", err
	}
	// Delete Helm releases
	for _, h := range sc.HelmReleases {
		_, stderr, err := helmcli.Uninstall(m.Log, h.Name, h.Namespace, m.DryRun)
		if err != nil {
			return string(stderr), err
		}
	}
	// Delete config map
	err = m.deleteConfigMap(cm)
	if err != nil {
		return "", err
	}
	return "", nil
}
