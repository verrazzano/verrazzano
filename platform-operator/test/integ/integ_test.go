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
const managedGeneratedName_1 = "verrazzano-cluster-cluster1"
const installNamespace = "verrazzano-install"
const vzMcNamespace = "verrazzano-mc"
const prometheusSecret = "prometheus-cluster1"
const vmiESIngest = "vmi-system-es-ingest"
const hostdata = "testhost"

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
		err := K8sClient.EnsureNamespace(vzMcNamespace)
		gomega.Expect(err).To(gomega.BeNil())
	})
	ginkgo.It("Create verrazzano-system namespace ", func() {
		err := K8sClient.EnsureNamespace(constants.VerrazzanoSystemNamespace)
		gomega.Expect(err).To(gomega.BeNil())
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
	ginkgo.It("Create Verrazzano secret needed to create ES secret ", func() {
		_, stderr := util.Kubectl(
			fmt.Sprintf("create secret generic %s -n %s --from-literal=password=mypw --from-literal=username=myuser", constants.Verrazzano, constants.VerrazzanoSystemNamespace))
		gomega.Expect(stderr).To(gomega.Equal(""))
	})
	ginkgo.It("Create system tls secret needed to create ES secret ", func() {
		_, stderr := util.Kubectl(
			// a generic secret is ok for testing
			fmt.Sprintf("create secret generic %s -n %s --from-literal=cr.crt=fakeCA", constants.SystemTLS, constants.VerrazzanoSystemNamespace))
		gomega.Expect(stderr).To(gomega.Equal(""))
	})
	ginkgo.It("Create Elasticsearch ingress needed to create ES secret", func() {
		ingressExists := func() bool {
			return K8sClient.DoesIngressExist(vmiESIngest, constants.VerrazzanoSystemNamespace)
		}
		createFakeElasticsearchIngress()
		gomega.Eventually(ingressExists(), "10s", "5s").Should(gomega.BeTrue(),
			"The Elasticsearch ingress should exist")
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
	ginkgo.It("registration secret exists ", func() {
		secretExists := func() bool {
			return K8sClient.DoesSecretExist(vzclusters.GetRegistrationSecretName(managedClusterName), vzMcNamespace)
		}
		gomega.Eventually(secretExists, "30s", "5s").Should(gomega.BeTrue(),
			fmt.Sprintf("The registration Secret %s should exist in %s", vzclusters.GetRegistrationSecretName(managedClusterName), vzMcNamespace))
	})
	ginkgo.It("Elasticsearch secret exists ", func() {
		secretExists := func() bool {
			return K8sClient.DoesSecretExist(vzclusters.GetElasticsearchSecretName(managedClusterName), vzMcNamespace)
		}
		gomega.Eventually(secretExists, "30s", "5s").Should(gomega.BeTrue(),
			fmt.Sprintf("The Elasticsearch Secret %s should exist in %s", vzclusters.GetElasticsearchSecretName(managedClusterName), vzMcNamespace))
	})
	ginkgo.It("manifest secret exists ", func() {
		secretExists := func() bool {
			return K8sClient.DoesSecretExist(vzclusters.GetManifestSecretName(managedClusterName), vzMcNamespace)
		}
		gomega.Eventually(secretExists, "30s", "5s").Should(gomega.BeTrue(),
			fmt.Sprintf("The kubeconfig Secret %s should exist in %s", vzclusters.GetManifestSecretName(managedClusterName), vzMcNamespace))
	})
	ginkgo.It("Checking the VMC related secrets ", func() {
		verifyAdminSecret()
		verifyRegistrationSecret()
		verifyManifestSecret()
	})
})

// Verify the admin secret
func verifyAdminSecret() {
	// Get the admin secret
	secret, err := K8sClient.GetSecret(vzclusters.GetAgentSecretName(managedClusterName), vzMcNamespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Unable to get registration secret %s: %v", vzclusters.GetAgentSecretName(managedClusterName), err))
	}

	// Get the kubeconfig from the secret
	kubconfigBytes := secret.Data["admin-kubeconfig"]
	if len(kubconfigBytes) == 0 {
		ginkgo.Fail(fmt.Sprintf("Cluster secret %s does not contain kubeconfig", err))
	}
}

// Verify the registration secret
func verifyRegistrationSecret() {
	// Get the registration secret
	secret, err := K8sClient.GetSecret(vzclusters.GetRegistrationSecretName(managedClusterName), vzMcNamespace)
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
	secret, err := K8sClient.GetSecret(vzclusters.GetManifestSecretName(managedClusterName), vzMcNamespace)
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
