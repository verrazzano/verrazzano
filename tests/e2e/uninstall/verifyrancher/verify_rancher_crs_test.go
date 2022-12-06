// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verifyrancher

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

var t = framework.NewTestFramework("uninstall verify Rancher CRs")

// This test verifies that Rancher cluster scoped custom resources have been deleted from the cluster.
var _ = t.Describe("Verify Rancher cluster scoped resources deleted after uninstall.", Label("f:platform-lcm.unnstall"), func() {
	crds, err := pkg.ListCRDs()
	if err != nil {
		Fail(err.Error())
	}

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Errorf("Failed getting kubeconfig: %s", err).Error())
	}

	// Get the Dynamic clientset
	clientset, err := pkg.GetDynamicClientInCluster(kubeconfigPath)
	if err != nil {
		Fail(fmt.Errorf("Failed getting Kubernetes clientset: %s", err).Error())
	}

	unexpectedCRs := false

	for _, crd := range crds.Items {
		if strings.HasSuffix(crd.Name, ".cattle.io") && crd.Spec.Scope == v1.ClusterScoped {
			for _, version := range crd.Spec.Versions {
				rancherCRs, err := clientset.Resource(schema.GroupVersionResource{
					Group:    crd.Spec.Group,
					Version:  version.Name,
					Resource: crd.Spec.Names.Plural,
				}).List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					Fail(fmt.Errorf("Failed listing custom resources: %s", err).Error())
				}
				if len(rancherCRs.Items) == 0 {
					continue
				}
				for _, rancherCR := range rancherCRs.Items {
					pkg.Log(pkg.Error, fmt.Sprintf("Unexpected custom resource %s was found for %s/%s", rancherCR.GetName(), crd.Spec.Group, version.Name))
					unexpectedCRs = true
				}
			}
		}
	}

	if unexpectedCRs {
		Fail("Failed to verify Rancher custom resources")
	}
})
