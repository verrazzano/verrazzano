// Copyright (C) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package topology

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vmoClient "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	systemNamespace = "verrazzano-system"
	verrazzanoName  = "verrazzano"
	vmiName         = "testing"
	timeout         = 20 * time.Minute
	pollInterval    = 15 * time.Second
	requestMemory   = "1Gi"
)

var (
	t             = framework.NewTestFramework("topology")
	namespace     = pkg.GenerateNamespace("vmi")
	client        *vmoClient.Clientset
	kubeClientSet *kubernetes.Clientset
)

var _ = t.BeforeSuite(func() {
	var err error
	client, err = vmiClientFromConfig()
	Expect(err).To(BeNil())
	kubeClientSet, err = k8sutil.GetKubernetesClientset()
	Expect(err).To(BeNil())

	Eventually(func() bool {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		_, err = pkg.CreateNamespace(namespace, nsLabels)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return false
		}
		err = copySecret(verrazzanoName, systemNamespace, namespace)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return false
		}
		err = copySecret("verrazzano-local-registration", systemNamespace, namespace)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return false
		}
		_, err = kubeClientSet.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "verrazzano-monitoring-operator",
			},
		}, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return false
		}
		return true
	}, timeout, pollInterval).Should(BeTrue())
})

var _ = t.AfterSuite(func() {
	Eventually(func() bool {
		if err := client.VerrazzanoV1().VerrazzanoMonitoringInstances(namespace).Delete(context.TODO(), vmiName, metav1.DeleteOptions{}); err != nil &&
			!apierrors.IsNotFound(err) {
			t.Logs.Errorf("failed to delete vmi: %v", err)
			return false
		}

		if err := pkg.DeleteNamespace(namespace); err != nil {
			t.Logs.Errorf("failed to cleanup namespace: %v", err)
			return false
		}
		return true
	}, timeout, pollInterval).Should(BeTrue())
})

var _ = t.Describe("OpenSearch Cluster Topology", func() {
	t.It("can scale the cluster", func() {
		// Initialize the single node cluster
		Eventually(func() bool {
			_, err := createSingleNodeVMI()
			if err != nil {
				t.Logs.Errorf("failed to create single node cluster: %v", err)
				return false
			}
			return true
		}, timeout, pollInterval).Should(BeTrue())
		eventuallyPodsReady(1, 1, 1)

		t.Logs.Info("Adding 2 master/data/ingest nodes")
		eventuallyUpdateVMI(t, func(vmi *vmov1.VerrazzanoMonitoringInstance) {
			vmi.Spec.Elasticsearch.MasterNode.Replicas = 3
		})
		eventuallyPodsReady(3, 3, 3)

		t.Logs.Info("Adding 3 data/ingest nodes and 1 master node")
		eventuallyUpdateVMI(t, func(vmi *vmov1.VerrazzanoMonitoringInstance) {
			vmi.Spec.Elasticsearch.Nodes = []vmov1.ElasticsearchNode{
				{
					Name:     "data-ingest",
					Replicas: 3,
					Roles: []vmov1.NodeRole{
						vmov1.DataRole,
						vmov1.IngestRole,
					},
					Resources: vmov1.Resources{
						RequestMemory: requestMemory,
					},
					Storage: &vmov1.Storage{
						Size: "5Gi",
					},
				},
				{
					Name:     "master",
					Replicas: 1,
					Roles: []vmov1.NodeRole{
						vmov1.MasterRole,
					},
					Resources: vmov1.Resources{
						RequestMemory: requestMemory,
					},
					Storage: &vmov1.Storage{
						Size: "5Gi",
					},
				},
			}
		})
		eventuallyPodsReady(4, 6, 6)

		t.Logs.Info("Removing 1 master/data/ingest nodes")
		eventuallyUpdateVMI(t, func(vmi *vmov1.VerrazzanoMonitoringInstance) {
			vmi.Spec.Elasticsearch.MasterNode.Replicas = 2
		})
		eventuallyPodsReady(3, 5, 5)
	})
})

func eventuallyPodsReady(master, data, ingest int) {
	Eventually(func() bool {
		if err := verifyReadyReplicas(master, data, ingest); err != nil {
			t.Logs.Errorf("pods not ready: %v", err)
			return false
		}
		return true
	}, timeout, pollInterval).Should(BeTrue())
}

func verifyReadyReplicas(master, data, ingest int) error {
	if err := assertPodsFound(kubeClientSet, master, labelSelector("master")); err != nil {
		return err
	}
	if err := assertPodsFound(kubeClientSet, data, labelSelector("data")); err != nil {
		return err
	}
	if err := assertPodsFound(kubeClientSet, ingest, labelSelector("ingest")); err != nil {
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

func eventuallyUpdateVMI(t *framework.TestFramework, updaterFunc func(vmi *vmov1.VerrazzanoMonitoringInstance)) {
	Eventually(func() bool {
		vmi, err := getVMI()
		if err != nil {
			t.Logs.Errorf("failed to get vmi: %v", err)
			return false
		}
		updaterFunc(vmi)
		if err := patchVMI(vmi); err != nil {
			t.Logs.Errorf("failed to patch vmi: %v", err)
			return false
		}
		return true
	}, timeout, pollInterval).Should(BeTrue())
}

func getVMI() (*vmov1.VerrazzanoMonitoringInstance, error) {
	return client.VerrazzanoV1().VerrazzanoMonitoringInstances(namespace).Get(context.TODO(), vmiName, metav1.GetOptions{})
}

func patchVMI(vmi *vmov1.VerrazzanoMonitoringInstance) error {
	_, err := client.VerrazzanoV1().VerrazzanoMonitoringInstances(namespace).Update(context.TODO(), vmi, metav1.UpdateOptions{})
	return err
}

func createSingleNodeVMI() (*vmov1.VerrazzanoMonitoringInstance, error) {
	vmi := &vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiName,
			Labels: map[string]string{
				"k8s-app":              "verrazzano.io",
				"managed-cluster-name": "",
				"verrazzano.binding":   vmiName,
			},
		},
		Spec: vmov1.VerrazzanoMonitoringInstanceSpec{
			SecretsName: verrazzanoName,
			Elasticsearch: vmov1.Elasticsearch{
				Enabled: true,
				MasterNode: vmov1.ElasticsearchNode{
					Name:     "es-master",
					Replicas: 1,
					Roles: []vmov1.NodeRole{
						vmov1.MasterRole,
						vmov1.DataRole,
						vmov1.IngestRole,
					},
					Storage: &vmov1.Storage{
						Size: "5Gi",
					},
					Resources: vmov1.Resources{
						RequestMemory: requestMemory,
					},
				},
			},
		},
	}

	return client.VerrazzanoV1().VerrazzanoMonitoringInstances(namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
}

func copySecret(secretName, src, dest string) error {
	secret, err := kubeClientSet.CoreV1().Secrets(src).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	secretCopy := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: dest,
		},
		Data: secret.Data,
	}
	_, err = kubeClientSet.CoreV1().Secrets(dest).Create(context.TODO(), secretCopy, metav1.CreateOptions{})
	return err
}
