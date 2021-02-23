// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"context"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/test/integ/k8s"
	"github.com/verrazzano/verrazzano/platform-operator/test/integ/util"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

const clusterAdmin = "cluster-admin"
const platformOperator = "verrazzano-platform-operator"
const managedGeneratedName_1 = "verrazzano-cluster-cluster1"
const installNamespace = "verrazzano-install"
const vzMcNamespace = "verrazzano-mc"
const prometheusSecret = "prometheus-cluster1"

var K8sClient k8s.Client

var _ = ginkgo.BeforeSuite(func() {
	var err error
	K8sClient, err = k8s.NewClient(util.GetKubeconfig())
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}
})

var _ = ginkgo.AfterSuite(func() {
})

var _ = ginkgo.Describe("verrazzano-install namespace resources ", func() {
	ginkgo.It(fmt.Sprintf("Namespace %s exists", installNamespace), func() {
		gomega.Expect(K8sClient.DoesNamespaceExist(installNamespace)).To(gomega.BeTrue(),
			"The install-namespace should exist")
	})
	ginkgo.It(fmt.Sprintf("ServiceAccount %s exists", platformOperator), func() {
		gomega.Expect(K8sClient.DoesServiceAccountExist(platformOperator, installNamespace)).To(gomega.BeTrue(),
			"The verrazzano-platform-operator service should exist")
	})
	ginkgo.It(fmt.Sprintf("Deployment %s exists", platformOperator), func() {
		gomega.Expect(K8sClient.DoesDeploymentExist(platformOperator, installNamespace)).To(gomega.BeTrue(),
			"The verrazzano-platform-operator should exist")
	})
	ginkgo.It(fmt.Sprintf("Pod prefixed by %s exists", platformOperator), func() {
		gomega.Expect(K8sClient.DoesPodExist(platformOperator, installNamespace)).To(gomega.BeTrue(),
			"The verrazzano-platform-operator pod should exist")
	})
	ginkgo.It("Platform operator pod is eventually running", func() {
		isPodRunningYet := func() bool {
			return K8sClient.IsPodRunning(platformOperator, installNamespace)
		}
		gomega.Eventually(isPodRunningYet, "2m", "5s").Should(gomega.BeTrue(),
			"The verrazzano-platform-operator pod should be in the Running state")
	})
})

var _ = ginkgo.Describe("Verrazzano cluster roles and bindings for platform operator", func() {
	ginkgo.It(fmt.Sprintf("Cluster admin role %s exists", clusterAdmin), func() {
		gomega.Expect(K8sClient.DoesClusterRoleExist(clusterAdmin)).To(gomega.BeTrue(),
			"The cluster-admin role should exist")
	})
	ginkgo.It(fmt.Sprintf("Cluster role binding for platform operator %s exists", platformOperator), func() {
		gomega.Expect(K8sClient.DoesClusterRoleBindingExist(platformOperator)).To(gomega.BeTrue(),
			"The cluster role binding for verrazzano-platform-operator should exist")
	})

})

var _ = ginkgo.Describe("Custom Resource Definition for verrazzano install", func() {
	ginkgo.It("verrazzanos.install.verrazzano.io exists", func() {
		gomega.Expect(K8sClient.DoesCRDExist("verrazzanos.install.verrazzano.io")).To(gomega.BeTrue(),
			"The verrazzanos.install.verrazzano.io CRD should exist")
	})
	ginkgo.It("verrazzanomanagedclusters.clusters.verrazzano.io exists", func() {
		gomega.Expect(K8sClient.DoesCRDExist("verrazzanomanagedclusters.clusters.verrazzano.io")).To(gomega.BeTrue(),
			"The verrazzanomanagedclusters.clusters.verrazzano.io CRD should exist")
	})
})

var _ = ginkgo.Describe("Testing VerrazzanoManagedCluster CRDs", func() {
	ginkgo.It("Platform operator pod is eventually running", func() {
		isPodRunningYet := func() bool {
			return K8sClient.IsPodRunning(platformOperator, installNamespace)
		}
		gomega.Eventually(isPodRunningYet, "2m", "5s").Should(gomega.BeTrue(),
			"The verrazzano-platform-operator pod should be in the Running state")
	})
	ginkgo.It("Create multi-cluster namespace ", func() {
		_, stderr := util.Kubectl(fmt.Sprintf("create namespace %s", vzMcNamespace))
		gomega.Expect(stderr).To(gomega.Equal(""))
	})
	ginkgo.It("Missing secret name validation ", func() {
		_, stderr := util.Kubectl("apply -f testdata/vmc_missing_secret_name.yaml")
		gomega.Expect(stderr).To(gomega.ContainSubstring("missing required field \"prometheusSecret\""))
	})
	ginkgo.It("Missing secret validation ", func() {
		_, stderr := util.Kubectl("apply -f testdata/vmc_sample.yaml")
		gomega.Expect(stderr).To(gomega.ContainSubstring(
			fmt.Sprintf(fmt.Sprintf("The Prometheus secret %s does not exist in namespace %s", prometheusSecret, vzMcNamespace))))
	})
	ginkgo.It("Create Prometheus secret ", func() {
		_, stderr := util.Kubectl(
			fmt.Sprintf("create secret generic %s -n %s --from-literal=password=mypw --from-literal=username=myuser", prometheusSecret, vzMcNamespace))
		gomega.Expect(stderr).To(gomega.Equal(""))
	})
	ginkgo.It("VerrazzanoManagedCluster can be created ", func() {
		_, stderr := util.Kubectl("apply -f testdata/vmc_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
	})
	ginkgo.It("ServiceAccount exists ", func() {
		serviceAccountExists := func() bool {
			return K8sClient.DoesServiceAccountExist(managedGeneratedName_1, vzMcNamespace)
		}
		gomega.Eventually(serviceAccountExists, "30s", "5s").Should(gomega.BeTrue(),
			"The ServiceAccount should exist")
	})
	ginkgo.It("ClusterRoleBinding exists ", func() {
		bindingExists := func() bool {
			return K8sClient.DoesClusterRoleBindingExist(managedGeneratedName_1)
		}
		gomega.Eventually(bindingExists, "30s", "5s").Should(gomega.BeTrue(),
			"The ClusterRoleBinding should exist")
	})
	ginkgo.It("kubeconfig Secret exists ", func() {
		secretExists := func() bool {
			return K8sClient.DoesSecretExist(managedGeneratedName_1, vzMcNamespace)
		}
		gomega.Eventually(secretExists, "30s", "5s").Should(gomega.BeTrue(),
			fmt.Sprintf("The kubeconfig Secret %s should exist in %s", managedGeneratedName_1, vzMcNamespace))
	})
	ginkgo.It("Checking access to admin cluster using kubeconfig in the secret ", func() {
		verifyKubeconfig()
	})
})

// Verify that the kubeconfig can be used to restrict access to the admin cluster
func verifyKubeconfig() {
	secret, err := K8sClient.GetSecret(managedGeneratedName_1, vzMcNamespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Unable to get cluster secret %s that contains kubeconfig: %v", managedGeneratedName_1, err))
	}

	// Get the kubeconfig from the secret and decode the base64
	kubconfigBytes := secret.Data["admin-kubeconfig"]
	if len(kubconfigBytes) == 0 {
		ginkgo.Fail(fmt.Sprintf("Cluster secret %s does not contain kubeconfig", err))
	}
	fmt.Println(string(kubconfigBytes))

	// Write kubeconfig to temp file
	f, err := ioutil.TempFile("", "kubeconfig")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create tempfile. Error: %v", err))
	}
	defer os.Remove(f.Name())
	_, err = f.Write(kubconfigBytes)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to write to tempfile.  Error: %v", err))
	}
	// Test access using the new client
	client, err := k8s.NewClient(string(f.Name()))
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}

	// Verify that we get not permitted
	_, err = client.Clientset.CoreV1().Namespaces().Get(context.TODO(), vzMcNamespace, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Should not be able to access  %s namespace %v", vzMcNamespace, err))
	}
}
