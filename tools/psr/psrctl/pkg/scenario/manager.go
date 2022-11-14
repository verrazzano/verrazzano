// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Manager contains the information needed to manage a Scenario
type Manager struct {
	Log                 vzlog.VerrazzanoLogger
	Client              corev1.CoreV1Interface
	Manifest            embedded.PsrManifests
	ExternalScenarioDir string
	Namespace           string
	WorkerImage         string
	DryRun              bool
	Verbose             bool
}

// NewManager returns a scenario Manager
func NewManager(namespace string, externalScenarioDir string, imageName string) (Manager, error) {
	client, err := k8sutil.GetCoreV1Client(vzlog.DefaultLogger())
	if err != nil {
		return Manager{}, fmt.Errorf("Failed to get CoreV1 client: %v", err)
	}
	if len(imageName) == 0 {
		imageName = constants.GetDefaultWorkerImage()
	}
	m := Manager{
		Namespace:           namespace,
		Log:                 vzlog.DefaultLogger(),
		Manifest:            *embedded.Manifests,
		ExternalScenarioDir: externalScenarioDir,
		WorkerImage:         imageName,
		Client:              client,
		Verbose:             true,
	}
	return m, nil
}
