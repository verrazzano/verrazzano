// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CRModifier interface {
	ModifyCR(cr *vzapi.Verrazzano)
}

type CRModifierV1beta1 interface {
	ModifyCRV1beta1(cr *v1beta1.Verrazzano)
}

// GetCR gets the CR.  If it is not "Ready", GetCR returns an error.
func GetCR() (*vzapi.Verrazzano, error) {
	var vz *vzapi.Verrazzano
	// Get the CR if it is Ready
	cr, err := pkg.GetVerrazzano()
	if err != nil {
		return nil, err
	}
	if cr.Status.State != vzapi.VzStateReady {
		return nil, fmt.Errorf("CR in state %s, not Ready yet", cr.Status.State)
	}
	vz = cr
	return vz, nil
}

// GetCRV1beta1 gets the CR.  If it is not "Ready", GetCRV1beta1 returns an error.
func GetCRV1beta1() (*v1beta1.Verrazzano, error) {
	// Wait for the CR to be Ready
	var vz *v1beta1.Verrazzano
	cr, err := pkg.GetVerrazzanoV1beta1()
	if err != nil {
		return nil, err
	}
	if cr.Status.State != v1beta1.VzStateReady {
		return nil, fmt.Errorf("v1beta1 CR in state %s, not Ready yet", cr.Status.State)
	}
	vz = cr
	return vz, nil
}

// UpdateCR updates the CR with the given CRModifier if the CR is "Ready".
// Then, it updates the modified.
// Any error during the process will return an error.
func UpdateCR(m CRModifier) error {
	// Get the CR
	cr, err := GetCR()
	if err != nil {
		return err
	}

	// Modify the CR
	m.ModifyCR(cr)

	// Update the CR
	path, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
	config, err := k8sutil.GetKubeConfigGivenPath(path)
	if err != nil {
		return err
	}
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		return err
	}
	vzClient := client.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	return err
}

// UpdateCRV1beta1 updates the CR with the given CRModifierV1beta1 if the CR is "Ready".
// Then, it updates the modified.
// Any error during the process will return an error.
func UpdateCRV1beta1(m CRModifierV1beta1) error {
	// GetCRV1beta1 gets the CR using v1beta1 client.
	cr, err := GetCRV1beta1()
	if err != nil {
		return err
	}

	// Modify the CR
	m.ModifyCRV1beta1(cr)

	// Update the CR
	path, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
	config, err := k8sutil.GetKubeConfigGivenPath(path)
	if err != nil {
		return err
	}
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		return err
	}
	vzClient := client.VerrazzanoV1beta1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	return err
}
