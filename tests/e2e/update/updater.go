// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"context"
	"fmt"
	"time"

	ginkgov2 "github.com/onsi/ginkgo/v2"
	gomege "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

type CRModifier interface {
	ModifyCR(cr *vzapi.Verrazzano)
}

// GetCR gets the CR.  If it is not "Ready", wait for up to 5 minutes for it to be "Ready".
func GetCR() *vzapi.Verrazzano {
	// Wait for the CR to be Ready
	gomege.Eventually(func() error {
		cr, err := pkg.GetVerrazzano()
		if err != nil {
			return err
		}
		if cr.Status.State != vzapi.VzStateReady {
			return fmt.Errorf("CR in state %s, not Ready yet", cr.Status.State)
		}
		return nil
	}, waitTimeout, pollingInterval).Should(gomege.BeNil(), "Expected to get Verrazzano CR with Ready state")

	// Get the CR
	cr, err := pkg.GetVerrazzano()
	if err != nil {
		ginkgov2.Fail(err.Error())
	}
	if cr == nil {
		ginkgov2.Fail("CR is nil")
	}

	return cr
}

// UpdateCR updates the CR with the given CRModifier.
// First it waits for CR to be "Ready" before using the specified CRModifier modifies the CR.
// Then, it updates the modified.
// Any error during the process will cause Ginkgo Fail.
func UpdateCR(m CRModifier) {
	// Get the CR
	cr := GetCR()

	// Modify the CR
	m.ModifyCR(cr)

	// Update the CR
	var err error
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		ginkgov2.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}
	vzClient := client.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	if err != nil {
		ginkgov2.Fail(fmt.Sprintf("error updating Verrazzano instance: %v", err))
	}
}
