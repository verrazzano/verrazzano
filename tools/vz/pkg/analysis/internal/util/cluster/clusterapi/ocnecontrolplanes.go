// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
)

const ocneControlPlaneResource = "ocnecontrolplanes.controlplane.cluster.x-k8s.io"

// Minimal definition of controlplanes.controlplane.x-k8s.io object that only contains the fields that will be analyzed.
type ocneControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ocneControlPlane `json:"items"`
}
type ocneControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            ocneControlPlaneStatus `json:"status,omitempty"`
}
type ocneControlPlaneStatus struct {
	Conditions []ocneControlPlaneCondition `json:"conditions,omitempty"`
}
type ocneControlPlaneCondition struct {
	Message string                 `json:"message,omitempty"`
	Reason  string                 `json:"reason,omitempty"`
	Status  corev1.ConditionStatus `json:"status"`
	Type    string                 `json:"type"`
}

// AnalyzeOCNEControlPlane handles the checking of the status of cluster-qpi ocneControlPlane resources.
func AnalyzeOCNEControlPlane(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &ocneControlPlaneList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "ocnecontrolplane.controlplane.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, ocneControlPlane := range list.Items {
		analyzeOCNEControlPlane(clusterRoot, ocneControlPlane, issueReporter)
	}

	return nil
}

// analyzeOCNEControlPlane - analyze a single cluster API ocneControlPlane and report any issues
func analyzeOCNEControlPlane(clusterRoot string, ocneControlPlane ocneControlPlane, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range ocneControlPlane.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Ready":
				subMessage = "is not ready"
			case "Available":
				subMessage = "is not available"
			case "CertificatesAvailable":
				subMessage = "certificates are not available"
			case "MachinesCreated":
				subMessage = "machines are not created"
			case "MachinesReady":
				subMessage = "machines are not ready"
			case "Resized":
				subMessage = "is not resized"
			case "ControlPlaneComponentsHealthy":
				subMessage = "control plane is not healthy"
			case "APIServerPodHealthy":
				subMessage = "kube-apiserver pod is not healthy"
			case "ControllerManagerPodHealthy":
				subMessage = "kube-controller-manager pod is not healthy"
			case "SchedulerPodHealthy":
				subMessage = "kube-scheduler pod is not healthy"
			case "EtcdPodHealthy":
				subMessage = "machine's is not healthy"
			case "EtcdClusterHealthy":
				subMessage = "cluster's etcd is not healthy"
			case "ModuleOperatorDeployed":
				subMessage = "Module Operator is not deployed"
			case "VerrazzanoPlatformOperatorDeployDeployed":
				subMessage = "Verrazzano Platform Operator is not deployed"
			case "MachinesSpecUpToDate":
				subMessage = "machines are not up-to-date"
			default:
				continue
			}
			// Add a message for the issue
			var message string
			if len(condition.Reason) == 0 {
				message = fmt.Sprintf("%q resource %q in namespace %q, %s", ocneControlPlaneResource, ocneControlPlane.Name, ocneControlPlane.Namespace, subMessage)
			} else {
				message = fmt.Sprintf("%q resource %q in namespace %q, %s - reason is %s", ocneControlPlaneResource, ocneControlPlane.Name, ocneControlPlane.Namespace, subMessage, condition.Reason)
			}
			messages = append([]string{message}, messages...)

		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterNotReady, clusterRoot, messages, []string{})
	}
}
