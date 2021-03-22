// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/onsi/ginkgo"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

const multiclusterNamespace = "verrazzano-mc"
const verrazzanoSystemNamespace = "verrazzano-system"

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = ginkgo.Describe("Multi Cluster Verify Register", func() {
	ginkgo.Context("Admin Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		ginkgo.It("admin cluster create VerrazzanoProject", func() {
			// create a project
			err := pkg.CreateOrUpdateResourceFromFile(fmt.Sprintf("testdata/multicluster/verrazzanoproject-%s.yaml", managedClusterName))
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to create VerrazzanoProject: %v", err))
			}

			gomega.Eventually(func() bool {
				return true
				// TODO check VerrazzanoProject
				/*
					vp := &v1alpha1.VerrazzanoProject{}
					listOptions := &client.ListOptions{Namespace: constants.VerrazzanoMultiClusterNamespace}
					err := pkg.GetVerrazzanoManagedClusterClientset().ClustersV1alpha1().(s.Context, allAdminProjects, listOptions)
				*/
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find VerrazzanoManagedCluster")
		})

		ginkgo.It("admin cluster has the expected VerrazzanoManagedCluster", func() {
			gomega.Eventually(func() bool {
				vmc, err := pkg.GetVerrazzanoManagedClusterClientset().ClustersV1alpha1().VerrazzanoManagedClusters(multiclusterNamespace).Get(context.TODO(), managedClusterName, metav1.GetOptions{})
				return err == nil && vmc.Status.LastAgentConnectTime.After(time.Now().Add(-30*time.Minute))
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find VerrazzanoManagedCluster")
		})

		ginkgo.It("admin cluster has the expected secrets", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(findSecret(multiclusterNamespace, fmt.Sprintf("verrazzano-cluster-%s-manifest", managedClusterName)),
						waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(findSecret(multiclusterNamespace, fmt.Sprintf("verrazzano-cluster-%s-agent", managedClusterName)),
						waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(findSecret(multiclusterNamespace, fmt.Sprintf("verrazzano-cluster-%s-registration", managedClusterName)),
						waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
			)
		})

		ginkgo.It("admin cluster has the expected system logs from admin and managed cluster", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(func() bool {
						return findLogs("vmo-local-filebeat-"+time.Now().Format("2006.01.02"),
							"fields.verrazzano.cluster.name", "local")
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a filebeat log record from admin cluster")
				},
				func() {
					gomega.Eventually(func() bool {
						return findLogs("vmo-local-journalbeat-"+time.Now().Format("2006.01.02"),
							"fields.verrazzano.cluster.name", "local")
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a journalbeat log record from admin cluster")
				},
				func() {
					gomega.Eventually(func() bool {
						return findLogs("vmo-local-filebeat-"+time.Now().Format("2006.01.02"),
							"fields.verrazzano.cluster.name", managedClusterName)
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a filebeat log record from managed cluster")
				},
				func() {
					gomega.Eventually(func() bool {
						return findLogs("vmo-local-journalbeat-"+time.Now().Format("2006.01.02"),
							"fields.verrazzano.cluster.name", managedClusterName)
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a journalbeat log record from managed cluster")
				},
			)
		})

		ginkgo.It("admin cluster has the expected metrics from managed cluster", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("up", "managed_cluster", managedClusterName)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a metrics from managed cluster")
		})
	})

	ginkgo.Context("Managed Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		ginkgo.It("managed cluster has the expected secrets", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(findSecret(verrazzanoSystemNamespace, "verrazzano-cluster-agent"),
						waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(findSecret(verrazzanoSystemNamespace, "verrazzano-cluster-registration"),
						waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
			)
		})

		ginkgo.It("managed cluster has the expected namespace and role bindings", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(findNamespace(fmt.Sprintf("ns-%s", managedClusterName)),
						waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
				// TODO check rolebinding
			)
		})
	})
})

func findSecret(namespace, name string) bool {
	s, err := pkg.GetSecret(namespace, name)
	return s != nil && err == nil
}

func findNamespace(namespace string) bool {
	ns, err := pkg.GetKubernetesClientset().CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			ginkgo.Fail(fmt.Sprintf("Failed to get namespace %s with error: %v", namespace, err))
		}
		fmt.Printf("The namespace %q is not found", namespace)
		return false
	}
	labels := ns.GetObjectMeta().GetLabels()
	if labels[constants.LabelVerrazzanoManaged] != constants.LabelVerrazzanoManagedDefault {
		fmt.Printf("The namespace %q label %q is set to wrong value of %q", namespace, constants.LabelVerrazzanoManaged, labels[constants.LabelVerrazzanoManaged])
		return false
	}
	if labels[constants.LabelIstioInjection] != constants.LabelIstioInjectionDefault {
		fmt.Printf("The namespace %q label %q is set to wrong value of %q", namespace, constants.LabelIstioInjection, labels[constants.LabelIstioInjection])
		return false
	}
	return true
}

func findLogs(index, fieldName, fieldValue string) bool {
	return pkg.LogRecordFound(index,
		time.Now().Add(-24*time.Hour),
		map[string]string{fieldName: fieldValue})
}
