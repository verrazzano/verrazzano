// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	nodeLabel         = "node-role.kubernetes.io/node"
	controlPlaneLabel = "node-role.kubernetes.io/control-plane"

	// env var names
	CAPINodeSSHKey               = "OCI_SSH_KEY"
	OCICredsKey                  = "OCI_CREDENTIALS_KEY"
	OCIPrivateCredsKeyBase64     = "OCI_CREDENTIALS_KEY_B64"
	OCITenancyIDKeyBase64        = "OCI_TENANCY_ID_B64"
	OCICredsFingerprintKeyBase64 = "OCI_CREDENTIALS_FINGERPRINT_B64"
	OCIUserIDKeyBase64           = "OCI_USER_ID_B64"
	OCIRegionKeyBase64           = "OCI_REGION_B64"
	OCIImageIDKey                = "OCI_IMAGE_ID"
	OCIVCNKey                    = "OCI_VCN_ID"
	OCISubnetKey                 = "OCI_SUBNET_ID"
	ocneServiceLbName            = "ocne-service-lb"
)

var capiInitFunc = clusterapi.New

// PrintYamlOutput is used to print yaml templates to stdout or a file
func (c CAPITestImpl) PrintYamlOutput(printer clusterapi.YamlPrinter, outputFile string) error {
	yaml, err := printer.Yaml()
	if err != nil {
		return err
	}
	yaml = append(yaml, '\n')
	outputFile = strings.TrimSpace(outputFile)
	if outputFile == "" || outputFile == "-" {
		if _, err := os.Stdout.Write(yaml); err != nil {
			return errors.Wrap(err, "failed to write yaml to Stdout")
		}
		return nil
	}
	outputFile = filepath.Clean(outputFile)
	if err := os.WriteFile(outputFile, yaml, 0600); err != nil {
		return errors.Wrap(err, "failed to write to destination file")
	}
	return nil
}

// ClusterTemplateGenerate used for cluster template generation
func (c CAPITestImpl) ClusterTemplateGenerate(clusterName, templatePath string, log *zap.SugaredLogger) (string, error) {
	log.Infof("Generate called for clustername '%s'...", clusterName)
	capiClient, err := capiInitFunc("")
	if err != nil {
		return "", err
	}
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig url due to %v", zap.Error(err))
		return "", err
	}

	templateOptions := clusterapi.GetClusterTemplateOptions{
		Kubeconfig: clusterapi.Kubeconfig{
			Path:    kubeconfigPath,
			Context: ""},
		URLSource: &clusterapi.URLSourceOptions{
			URL: templatePath,
		},
		ClusterName:     clusterName,
		TargetNamespace: OCNENamespace,
	}

	template, err := capiClient.GetClusterTemplate(templateOptions)
	if err != nil {
		log.Errorf("template '%s' generation error  = %v", templatePath, zap.Error(err))
		return "", err
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		return "", fmt.Errorf("Failed to create temporary file: %v", err)
	}

	if err := c.PrintYamlOutput(template, tmpFile.Name()); err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

// GetUnstructuredData common utility to fetch unstructured data
func (c CAPITestImpl) GetUnstructuredData(group, version, resource, resourceName, nameSpaceName string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	var dataFetched *unstructured.Unstructured
	var err error
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return nil, err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	if nameSpaceName != "" {
		log.Infof("fetching '%s' '%s' in namespace '%s'", resource, resourceName, nameSpaceName)
		dataFetched, err = dclient.Resource(gvr).Namespace(nameSpaceName).Get(context.TODO(), resourceName, metav1.GetOptions{})
	} else {
		log.Infof("fetching '%s' '%s'", resource, resourceName)
		dataFetched, err = dclient.Resource(gvr).Get(context.TODO(), resourceName, metav1.GetOptions{})
	}
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Errorf("resource %s %s not found", resource, resourceName)
			return nil, nil
		}
		log.Errorf("Unable to fetch %s %s due to '%v'", resource, resourceName, zap.Error(err))
		return nil, err
	}
	return dataFetched, nil
}

// GetUnstructuredData common utility to fetch list of unstructured data
func (c CAPITestImpl) GetUnstructuredDataList(group, version, resource, nameSpaceName string, log *zap.SugaredLogger) (*unstructured.UnstructuredList, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return nil, err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return nil, err
	}

	log.Infof("Fetching resource %s", resource)
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	dataFetched, err := dclient.Resource(gvr).Namespace(nameSpaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Unable to fetch resource %s due to '%v'", resource, zap.Error(err))
		return nil, err
	}
	return dataFetched, nil
}

// GetCluster is used to fetch capi cluster info given a cluster name and namespace, if it exists
func (c CAPITestImpl) GetCluster(namespace, clusterName string, log *zap.SugaredLogger) (*Cluster, error) {
	var capiCluster Cluster
	clusterFetched, err := c.GetUnstructuredData("cluster.x-k8s.io", "v1beta1", "clusters", clusterName, namespace, log)
	if err != nil {
		log.Errorf("Unable to fetch CAPI cluster '%s' due to '%v'", clusterName, zap.Error(err))
		return nil, err
	}

	if clusterFetched == nil {
		log.Infof("No CAPI clusters with name '%s' in namespace '%s' was detected", clusterName, namespace)
		return &capiCluster, nil
	}

	bdata, err := json.Marshal(clusterFetched)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(bdata, &capiCluster)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return nil, err
	}

	return &capiCluster, nil
}

// GetOCNEControlPlane is used to fetch OCNE control plane info given a control plane name and namespace, if it exists
func (c CAPITestImpl) GetOCNEControlPlane(namespace string, log *zap.SugaredLogger) (*OCNEControlPlane, error) {
	ocnecpesFetched, err := c.GetUnstructuredDataList("controlplane.cluster.x-k8s.io", "v1alpha1", "ocnecontrolplanes", namespace, log)
	if err != nil {
		log.Errorf("Unable to fetch machines due to '%v'", zap.Error(err))
		return nil, err
	}

	if ocnecpesFetched == nil {
		log.Infof("No OCNE control plane in namespace '%s' was detected", namespace)
	}

	var ocneControlPlane OCNEControlPlane
	bdata, err := json.Marshal(ocnecpesFetched.Items[0])
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(bdata, &ocneControlPlane)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return nil, err
	}
	// Set global variable to be used later
	OCNEControlPlaneName = ocneControlPlane.Metadata.Name
	err = os.Setenv("OCNE_CONTROL_PLANE_NAME", ocneControlPlane.Metadata.Name)
	if err != nil {
		log.Errorf("Unable to set env variable OCNE_CONTROL_PLANE_NAME")
		return nil, err
	}
	return &ocneControlPlane, nil
}

// GetCapiClusterKubeConfig returns the content of the kubeconfig file of an OCNE cluster if it exists.
func (c CAPITestImpl) GetCapiClusterKubeConfig(clusterName string, log *zap.SugaredLogger) ([]byte, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return nil, err
	}

	secret, err := clientset.CoreV1().Secrets(clusterName).Get(context.TODO(), fmt.Sprintf("%s-kubeconfig", clusterName), metav1.GetOptions{})
	if err != nil {
		log.Infof("Error fetching secret ", zap.Error(err))
		return nil, err
	}

	return secret.Data["value"], nil
}

// GetCapiClusterK8sClient returns the K8s client of an OCNE cluster if it exists.
func (c CAPITestImpl) GetCapiClusterK8sClient(clusterName string, log *zap.SugaredLogger) (client *kubernetes.Clientset, err error) {
	capiK8sConfig, err := c.GetCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return nil, err
	}
	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create temporary file")
	}

	if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		return nil, errors.Wrap(err, "failed to write to destination file")
	}

	k8sRestConfig, err := k8sutil.GetKubeConfigGivenPathAndContext(tmpFile.Name(), fmt.Sprintf("%s-admin@%s", clusterName, clusterName))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get k8s rest config")
	}

	return k8sutil.GetKubernetesClientsetWithConfig(k8sRestConfig)
}

// TriggerCapiClusterCreation starts the OCNE workload cluster creation by applying the template YAML
func (c CAPITestImpl) TriggerCapiClusterCreation(clusterName, templateName string, log *zap.SugaredLogger) error {
	tmpFilePath, err := c.ClusterTemplateGenerate(clusterName, templateName, log)
	if err != nil {
		log.Errorf("unable to generate template for cluster : %v", zap.Error(err))
		return err
	}
	defer os.RemoveAll(tmpFilePath)
	clusterTemplateData, err := os.ReadFile(tmpFilePath)
	if err != nil {
		return nil
	}
	err = resource.CreateOrUpdateResourceFromBytes(clusterTemplateData, log)
	if err != nil {
		log.Errorf("Error creating cluster from template ", zap.Error(err))
		return err
	}
	log.Infof("Wait for 30 seconds before verification")
	time.Sleep(30 * time.Second)
	return nil
}

// DeployClusterInfraClusterResourceSets deploys the ClusterResourceSets by deploying the addon template YAML
// ClusterResourceSets are used to deploy the following on OCNE workload clusters
// 1. CCM Secrets
// 2. Calico Module
// 3. CCM Module
func (c CAPITestImpl) DeployClusterInfraClusterResourceSets(clusterName, templateName string, log *zap.SugaredLogger) error {
	log.Info("Preparing to deploy Clusterresourcesets...")
	oci, err := NewClient(GetOCIConfigurationProvider(log))
	if err != nil {
		log.Error("Unable to create OCI client %v", zap.Error(err))
		return err
	}

	OCIVcnID, err = oci.GetVcnIDByName(context.TODO(), OCICompartmentID, clusterName, log)
	if err != nil {
		return err
	}

	OCISubnetID, err = oci.GetSubnetIDByName(context.TODO(), OCICompartmentID, OCIVcnID, ocneServiceLbName, log)
	if err != nil {
		return err
	}

	os.Setenv(OCIVCNKey, OCIVcnID)
	os.Setenv(OCISubnetKey, OCISubnetID)

	tmpFilePath, err := c.ClusterTemplateGenerate(clusterName, templateName, log)
	if err != nil {
		log.Errorf("unable to generate template for clusterresourcesets : %v", zap.Error(err))
		return err
	}
	defer os.RemoveAll(tmpFilePath)
	clusterTemplateData, err := os.ReadFile(tmpFilePath)
	if err != nil {
		log.Errorf("unable to get read file : %v", zap.Error(err))
		return err
	}

	err = resource.CreateOrUpdateResourceFromBytes(clusterTemplateData, log)
	if err != nil {
		log.Error("unable to get create clusterresourcesets on workload cluster :", zap.Error(err))
		return err
	}

	log.Infof("Wait for 30 seconds for cluster resourceset resources to deploy")
	time.Sleep(30 * time.Second)
	localTest := getEnvDefault("LOCAL_TEST", "false")
	if strings.ToLower(localTest) == "false" {
		// When running on Jenkins instance
		return c.CreateImagePullSecrets(clusterName, log)
	}
	return nil
}

// DeployAnyClusterResourceSets deploys the VZ ClusterResourceSets by deploying the addon template YAML
func (c CAPITestImpl) DeployAnyClusterResourceSets(clusterName, templateName string, log *zap.SugaredLogger) error {
	log.Info("Preparing to deploy VZ Clusterresourcesets...")

	tmpFilePath, err := c.ClusterTemplateGenerate(clusterName, templateName, log)
	if err != nil {
		log.Errorf("unable to generate template for clusterresourcesets : %v", zap.Error(err))
		return err
	}
	defer os.RemoveAll(tmpFilePath)
	clusterTemplateData, err := os.ReadFile(tmpFilePath)
	if err != nil {
		log.Errorf("unable to get read file : %v", zap.Error(err))
		return err
	}

	err = resource.CreateOrUpdateResourceFromBytes(clusterTemplateData, log)
	if err != nil {
		log.Error("unable to get create clusterresourcesets on workload cluster :", zap.Error(err))
		return err
	}

	log.Infof("Wait for 5 seconds for controllers to reconcile...")
	time.Sleep(5 * time.Second)

	return nil
}

// EnsureMachinesAreProvisioned fetches the machines that are deployed during OCNE cluster creation.
func (c CAPITestImpl) EnsureMachinesAreProvisioned(namespace, clusterName string, log *zap.SugaredLogger) error {
	machinesFetched, err := c.GetUnstructuredDataList("cluster.x-k8s.io", "v1beta1", "machines", namespace, log)
	if err != nil {
		log.Errorf("Unable to fetch machines due to '%v'", zap.Error(err))
		return err
	}

	if machinesFetched == nil {
		log.Infof("No machines for cluster '%s' in namespace '%s' was detected", clusterName, namespace)
	}

	log.Infof("OCNE machine details:")
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tCluster\tNodename\tProviderID\tPhase\tAge\tVersion")

	var healthTracker []bool

	for _, ma := range machinesFetched.Items {
		var machine Machine
		bdata, err := json.Marshal(ma.Object)
		if err != nil {
			log.Errorf("Json marshalling error %v", zap.Error(err))
			return err
		}
		err = json.Unmarshal(bdata, &machine)
		if err != nil {
			log.Errorf("Json unmarshall error %v", zap.Error(err))
			return err
		}
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v",
			machine.Metadata.Name, machine.Metadata.Labels.ClusterXK8SIoClusterName, machine.Status.NodeRef.Name,
			machine.Spec.ProviderID, machine.Status.Phase, time.Until(machine.Metadata.CreationTimestamp).Abs(), machine.Spec.Version))

		if strings.ToLower(machine.Status.Phase) == "running" {
			healthTracker = append(healthTracker, true)
		} else {
			healthTracker = append(healthTracker, false)
		}
	}
	writer.Flush()

	if checkAll(healthTracker) {
		log.Infof("All machines for cluster '%s' in namesapce '%s' are in 'Running' state", clusterName, namespace)
		return nil
	}

	msg := fmt.Sprintf("Not all machines for cluster '%s' in namesapce '%s' are in 'Running' state", clusterName, namespace)
	log.Errorf(msg)
	return fmt.Errorf(msg)
}

func (c CAPITestImpl) MonitorCapiClusterDeletion(clusterName string, log *zap.SugaredLogger) error {
	var err error
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "cluster.x-k8s.io",
		Version:  "v1beta1",
		Resource: "clusters",
	}
	var capiCluster Cluster

	kluster, err := dclient.Resource(gvr).Namespace(OCNENamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Errorf("cluster resource %s not found", clusterName)
				return nil
			}
			log.Errorf("Unable to fetch %s %s due to '%v'", clusterName, zap.Error(err))
			return err
		}
	}
	bdata, err := json.Marshal(kluster)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return err
	}
	err = json.Unmarshal(bdata, &capiCluster)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return err
	}

	return fmt.Errorf("cluster '%s' is in '%s' state", clusterName, capiCluster.Status.Phase)
}

// MonitorCapiClusterCreation fetches the workload OCNE cluster elements and prints them as a formatted table.
// Returns an error if Cluster is not Ready
func (c CAPITestImpl) MonitorCapiClusterCreation(clusterName string, log *zap.SugaredLogger) error {
	klusterData, err := c.GetCluster(OCNENamespace, clusterName, log)
	if err != nil {
		return err
	}

	log.Infof("----------- Cluster Events ---------------------")
	err = c.ShowEvents(OCNENamespace, log)
	if err != nil {
		return err
	}

	log.Infof("----------- OCNE Control Plane ---------------------")
	ocneCP, err := c.GetOCNEControlPlane(OCNENamespace, log)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tCluster\tInitialized\tReplicas\tUpdated\tUnavailable\tReady\tAge")
	fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v",
		ocneCP.Metadata.Name, ocneCP.Metadata.Labels.ClusterXK8SIoClusterName, ocneCP.Status.Initialized, ocneCP.Status.Replicas,
		ocneCP.Status.UpdatedReplicas, ocneCP.Status.UnavailableReplicas, ocneCP.Status.ReadyReplicas, time.Until(ocneCP.Metadata.CreationTimestamp).Abs()))
	writer.Flush()

	log.Infof("----------- CAPI Machines ---------------------")
	err = c.EnsureMachinesAreProvisioned(OCNENamespace, clusterName, log)
	if err != nil {
		return err
	}

	// OCNE cluster is ready when both control plane and worker nodes are up
	switch OCNEK8sVersion {
	case "1.25.7", "1.25.11":
		if klusterData.Status.ControlPlaneReady && klusterData.Status.InfrastructureReady {
			log.Infof("Cluster '%s' phase is => '%s'. All machines are also in '%s' state.", clusterName, klusterData.Status.Phase, klusterData.Status.Phase)
			return nil
		}
	default:
		if klusterData.Status.InfrastructureReady {
			log.Infof("Cluster '%s' phase is => '%s'. All machines are also in '%s' state.", clusterName, klusterData.Status.Phase, klusterData.Status.Phase)
			return nil
		}
	}

	msg := fmt.Sprintf("cluster '%s' not ready. cluster phase is in '%s' state.", clusterName, klusterData.Status.Phase)
	log.Errorf(msg)
	return fmt.Errorf(msg)
}

func (c CAPITestImpl) TriggerCapiClusterDeletion(clusterName, nameSpaceName string, log *zap.SugaredLogger) error {
	var err error
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "cluster.x-k8s.io",
		Version:  "v1beta1",
		Resource: "clusters",
	}

	err = dclient.Resource(gvr).Namespace(nameSpaceName).Delete(context.TODO(), clusterName, metav1.DeleteOptions{})
	if err != nil {
		log.Errorf("Unable to delete cluster %s due to '%v'", clusterName, zap.Error(err))
		return err
	}
	return nil
}

// ShowNodeInfo displays the nodes of workload OCNE cluster as a formatted table.
func (c CAPITestImpl) ShowNodeInfo(client *kubernetes.Clientset, clustername string, log *zap.SugaredLogger) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tRole\tVersion\tInternalIP\tExternalIP\tOSImage\tKernelVersion\tContainerRuntime\tAge")
	nodeList, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get list of nodes from cluster '%s'", clustername))
	}
	for _, node := range nodeList.Items {
		labels := node.GetLabels()
		_, nodeOK := labels[nodeLabel]
		_, controlPlaneOK := labels[nodeLabel]
		var role, internalIP string
		if nodeOK {
			role = strings.Split(nodeLabel, "/")[len(strings.Split(nodeLabel, "/"))-1]
		}
		if controlPlaneOK {
			role = strings.Split(controlPlaneLabel, "/")[len(strings.Split(controlPlaneLabel, "/"))-1]
		}

		addresses := node.Status.Addresses
		for _, address := range addresses {
			if address.Type == "InternalIP" {
				internalIP = address.Address
				break
			}
		}
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v",
			node.GetName(), role, node.Status.NodeInfo.KubeletVersion, internalIP, "None", node.Status.NodeInfo.OSImage, node.Status.NodeInfo.KernelVersion,
			node.Status.NodeInfo.ContainerRuntimeVersion, time.Until(node.GetCreationTimestamp().Time).Abs()))
	}
	writer.Flush()
	return nil
}

// ShowPodInfo displays the pods of workload OCNE cluster as a formatted table.
func (c CAPITestImpl) ShowPodInfo(client *kubernetes.Clientset, clusterName string, log *zap.SugaredLogger) error {
	nsList, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get list of namespaces from cluster '%s'", clusterName))
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tNamespace\tStatus\tIP\tNode\tAge")
	//var dnsPod, ccmPod, calicokubePod *v1.Pod
	for _, ns := range nsList.Items {
		podList, err := client.CoreV1().Pods(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get list of pods from cluster '%s'", clusterName))
		}
		for _, pod := range podList.Items {
			podData, err := client.CoreV1().Pods(ns.Name).Get(context.TODO(), pod.Name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					log.Infof("No pods in namespace '%s'", ns.Name)
				} else {
					return errors.Wrap(err, fmt.Sprintf("failed to get pod '%s' from cluster '%s'", pod.Name, clusterName))
				}
			}

			fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v",
				podData.GetName(), podData.GetNamespace(), podData.Status.Phase, podData.Status.PodIP, podData.Spec.NodeName,
				time.Until(podData.GetCreationTimestamp().Time).Abs()))
		}
	}
	writer.Flush()
	return nil
}

// ShowEvents displays the events from a specific namespace
func (c CAPITestImpl) ShowEvents(namespace string, log *zap.SugaredLogger) error {
	log.Infof("Showing events for namespace '%s'", namespace)
	k8sclient, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	events, err := k8sclient.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Info("Failed to get events from namespace: %v", zap.Error(err))
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Namespace\tLast Seen\tType\tReason\tObject\tMessage")

	for _, event := range events.Items {
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v",
			event.Namespace, event.LastTimestamp, event.Type, event.Reason, fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			event.Message))

	}
	writer.Flush()
	return nil
}

// DisplayWorkloadClusterResources displays the pods of workload OCNE cluster as a formatted table.
func (c CAPITestImpl) DisplayWorkloadClusterResources(clusterName string, log *zap.SugaredLogger) error {
	client, err := c.GetCapiClusterK8sClient(clusterName, log)
	if err != nil {
		return errors.Wrap(err, "Failed to get k8s client for workload cluster")
	}

	log.Infof("----------- Node in workload cluster ---------------------")
	err = c.ShowNodeInfo(client, clusterName, log)
	if err != nil {
		return err
	}

	log.Infof("----------- Pods running on workload cluster ---------------------")
	return c.ShowPodInfo(client, clusterName, log)
}

/*
func deleteNamespace(namespace string, log *zap.SugaredLogger) error {
	log.Infof("deleting namespace '%s'", namespace)
	k8s, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	return k8s.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
}
*/

// UpdateOCINSG allows updating nsg based on source subnet cidr using the source subnet name
func (c CAPITestImpl) UpdateOCINSG(clusterName, nsgDisplayNameToUpdate, nsgDisplayNameInRule, info string, rule *SecurityRuleDetails, log *zap.SugaredLogger) error {
	log.Infof("Updating NSG rules for cluster '%s' and nsg '%s' for '%s'", clusterName, nsgDisplayNameToUpdate, info)
	oci, err := NewClient(GetOCIConfigurationProvider(log))
	if err != nil {
		log.Error("Unable to create OCI client %v", zap.Error(err))
		return err
	}

	vcnID, err := oci.GetVcnIDByName(context.TODO(), OCICompartmentID, clusterName, log)
	if err != nil {
		return err
	}

	nsgID, err := oci.GetNsgIDByName(context.TODO(), OCICompartmentID, vcnID, nsgDisplayNameToUpdate, log)
	if err != nil {
		return err
	}

	ruleCIDR, err := oci.GetSubnetCIDRByName(context.TODO(), OCICompartmentID, vcnID, nsgDisplayNameInRule, log)
	if err != nil {
		return err
	}
	rule.Source = ruleCIDR

	return oci.UpdateNSG(context.TODO(), nsgID, rule, log)
}

// UpdateOCINSGEW allows updating nsg based on vcn cidr
func (c CAPITestImpl) UpdateOCINSGEW(clusterName, nsgDisplayNameToUpdate, info string, rule *SecurityRuleDetails, log *zap.SugaredLogger) error {
	log.Infof("Updating NSG rules for cluster '%s' and nsg '%s' for '%s'", clusterName, nsgDisplayNameToUpdate, info)
	oci, err := NewClient(GetOCIConfigurationProvider(log))
	if err != nil {
		log.Error("Unable to create OCI client %v", zap.Error(err))
		return err
	}

	vcnID, err := oci.GetVcnIDByName(context.TODO(), OCICompartmentID, clusterName, log)
	if err != nil {
		return err
	}

	nsgID, err := oci.GetNsgIDByName(context.TODO(), OCICompartmentID, vcnID, nsgDisplayNameToUpdate, log)
	if err != nil {
		return err
	}

	ruleCIDR, err := oci.GetVcnCIDRByName(context.TODO(), OCICompartmentID, clusterName, log)
	if err != nil {
		return err
	}
	rule.Source = ruleCIDR

	return oci.UpdateNSG(context.TODO(), nsgID, rule, log)
}

func (c CAPITestImpl) GetCapiClusterDynamicClient(clusterName string, log *zap.SugaredLogger) (dynamic.Interface, error) {
	capiK8sConfig, err := c.GetCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return nil, err
	}
	tmpFile, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s-kubeconfig", clusterName))
	if err != nil {
		log.Errorf("Failed to create temporary file : %v", zap.Error(err))
		return nil, err
	}

	if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		log.Errorf("failed to write to destination file : %v", zap.Error(err))
		return nil, err
	}

	k8sRestConfig, err := k8sutil.GetKubeConfigGivenPathAndContext(tmpFile.Name(), fmt.Sprintf("%s-admin@%s", clusterName, clusterName))
	if err != nil {
		log.Errorf("failed to obtain k8s rest config : %v", zap.Error(err))
		return nil, err
	}

	dclient, err := dynamic.NewForConfig(k8sRestConfig)
	if err != nil {
		log.Errorf("unable to create dynamic client for workload cluster %v", zap.Error(err))
		return nil, err
	}
	return dclient, nil

}

func (c CAPITestImpl) GetVerrazzano(clusterName, namespace, vzinstallname string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	dclient, err := c.GetCapiClusterDynamicClient(clusterName, log)
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    "install.verrazzano.io",
		Version:  "v1beta1",
		Resource: "verrazzanos",
	}

	return dclient.Resource(gvr).Namespace(namespace).Get(context.TODO(), vzinstallname, metav1.GetOptions{})
}

func (c CAPITestImpl) EnsureVerrazzano(clusterName string, log *zap.SugaredLogger) error {

	vzFetched, err := c.GetVerrazzano(clusterName, "default", "workload-verrazzano", log)
	if err != nil {
		log.Errorf("unable to fetch vz resource from %s due to '%v'", clusterName, zap.Error(err))
		return err
	}
	var vz Verrazzano
	modBinaryData, err := json.Marshal(vzFetched)
	if err != nil {
		log.Error("json marshalling error ", zap.Error(err))
		return err
	}

	err = json.Unmarshal(modBinaryData, &vz)
	if err != nil {
		log.Error("json unmarshalling error ", zap.Error(err))
		return err
	}

	curState := "InstallStarted"
	for _, cond := range vz.Status.Conditions {
		if cond.Type == "InstallComplete" {
			curState = cond.Type
		}
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tAvailable\tStatus\tVersion")
	fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v",
		vz.Metadata.Name, vz.Status.Available, curState, vz.Status.Version))
	writer.Flush()

	if curState == "InstallComplete" {
		return nil
	}
	return fmt.Errorf("All components are not ready: Current State = %v", curState)
}

func (c CAPITestImpl) DebugSVCOutput(clusterName string, log *zap.SugaredLogger) error {

	capiK8sConfig, err := c.GetCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		log.Error("Failed to create temporary file ", zap.Error(err))
		return err
	}

	if err = os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		log.Error("failed to write to destination file ", zap.Error(err))
		return err
	}

	var cmdArgs []string
	var bcmd helpers.BashCommand
	dockerSecretCommand := fmt.Sprintf("kubectl --kubeconfig %s get svc -A", tmpFile.Name())
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	debugCmdResponse := helpers.Runner(&bcmd, log)
	if debugCmdResponse.CommandError != nil {
		return debugCmdResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s get pod -A", tmpFile.Name())
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	debugCmdResponse = helpers.Runner(&bcmd, log)
	if debugCmdResponse.CommandError != nil {
		return debugCmdResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s get vz", tmpFile.Name())
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	debugCmdResponse = helpers.Runner(&bcmd, log)
	if debugCmdResponse.CommandError != nil {
		return debugCmdResponse.CommandError
	}

	return nil
}

func (c CAPITestImpl) CreateImagePullSecrets(clusterName string, log *zap.SugaredLogger) error {
	log.Infof("Creating image pull secrets on workload cluster ...")

	capiK8sConfig, err := c.GetCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		log.Error("Failed to create temporary file ", zap.Error(err))
		return err
	}

	if err = os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		log.Error("failed to write to destination file ", zap.Error(err))
		return err
	}

	var cmdArgs []string
	var bcmd helpers.BashCommand
	dockerSecretCommand := fmt.Sprintf("kubectl --kubeconfig %s create secret docker-registry %s --docker-server=%s --docker-username=%s --docker-password=%s", tmpFile.Name(), ImagePullSecret, DockerRepo, DockerCredsUser, DockerCredsPassword)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse := helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s create secret docker-registry %s --docker-server=%s --docker-username=%s --docker-password=%s -n verrazzano-install", tmpFile.Name(), ImagePullSecret, DockerRepo, DockerCredsUser, DockerCredsPassword)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse = helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s create secret docker-registry github-packages --docker-server=%s --docker-username=%s --docker-password=%s", tmpFile.Name(), DockerRepo, DockerCredsUser, DockerCredsPassword)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse = helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s create secret docker-registry ocr --docker-server=%s --docker-username=%s --docker-password=%s", tmpFile.Name(), DockerRepo, DockerCredsUser, DockerCredsPassword)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse = helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}

	return nil

}

func (c CAPITestImpl) CheckOCNEControlPlaneStatus(clusterName, expectedStatusType, expectedStatus, expectedReason string, log *zap.SugaredLogger) bool {
	ocneCP, err := c.GetOCNEControlPlane(OCNENamespace, log)
	if err != nil {
		log.Error("unable to fetch OCNE control plane : ", zap.Error(err))
		return false
	}

	for _, cond := range ocneCP.Status.Conditions {
		if cond.Type == expectedStatusType {
			if cond.Status == expectedStatus {
				switch cond.Status {
				case "False":
					if cond.Reason == expectedReason {
						// all matched
						return true
					}
				case "True":
					// reason is irrelevant
					return true
				}
			}
			log.Errorf("HAVE status type = %s, value = %s, reason = %s. WANT status type = %s, value= %s.", cond.Type, cond.Status, cond.Reason, expectedStatusType, expectedStatus)
			return false
		}
	}

	log.Errorf("OCNE controlplane check failure. All conditions %+v", ocneCP.Status.Conditions)
	return false
}
