// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// unitTestBomFIle is used for unit test
const unitTestBomFile = "../../../../verrazzano-bom.json"

// TestRestartAllWorkloadTypes tests the RestartComponents method for the following use case
// GIVEN a request to RestartComponents a component
// WHEN where the fake client has deployments, statefulset, and daemonsets that need to be restarted
// THEN the upgrade completes normally and the correct spi.Component upgrade methods have not been invoked for the disabled component
func TestRestartAllWorkloadTypes(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	goClient, err := initFakeClient()
	asserts.NoError(err)
	k8sutil.SetFakeClient(goClient)

	namespaces := []string{constants.VerrazzanoSystemNamespace}
	err = RestartComponents(vzlog.DefaultLogger(), namespaces, 1)

	// Validate the results
	asserts.NoError(err)
	dep, err := goClient.AppsV1().Deployments("verrazzano-system").Get(context.TODO(), "testDeployment", metav1.GetOptions{})
	asserts.NoError(err)
	fmt.Printf(dep.Name)
}

// initFakeClient inits a fake go-client and loads it with fake resources
func initFakeClient() (kubernetes.Interface, error) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "verrazzano-system",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
	}
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "verrazzano-system",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
	}
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "verrazzano-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testPod",
			Namespace: "verrazzano-system",
			Labels:    map[string]string{"app": "foo"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "proxy",
				Image: "proxyv2.old",
			}},
		},
	}
	clientSet := fake.NewSimpleClientset(dep, pod, sts, daemonSet)
	return clientSet, nil
}
