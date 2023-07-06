// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

type CAPITestImpl struct{}

// NewCapiTestClient creates a new CAPI Test Client
func NewCapiTestClient() CapiTestClient {
	return &CAPITestImpl{}
}

type CapiTestClient interface {
	PrintYamlOutput(printer clusterapi.YamlPrinter, outputFile string) error
	ClusterTemplateGenerate(clusterName, templatePath string, log *zap.SugaredLogger) (string, error)
	GetUnstructuredData(group, version, resource, resourceName, nameSpaceName string, log *zap.SugaredLogger) (*unstructured.Unstructured, error)
	GetCluster(namespace, clusterName string, log *zap.SugaredLogger) (*Cluster, error)
	GetOCNEControlPlane(namespace, controlPlaneName string, log *zap.SugaredLogger) (*OCNEControlPlane, error)
	GetCapiClusterKubeConfig(clusterName string, log *zap.SugaredLogger) ([]byte, error)
	GetCapiClusterK8sClient(clusterName string, log *zap.SugaredLogger) (client *kubernetes.Clientset, err error)
	TriggerCapiClusterCreation(clusterName, templateName string, log *zap.SugaredLogger) error
	DeployClusterResourceSets(clusterName, templateName string, log *zap.SugaredLogger) error
	EnsureMachinesAreProvisioned(namespace, clusterName string, log *zap.SugaredLogger) error
	MonitorCapiClusterDeletion(clusterName string, log *zap.SugaredLogger) error
	MonitorCapiClusterCreation(clusterName string, log *zap.SugaredLogger) error
	TriggerCapiClusterDeletion(clusterName, nameSpaceName string, log *zap.SugaredLogger) error
	ShowNodeInfo(client *kubernetes.Clientset, clustername string, log *zap.SugaredLogger) error
	ShowPodInfo(client *kubernetes.Clientset, clusterName string, log *zap.SugaredLogger) error
	DisplayWorkloadClusterResources(clusterName string, log *zap.SugaredLogger) error
	UpdateOCINSG(clusterName, nsgDisplayNameToUpdate, nsgDisplayNameInRule, info string, rule *SecurityRuleDetails, log *zap.SugaredLogger) error
	CreateImagePullSecrets(clusterName string, log *zap.SugaredLogger) error
	ProcessOCIPrivateKeysBase64(file, key string, log *zap.SugaredLogger) error
	ProcessOCISSHKeys(file, key string, log *zap.SugaredLogger) error
	ProcessOCIPrivateKeysSingleLine(file, key string, log *zap.SugaredLogger) error
	CreateNamespace(namespace string, log *zap.SugaredLogger) error
	SetImageID(key string, log *zap.SugaredLogger) error
	GetCapiClusterDynamicClient(clusterName string, log *zap.SugaredLogger) (dynamic.Interface, error)
	GetVerrazzano(clusterName, namespace, vzinstallname string, log *zap.SugaredLogger) (*unstructured.Unstructured, error)
	EnsureVerrazzano(clusterName string, log *zap.SugaredLogger) error
}
