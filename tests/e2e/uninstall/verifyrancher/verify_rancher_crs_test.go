// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verifyrancher

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"strings"
	"time"
)

const (
	waitTimeout     = 2 * time.Minute
	pollingInterval = 5 * time.Second
)

var (
	crds      *apiextv1.CustomResourceDefinitionList
	clientset dynamic.Interface
)

var t = framework.NewTestFramework("uninstall verify Rancher CRs")

// This test verifies that Rancher custom resources have been deleted from the cluster.
var _ = t.Describe("Verify Rancher custom resources", Label("f:platform-lcm.unnstall"), func() {
	t.It("Check for unexpected Rancher custom resources", func() {
		Eventually(func() (*apiextv1.CustomResourceDefinitionList, error) {
			var err error
			crds, err = pkg.ListCRDs()
			return crds, err
		}, waitTimeout, pollingInterval).ShouldNot(BeNil())

		Eventually(func() (dynamic.Interface, error) {
			kubePath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				return nil, err
			}
			clientset, err = pkg.GetDynamicClientInCluster(kubePath)
			return clientset, err
		}, waitTimeout, pollingInterval).ShouldNot(BeNil())

		unexpectedCRs := false

		for _, crd := range crds.Items {
			if strings.HasSuffix(crd.Name, ".cattle.io") {
				for _, version := range crd.Spec.Versions {
					rancherCRs, err := clientset.Resource(schema.GroupVersionResource{
						Group:    crd.Spec.Group,
						Version:  version.Name,
						Resource: crd.Spec.Names.Plural,
					}).List(context.TODO(), metav1.ListOptions{})
					if err != nil {
						Expect(err).To(Not(HaveOccurred()), fmt.Sprintf("Failed listing custom resources for %s", crd.Spec.Names.Kind))
					}
					if len(rancherCRs.Items) == 0 {
						continue
					}
					for _, rancherCR := range rancherCRs.Items {
						pkg.Log(pkg.Error, fmt.Sprintf("Unexpected custom resource %s/%s was found for %s.%s/%s", rancherCR.GetNamespace(), rancherCR.GetName(), crd.Spec.Names.Singular, crd.Spec.Group, version.Name))
						unexpectedCRs = true
					}
				}
			}
		}

		if unexpectedCRs {
			Fail("Failed to verify Rancher custom resources")
		}
	})
})
