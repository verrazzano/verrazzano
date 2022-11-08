package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	e2epkg "github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateCR updates the CR with the given CRModifier
// - if the CRModifier implements rest.WarningHandler it will be added to the client config
//
// First it waits for CR to be "Ready" before using the specified CRModifier modifies the CR.
// Then, it updates the modified.
// Any error during the process will cause Ginkgo Fail.
func UpdateCR(ctrlRuntimeClient client.Client, m update.CRModifier) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
	// Get the CR
	cr, err := e2epkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return err
	}

	// Modify the CR
	m.ModifyCR(cr)

	// Update the CR
	err = ctrlRuntimeClient.Update(context.TODO(), cr, &client.UpdateOptions{})
	return err
}
