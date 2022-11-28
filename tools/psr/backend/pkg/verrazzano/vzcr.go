// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	er "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzalpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"github.com/verrazzano/verrazzano/tools/psr/backend/pkg/k8sclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// UpdateVZCR updates the Verrazzano CR and retries if there is a conflict error
func UpdateVZCR(c k8sclient.PsrClient, log vzlog.VerrazzanoLogger, cr *vzalpha1.Verrazzano, m update.CRModifier) error {
	for {
		// Modify the CR
		m.ModifyCR(cr)

		err := UpdateVerrazzano(c.VzInstall, cr)
		if err == nil {
			break
		}
		if !er.IsUpdateConflict(err) {
			return fmt.Errorf("Failed to update Verrazzano cr: %v", err)
		}
		// Conflict error, get latest vz cr
		time.Sleep(1 * time.Second)
		log.Info("conflict error updating Verrazzano cr, retrying")

		cr, err = GetVerrazzano(c.VzInstall)
		if err != nil {
			return err
		}
	}
	log.Info("Updated Verrazzano CR")
	return nil
}

// UpdateVerrazzano updates the CR with the given CRModifier
func UpdateVerrazzano(client vpoClient.Interface, cr *vzalpha1.Verrazzano) error {
	// Update the CR
	_, err := client.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace).Update(context.TODO(), cr, metav1.UpdateOptions{})
	return err
}

// IsReady returns true if the Verrazzano CR is ready
func IsReady(cr *vzalpha1.Verrazzano) bool {
	return cr.Status.State == vzalpha1.VzStateReady
}

// GetVerrazzano returns the installed Verrazzano CR in the given cluster
// (there should only be 1 per cluster)
func GetVerrazzano(client vpoClient.Interface) (*vzalpha1.Verrazzano, error) {
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
