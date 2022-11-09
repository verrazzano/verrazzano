package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzalpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	e2epkg "github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateCR updates the CR with the given CRModifier
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

// GetVzCr returns true if VZ CR is ready
func GetVzCr() (*vzalpha1.Verrazzano, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return nil, err
	}
	// Get the CR
	cr, err := e2epkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return cr, nil
}

func IsReady(cr *vzalpha1.Verrazzano) bool {
	return cr.Status.State == vzalpha1.VzStateReady
}
