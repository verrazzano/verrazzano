// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	vzclusters "github.com/verrazzano/verrazzano/platform-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/platform-operator/test/integ/k8s"
	"github.com/verrazzano/verrazzano/platform-operator/test/integ/util"
	k8net "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const managedClusterName = "cluster1"
const clusterAdmin = "cluster-admin"
const platformOperator = "verrazzano-platform-operator"
const managedGeneratedName1 = "verrazzano-cluster-cluster1"
const installNamespace = "verrazzano-install"
const prometheusSecret = "prometheus-cluster1"
const vmiESIngest = "vmi-system-es-ingest"
const hostdata = "testhost"
const adminClusterConfigMap = "verrazzano-admin-cluster"

var K8sClient k8s.Client

var _ = ginkgo.BeforeSuite(func() {
	var err error
	K8sClient, err = k8s.NewClient(util.GetKubeconfig())
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}

	// Platform operator pod is eventually running
	isPodRunningYet := func() bool {
		return K8sClient.IsPodRunning(platformOperator, installNamespace)
	}
	gomega.Eventually(isPodRunningYet, "2m", "5s").Should(gomega.BeTrue(),
		"The verrazzano-platform-operator pod should be in the Running state")

	// Create multi-cluster namespace
	if !K8sClient.DoesNamespaceExist(constants.VerrazzanoMultiClusterNamespace) {
		err = K8sClient.EnsureNamespace(constants.VerrazzanoMultiClusterNamespace)
		gomega.Expect(err).To(gomega.BeNil())
	}

	// Create verrazzano-system namespace
	if !K8sClient.DoesNamespaceExist(constants.VerrazzanoSystemNamespace) {
		err = K8sClient.EnsureNamespace(constants.VerrazzanoSystemNamespace)
		gomega.Expect(err).To(gomega.BeNil())
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

// Verify the agent secret
func verifyAgentSecret() {
	// Get the agent secret
	secret, err := K8sClient.GetSecret(vzclusters.GetAgentSecretName(managedClusterName), constants.VerrazzanoMultiClusterNamespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Unable to get registration secret %s: %v", vzclusters.GetAgentSecretName(managedClusterName), err))
	}

	// Get the kubeconfig from the secret
	kubconfigBytes := secret.Data["admin-kubeconfig"]
	if len(kubconfigBytes) == 0 {
		ginkgo.Fail(fmt.Sprintf("Cluster secret %s does not contain kubeconfig", err))
	}

	// check the cluster name
	clusterName := secret.Data["managed-cluster-name"]
	if string(clusterName) != managedClusterName {
		ginkgo.Fail(fmt.Sprintf("The managed cluster name %s in the kubeconfig is incorrect", clusterName))
	}
}

// Verify the registration secret
func verifyRegistrationSecret() {
	// Get the registration secret
	secret, err := K8sClient.GetSecret(vzclusters.GetRegistrationSecretName(managedClusterName), constants.VerrazzanoMultiClusterNamespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Unable to get registration secret %s: %v", vzclusters.GetRegistrationSecretName(managedClusterName), err))
	}

	// check the cluster name
	clusterName := secret.Data["managed-cluster-name"]
	if string(clusterName) != managedClusterName {
		ginkgo.Fail(fmt.Sprintf("The managed cluster name %s in the kubeconfig is incorrect", clusterName))
	}
}

// Verify the manifest secrets
func verifyManifestSecret() {
	secret, err := K8sClient.GetSecret(vzclusters.GetManifestSecretName(managedClusterName), constants.VerrazzanoMultiClusterNamespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Unable to get cluster secret %s that contains kubeconfig: %v", vzclusters.GetManifestSecretName(managedClusterName), err))
	}

	// Get the yaml from the secret
	kubconfigBytes := secret.Data["yaml"]
	if len(kubconfigBytes) == 0 {
		ginkgo.Fail(fmt.Sprintf("Manifest secret %s does not contain yaml", err))
	}
}

// Create a fake ES ingress in Verrazzano system so that we can build the ES secret
func createFakeElasticsearchIngress() {
	// Create the ingress
	ing := k8net.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiESIngest,
		},
		Spec: k8net.IngressSpec{
			Rules: []k8net.IngressRule{{
				Host: hostdata,
			}},
		},
	}
	_, err := K8sClient.Clientset.NetworkingV1beta1().Ingresses(constants.VerrazzanoSystemNamespace).Create(context.TODO(), &ing, metav1.CreateOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Unable to create fake Elasticsearch ingress: %v", err))
	}
}
