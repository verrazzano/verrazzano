// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"context"
	"fmt"
	"os"
	"time"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
				return findVerrazzanoProject(fmt.Sprintf("project-%s", managedClusterName))
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find VerrazzanoProject")
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
		pkg.Log(pkg.Info, fmt.Sprintf("The namespace %q is not found", namespace))
		return false
	}
	labels := ns.GetObjectMeta().GetLabels()
	if labels[constants.LabelVerrazzanoManaged] != constants.LabelVerrazzanoManagedDefault {
		pkg.Log(pkg.Info, fmt.Sprintf("The namespace %q label %q is set to wrong value of %q", namespace, constants.LabelVerrazzanoManaged, labels[constants.LabelVerrazzanoManaged]))
		return false
	}
	if labels[constants.LabelIstioInjection] != constants.LabelIstioInjectionDefault {
		pkg.Log(pkg.Info, fmt.Sprintf("The namespace %q label %q is set to wrong value of %q", namespace, constants.LabelIstioInjection, labels[constants.LabelIstioInjection]))
		return false
	}
	return true
}

func findLogs(index, fieldName, fieldValue string) bool {
	return pkg.LogRecordFound(index,
		time.Now().Add(-24*time.Hour),
		map[string]string{fieldName: fieldValue})
}

func findVerrazzanoProject(projectName string) bool {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("TEST_KUBECONFIG"))
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to build config from %s with error: %v", os.Getenv("TEST_KUBECONFIG"), err))
	}

	scheme := runtime.NewScheme()
	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)

	clustersClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get clusters client with error: %v", err))
	}

	//var project clustersv1alpha1.VerrazzanoProject
	//clustersClient.Get(context.TODO(), types.NamespacedName{Namespace: multiclusterNamespace, Name: projectName}, &project)

	projectList := clustersv1alpha1.VerrazzanoProjectList{}
	err = clustersClient.List(context.TODO(), &projectList, &client.ListOptions{Namespace: constants.VerrazzanoMultiClusterNamespace})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list VerrazzanoProject with error: %v", err))
	}
	for _, item := range projectList.Items {
		if item.Name == projectName && item.Namespace == multiclusterNamespace {
			return true
		}
	}
	return false
}
