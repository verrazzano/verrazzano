// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package availability

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"time"
)

const (
	pollingInterval = 10 * time.Second
	timeout         = 10 * time.Minute
	deployName      = "rancher"
	deployNamespace = "cattle-system"
)

var (
	t         = framework.NewTestFramework("availability")
	clientset = k8sutil.GetKubernetesClientsetOrDie()
)

var _ = t.BeforeSuite(func() {})

var _ = t.Describe("Status", func() {
	// If a component is taken offline, the number of unavailable components should increase by 1
	// When that component is put back online, the number of unavailable components should be 0
	t.It("dynamically scales availability", func() {
		t.Logs.Info("Should be 0 unavailable components when the test begins")
		hasUnavailableComponents(0)
		t.Logs.Infof("Scale down target deployment %s/%s", deployNamespace, deployName)
		scaleDeployment(deployName, deployNamespace, 0)
		t.Logs.Info("After the health check updates the status, 1 component should be unavailable")
		hasUnavailableComponents(1)
		t.Logs.Info("Unavailable components should reconcile and be available")
		hasUnavailableComponents(0)
	})
})

func scaleDeployment(name, namespace string, replicas int32) {
	Eventually(func() bool {
		deploy, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			t.Logs.Errorf("Error fetching deployment: %v", err)
			return false
		}
		deploy.Spec.Replicas = &replicas
		_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deploy, metav1.UpdateOptions{})
		return err == nil
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(BeTrue())
}

func hasUnavailableComponents(numUnavailable int) {
	Eventually(func() bool {
		verrazzano, err := pkg.GetVerrazzano()
		if err != nil {
			t.Logs.Errorf("Error fetching Verrazzano: %v", err)
			return false
		}
		isMet, err := availabilityCriteriaMet(verrazzano.Status.Available, numUnavailable)
		if err != nil {
			t.Logs.Errorf("Error finding availability criteria: %v", err)
			return false
		}
		return isMet
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(BeTrue())
}

func availabilityCriteriaMet(availability *string, numUnavailable int) (bool, error) {
	if availability == nil {
		return false, nil
	}
	// availability is of the form "x/y" where x is components available, y is components enabled
	lValue, rValue, err := parseAvailability(*availability)
	if err != nil {
		return false, err
	}
	return lValue+numUnavailable == rValue, nil
}

func parseAvailability(availability string) (int, int, error) {
	const invalid = -1
	split := strings.Split(availability, "/")
	if len(split) != 2 {
		return invalid, invalid, fmt.Errorf("availability should have to numeric terms, got %s", availability)
	}
	lValue, err := strconv.Atoi(split[0])
	if err != nil {
		return invalid, invalid, err
	}
	rValue, err := strconv.Atoi(split[1])
	if err != nil {
		return invalid, invalid, err
	}
	return lValue, rValue, nil
}
