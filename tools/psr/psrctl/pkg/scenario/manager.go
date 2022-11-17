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
	client, err := k8sutil.GetCoreV1Client(vzlog.DefaultLogger())
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
