// Copyright (C) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package topology

import (
	"context"
	"fmt"
	. "github.com/onsi/gomega"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vmoClient "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

const (
	namespace    = "verrazzano-system"
	vmiName      = "system"
	timeout      = 15 * time.Minute
	pollInterval = 15 * time.Second
)

var t = framework.NewTestFramework("topology")
var client *vmoClient.Clientset

var _ = t.BeforeSuite(func() {
	var err error
	client, err = vmiClientFromConfig()
	Expect(err).To(BeNil())
})

var _ = t.Describe("OpenSearch Cluster Topology", func() {
	if pkg.IsDevProfile() {
		t.It("Scales the dev profile", func() {
			eventuallyPodReady(1, 0, 0)
			Eventually(func() bool {
				vmi, err := getVMI()
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to get vmi from dev profile: %v", err))
					return false
				}
				vmi.Spec.Elasticsearch.MasterNode.Replicas = 3
				vmi.Spec.Elasticsearch.MasterNode.Roles = []vmov1.NodeRole{
					vmov1.MasterRole,
					vmov1.DataRole,
					vmov1.IngestRole,
				}
				if err := patchVMI(vmi); err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to patch vmi from dev profile: %v", err))
				}
				return true
			}, timeout, pollInterval).Should(BeTrue())
			// 3 of each node role is expected, since we have a 3-node cluster with all roles on each node
			eventuallyPodReady(3, 3, 3)
		})
	} else {
		t.It("Adds node groups to the prod profile", func() bool {
			eventuallyPodReady(3, 3, 1)
			vmi, err := getVMI()
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("failed to get vmi from prod profile: %v", err))
				return false
			}
			vmi.Spec.Elasticsearch.Nodes = append(vmi.Spec.Elasticsearch.Nodes, vmov1.ElasticsearchNode{
				Name:     "data-ingest",
				Replicas: 3,
				Roles: []vmov1.NodeRole{
					vmov1.DataRole,
					vmov1.IngestRole,
				},
				Resources: vmov1.Resources{
					RequestMemory: "48Mi",
				},
				Storage: &vmov1.Storage{
					Size: "3Gi",
				},
			})
			if err := patchVMI(vmi); err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("failed to patch vmi from prod profile: %v", err))
			}
			// prod has 3 master, 3 data, 1 ingest nodes. add 3 data+ingest nodes and you have 3 master, 6 data, and 4 ingest nodes.
			eventuallyPodReady(3, 6, 4)
			return true
		})
	}
})

func eventuallyPodReady(master, data, ingest int) {
	Eventually(func() bool {
		if err := verifyReadyReplicas(master, data, ingest); err != nil {
			t.Logs.Errorf("pods not ready: %v", err)
			return false
		}
		return true
	}, timeout, pollInterval).Should(BeTrue())
}

func verifyReadyReplicas(master, data, ingest int) error {
	client, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	if err := assertPodsFound(client, master, labelSelector("master")); err != nil {
		return err
	}
	if err := assertPodsFound(client, data, labelSelector("data")); err != nil {
		return err
	}
	if err := assertPodsFound(client, ingest, labelSelector("ingest")); err != nil {
		return err
	}
	return nil
}

func labelSelector(label string) string {
	return fmt.Sprintf("opensearch.verrazzano.io/role-%s=true", label)
}

func assertPodsFound(clientSet *kubernetes.Clientset, count int, selector string) error {
	pods, err := clientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	if len(pods.Items) != count {
		return fmt.Errorf("expected %d pods, found %d", count, len(pods.Items))
	}
	for _, pod := range pods.Items {
		for _, status := range pod.Status.ContainerStatuses {
			if !status.Ready {
				return fmt.Errorf("container %s/%s is not yet ready", pod.Name, status.Name)
			}
		}
		if pod.Status.Phase != corev1.PodRunning {

		}
	}
	return nil
}

func vmiClientFromConfig() (*vmoClient.Clientset, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	return vmoClient.NewForConfig(config)
}

func getVMI() (*vmov1.VerrazzanoMonitoringInstance, error) {
	return client.VerrazzanoV1().VerrazzanoMonitoringInstances(namespace).Get(context.TODO(), vmiName, metav1.GetOptions{})
}

func patchVMI(vmi *vmov1.VerrazzanoMonitoringInstance) error {
	_, err := client.VerrazzanoV1().VerrazzanoMonitoringInstances(namespace).Update(context.TODO(), vmi, metav1.UpdateOptions{})
	return err
}
