package verrazzano

import (
	"context"
	"fmt"
	vzalpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//
//// UpdateCR updates the CR with the given CRModifier
//func UpdateCR(ctrlRuntimeClient client.Client,cr *vzalpha1.Verrazzano, m update.CRModifier) error {
//
//	// Modify the CR
//	m.ModifyCR(cr)
//
//	// Update the CR
//	err := ctrlRuntimeClient.Update(context.TODO(), cr, &client.UpdateOptions{})
//	return err
//}

// UpdateVerrazzanoResource updates the CR with the given CRModifier
func UpdateVerrazzanoResource(client vpoClient.Interface, cr *vzalpha1.Verrazzano) error {
	// Update the CR
	_, err := client.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace).Update(context.TODO(), cr, metav1.UpdateOptions{})
	return err
}

func IsReady(cr *vzalpha1.Verrazzano) bool {
	return cr.Status.State == vzalpha1.VzStateReady
}

// GetVerrazzanoResource returns the installed Verrazzano CR in the given cluster
// (there should only be 1 per cluster)
func GetVerrazzanoResource(client vpoClient.Interface) (*vzalpha1.Verrazzano, error) {
	vzClient := client.VerrazzanoV1alpha1().Verrazzanos("")
	vzList, err := vzClient.List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("error listing out Verrazzano instances: %v", err)
	}
	numVzs := len(vzList.Items)
	if numVzs == 0 {
		return nil, fmt.Errorf("did not find installed Verrazzano instance")
	}
	vz := vzList.Items[0]
	return &vz, nil
}
