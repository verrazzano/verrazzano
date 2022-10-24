// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package inplaceupgrade

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/common/auth"
	ocice "github.com/oracle/oci-go-sdk/v53/containerengine"
	ocicore "github.com/oracle/oci-go-sdk/v53/core"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	hacommon "github.com/verrazzano/verrazzano/tests/e2e/pkg/ha"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
)

const (
	clusterIDEnvVar   = "OKE_CLUSTER_ID"
	ociRegionEnvVar   = "OCI_CLI_REGION"
	skipUpgradeEnvVar = "SKIP_KUBERNETES_UPGRADE"

	waitTimeout     = 20 * time.Minute
	pollingInterval = 30 * time.Second

	waitForDeleteTimeout = 600 * time.Second

	mysqlComponentLabel = "component"
	mysqldComponentName = "mysqld"
)

var clientset = k8sutil.GetKubernetesClientsetOrDie()
var t = framework.NewTestFramework("in_place_upgrade")

var (
	failed                    bool
	region                    string
	clusterID                 string
	skipClusterVersionUpgrade bool

	okeClient     ocice.ContainerEngineClient
	computeClient ocicore.ComputeClient
)

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.BeforeSuite(func() {
	clusterID = os.Getenv(clusterIDEnvVar)
	region = os.Getenv(ociRegionEnvVar)

	if skipUpgradeVal, set := os.LookupEnv(skipUpgradeEnvVar); set {
		var parseErr error
		skipClusterVersionUpgrade, parseErr = strconv.ParseBool(skipUpgradeVal)
		Expect(parseErr).ShouldNot(HaveOccurred(), fmt.Sprintf("Invalid value for %s: %s", skipUpgradeVal, skipUpgradeVal))
	}

	Expect(clusterID).ToNot(BeEmpty(), fmt.Sprintf("%s env var must be set", clusterIDEnvVar))
	// region is optional so don't Expect

	var provider common.ConfigurationProvider
	var err error
	provider, err = getOCISDKProvider(region)
	Expect(err).ShouldNot(HaveOccurred(), "Error configuring OCI SDK provider")

	okeClient, err = ocice.NewContainerEngineClientWithConfigurationProvider(provider)
	Expect(err).ShouldNot(HaveOccurred(), "Error configuring OCI SDK container engine client")

	computeClient, err = ocicore.NewComputeClientWithConfigurationProvider(provider)
	Expect(err).ShouldNot(HaveOccurred(), "Error configuring OCI SDK compute client")
})

var _ = t.AfterSuite(func() {
	// signal that the upgrade is done so the tests know to stop
	hacommon.EventuallyCreateShutdownSignal(clientset, t.Logs)
})

var _ = t.Describe("OKE In-Place Upgrade", Label("f:platform-lcm:ha"), func() {
	var clusterResponse ocice.GetClusterResponse
	var upgradeVersion string

	t.It("upgrades the control plane Kubernetes version", func() {
		if skipClusterVersionUpgrade {
			t.Logs.Infof("%s=%v, skipping cluster Control Plane upgrade", skipUpgradeEnvVar, skipClusterVersionUpgrade)
			return
		}

		// first get the cluster details and find the available upgrade versions
		var err error
		clusterResponse, err = okeClient.GetCluster(context.Background(), ocice.GetClusterRequest{ClusterId: &clusterID})
		Expect(err).ShouldNot(HaveOccurred())
		t.Logs.Debugf("Cluster response: %+v", clusterResponse)
		Expect(clusterResponse.AvailableKubernetesUpgrades).ToNot(BeEmpty(), "No available upgrade versions")

		// upgrade the control plane to the first available upgrade version
		upgradeVersion = clusterResponse.AvailableKubernetesUpgrades[0]
		t.Logs.Infof("Upgrading the OKE cluster control plane to version: %s", upgradeVersion)
		details := ocice.UpdateClusterDetails{KubernetesVersion: &upgradeVersion}
		updateResponse, err := okeClient.UpdateCluster(context.Background(), ocice.UpdateClusterRequest{ClusterId: &clusterID, UpdateClusterDetails: details})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(updateResponse.OpcWorkRequestId).ShouldNot(BeNil())

		// wait for the work request to complete, this can take roughly 5-15 minutes
		waitForWorkRequest(*updateResponse.OpcWorkRequestId)
	})

	t.It("upgrades the node pool Kubernetes version", func() {
		if skipClusterVersionUpgrade {
			t.Logs.Infof("%s=%v, skipping node pool upgrade", skipUpgradeEnvVar, skipClusterVersionUpgrade)
			return
		}
		// first get the node pool, the cluster response struct does not have node pools so we have to list the node pools
		// in the compartment and filter by the cluster id
		nodePoolsResponse, err := okeClient.ListNodePools(context.Background(), ocice.ListNodePoolsRequest{CompartmentId: clusterResponse.CompartmentId, ClusterId: clusterResponse.Id})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(nodePoolsResponse.Items)).To(Equal(1))

		// upgrade the node pool to the same Kubernetes version as the control plane
		t.Logs.Infof("Upgrading the OKE cluster node pool to version: %s", upgradeVersion)
		details := ocice.UpdateNodePoolDetails{KubernetesVersion: &upgradeVersion}
		updateResponse, err := okeClient.UpdateNodePool(context.Background(), ocice.UpdateNodePoolRequest{NodePoolId: nodePoolsResponse.Items[0].Id, UpdateNodePoolDetails: details})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(updateResponse.OpcWorkRequestId).ShouldNot(BeNil())

		// wait for the work request to complete
		waitForWorkRequest(*updateResponse.OpcWorkRequestId)
	})

	t.It("replaces each worker node in the node pool", func() {
		// get the nodes
		nodes := hacommon.EventuallyGetNodes(clientset, t.Logs)
		latestNodes := nodes
		for _, node := range nodes.Items {
			if !hacommon.IsControlPlaneNode(node) {
				// cordon and drain the node - this function is implemented in kubectl itself and is not available
				// using a k8s client
				t.Logs.Infof("Draining node: %s", node.Name)
				kubectlArgs := []string{
					"drain",
					"--ignore-daemonsets",
					"--delete-emptydir-data",
					"--force",
					fmt.Sprintf("--skip-wait-for-delete-timeout=%v", int(waitForDeleteTimeout.Seconds())),
					"--timeout=15m",
					node.Name,
				}
				out, err := exec.Command("kubectl", kubectlArgs...).CombinedOutput() //nolint:gosec //#nosec G204
				t.Logs.Infof("Combined output from kubectl drain command: %s", out)
				if err != nil {
					t.Logs.Infof("Error occurred running kubectl drain command: %s", err.Error())
				}
				Expect(err).ShouldNot(HaveOccurred())

				// Handle the case where MySQL pods can be left dangling waiting for finalizers to be cleaned up
				// - this is a workaround until we can perhaps get a fix from the MySQL team
				cleanupDanglingMySQLPods(clientset)

				// terminate the compute instance that the node is on, OKE will replace it with a new node
				// running the upgraded Kubernetes version
				t.Logs.Infof("Terminating compute instance: %s", node.Spec.ProviderID)
				err = terminateComputeInstance(node.Spec.ProviderID)
				Expect(err).ShouldNot(HaveOccurred())

				latestNodes, err = waitForReplacementNode(latestNodes)
				Expect(err).ShouldNot(HaveOccurred())

				// Handle the case where MySQL pods are waiting for all readiness gates to be met.  This condition
				// may be the result of manually removing the finalizers on a dangling MySQL pod.  So this work around
				// could end up being resolved by the same issue causing the dangling MySQL pods.
				repairMySQLPodsWaitingReadinessGates(clientset)

				// wait for all pods to be ready before continuing to the next node
				t.Logs.Infof("Waiting for all pods to be ready")
				hacommon.EventuallyPodsReady(t.Logs, clientset)
			}
		}
	})

	t.It("validates the k8s version of each worker node in the node pool", func() {
		if skipClusterVersionUpgrade {
			t.Logs.Infof("%s=%v, skipping node pool verification", skipUpgradeEnvVar, skipClusterVersionUpgrade)
			return
		}
		// get the nodes and check both the kube proxy and kubelet versions
		nodes := hacommon.EventuallyGetNodes(clientset, t.Logs)
		for _, node := range nodes.Items {
			Expect(node.Status.NodeInfo.KubeProxyVersion).To(Equal(upgradeVersion), "kube proxy version is incorrect")
			Expect(node.Status.NodeInfo.KubeletVersion).To(Equal(upgradeVersion), "kubelet version is incorrect")
		}
	})
})

// cleanupDanglingMySQLPods - workaround to clean up any dangling MySQL pods, workaround for case where finalizers don't get cleaned up
func cleanupDanglingMySQLPods(client *kubernetes.Clientset) {
	Eventually(func() error {
		t.Logs.Info("Cleaning up any dangling MySQL pods after node drain")
		mysqldReq, err := labels.NewRequirement(mysqlComponentLabel, selection.Equals, []string{mysqldComponentName})
		if err != nil {
			return err
		}
		selector := labels.NewSelector().Add(*mysqldReq)
		selector = selector.Add(*mysqldReq)

		list, err := client.CoreV1().Pods(constants.KeycloakNamespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return err
		}
		if len(list.Items) == 0 {
			return fmt.Errorf("No pods found matching selector %s", selector.String())
		}
		for i := range list.Items {
			mysqlPod := list.Items[i]
			deletionTimestamp := mysqlPod.ObjectMeta.DeletionTimestamp
			if !deletionTimestamp.IsZero() {
				diff := metav1.Now().Sub(deletionTimestamp.Time)
				t.Logs.Infof("Pod %s/%s deletion time: %s, difference: %v", mysqlPod.Namespace, mysqlPod.Name, deletionTimestamp.String(), diff.Seconds())
				if diff.Seconds() >= waitForDeleteTimeout.Seconds() {
					t.Logs.Infof("Found dangling MySQL pod %s/%s, patching out finalizers %v", mysqlPod.Namespace, mysqlPod.Name, mysqlPod.Finalizers)
					mysqlPod.Finalizers = []string{}
					_, err := client.CoreV1().Pods(constants.KeycloakNamespace).Update(context.TODO(), &mysqlPod, metav1.UpdateOptions{})
					if err != nil {
						t.Logs.Errorf("Error occurred patching finalizers for %s/%s, %s", mysqlPod.Namespace, mysqlPod.Name, err.Error())
						return err
					}
				}
			}
		}
		return nil
	}).WithTimeout(waitTimeout).WithPolling(pollingInterval).ShouldNot(HaveOccurred())
}

// repairMySQLPodsWaitingReadinessGates - workaround to repair any MySQL pods that are stuck starting up due
// to readiness gates not all true.
func repairMySQLPodsWaitingReadinessGates(client *kubernetes.Clientset) {
	Eventually(func() error {
		t.Logs.Info("Cleaning up any MySQL pods stuck restarting after a node drain")
		mysqldReq, err := labels.NewRequirement(mysqlComponentLabel, selection.Equals, []string{mysqldComponentName})
		if err != nil {
			return err
		}
		selector := labels.NewSelector().Add(*mysqldReq)
		selector = selector.Add(*mysqldReq)

		list, err := client.CoreV1().Pods(constants.KeycloakNamespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return err
		}
		if len(list.Items) == 0 {
			return fmt.Errorf("No pods found matching selector %s", selector.String())
		}

		for i := range list.Items {
			mysqlPod := list.Items[i]

			// Check if the readiness conditions have been met
			conditions := mysqlPod.Status.Conditions
			if len(conditions) == 0 {
				return fmt.Errorf("no status conditions found for pod %s/%s", mysqlPod.Namespace, mysqlPod.Name)
			}
			for _, condition := range conditions {
				if condition.Type == "mysql.oracle.com/configured" || condition.Type == "mysql.oracle.com/ready" {
					if condition.Status != corev1.ConditionTrue {
						// Restart the mysql-operator to see if it will finish setting the readiness gates
						t.Logs.Info("Restarting the mysql-operator to see if it will repair MySQL pods stuck waiting for readiness gates")
						err := client.CoreV1().Pods(constants.Keycloak).Delete(context.TODO(), mysqlPod.Name, metav1.DeleteOptions{})
						if err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	}).WithTimeout(waitTimeout).WithPolling(pollingInterval).ShouldNot(HaveOccurred())
}

// waitForWorkRequest waits for the work request to transition to success
func waitForWorkRequest(workRequestID string) {
	Eventually(func() (ocice.WorkRequestStatusEnum, error) {
		t.Logs.Infof("Waiting for work request with id %s to complete", workRequestID)
		workRequestResponse, err := okeClient.GetWorkRequest(context.Background(), ocice.GetWorkRequestRequest{WorkRequestId: &workRequestID})
		if err != nil {
			return "", err
		}
		t.Logs.Debugf("Work request response: %+v", workRequestResponse)
		return workRequestResponse.Status, nil
	}).WithTimeout(waitTimeout).WithPolling(pollingInterval).Should(Equal(ocice.WorkRequestStatusSucceeded))
}

// terminateComputeInstance terminates a compute instance
func terminateComputeInstance(instanceID string) error {
	_, err := computeClient.TerminateInstance(context.Background(), ocicore.TerminateInstanceRequest{InstanceId: &instanceID})
	if err != nil {
		return err
	}
	return nil
}

// waitForReplacementNode waits for a replacement node to be ready. It returns the new list of nodes that includes
// the replacement node.
func waitForReplacementNode(existingNodes *corev1.NodeList) (*corev1.NodeList, error) {
	var replacement string
	var latestNodes *corev1.NodeList

	Eventually(func() string {
		t.Logs.Infof("Waiting for replacement worker node")
		latestNodes = hacommon.EventuallyGetNodes(clientset, t.Logs)
		for _, node := range latestNodes.Items {
			if !hacommon.IsControlPlaneNode(node) {
				if !isExistingNode(node, existingNodes) {
					replacement = node.Name
					break
				}
			}
		}
		return replacement
	}).WithTimeout(waitTimeout).WithPolling(pollingInterval).ShouldNot(BeEmpty())

	if len(replacement) == 0 {
		return nil, errors.New("Timed out waiting for new worker to be added to node pool")
	}

	Eventually(func() (bool, error) {
		t.Logs.Infof("Waiting for new worker node %s to be ready", replacement)
		return isNodeReady(replacement)
	}).WithTimeout(waitTimeout).WithPolling(pollingInterval).Should(BeTrue())

	return latestNodes, nil
}

// isExistingNode returns true if the specified node is in the list of existing nodes
func isExistingNode(node corev1.Node, existingNodes *corev1.NodeList) bool {
	for _, existingNode := range existingNodes.Items {
		if node.Name == existingNode.Name {
			return true
		}
	}
	return false
}

// isNodeReady returns true if the NodeReady condition is true
func isNodeReady(name string) (bool, error) {
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true, nil
		}
	}

	return false, nil
}

// getOCISDKProvider returns an OCI SDK configuration provider. If a region is specified then
// use an instance principal auth provider, otherwise use the default provider (auth config comes from
// an OCI config file or environment variables).
func getOCISDKProvider(region string) (common.ConfigurationProvider, error) {
	var provider common.ConfigurationProvider
	var err error

	if region != "" {
		t.Logs.Infof("Using OCI SDK instance principal provider with region: %s", region)
		provider, err = auth.InstancePrincipalConfigurationProviderForRegion(common.StringToRegion(region))
	} else {
		t.Logs.Info("Using OCI SDK default provider")
		provider = common.DefaultConfigProvider()
	}

	if err != nil {
		return nil, err
	}

	defaultRetryPolicy := common.DefaultRetryPolicy()
	common.GlobalRetry = &defaultRetryPolicy
	return provider, nil
}
