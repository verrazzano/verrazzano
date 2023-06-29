// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
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
)

var capiInitFunc = clusterapi.New

// printYamlOutput is used to print yaml templates to stdout or a file
func printYamlOutput(printer clusterapi.YamlPrinter, outputFile string) error {
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

// clusterTemplateGenerate used for cluster template generation
func clusterTemplateGenerate(clusterName, templatePath string, log *zap.SugaredLogger) error {
	log.Infof("Generate called for clustername '%s'...", clusterName)
	capiClient, err := capiInitFunc("")
	if err != nil {
		return err
	}
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig url due to %v", zap.Error(err))
		return err
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
		return err
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		return fmt.Errorf("Failed to create temporary file: %v", err)
	}

	log.Infof("Temp file name = %v", tmpFile.Name())
	ClusterTemplateGeneratedFilePath = tmpFile.Name()

	return printYamlOutput(template, tmpFile.Name())
}

// getUnstructuredData common utility to fetch unstructured data
func getUnstructuredData(group, version, resource, resourceName, nameSpaceName string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
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

// getUnstructuredData common utility to fetch list of unstructured data
func getUnstructuredDataList(group, version, resource, nameSpaceName string, log *zap.SugaredLogger) (*unstructured.UnstructuredList, error) {
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

func getCluster(namespace, clusterName string, log *zap.SugaredLogger) (*Cluster, error) {
	var capiCluster Cluster
	clusterFetched, err := getUnstructuredData("cluster.x-k8s.io", "v1beta1", "clusters", clusterName, namespace, log)
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

func getOCNEControlPlane(namespace, controlPlaneName string, log *zap.SugaredLogger) (*OCNEControlPlane, error) {
	ocnecpFetched, err := getUnstructuredData("controlplane.cluster.x-k8s.io", "v1alpha1", "ocnecontrolplanes", controlPlaneName, namespace, log)
	if err != nil {
		log.Errorf("Unable to fetch OCNE control plane '%s' due to '%v'", controlPlaneName, zap.Error(err))
		return nil, err
	}

	if ocnecpFetched == nil {
		log.Infof("No OCNE control plane with name '%s' in namespace '%s' was detected", controlPlaneName, namespace)
	}

	var ocneControlPlane OCNEControlPlane
	bdata, err := json.Marshal(ocnecpFetched)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(bdata, &ocneControlPlane)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return nil, err
	}

	return &ocneControlPlane, nil
}

func checkAll(data []bool) bool {
	for _, item := range data {
		// return false if any item is false
		if !item {
			return false
		}
	}
	return true
}

func getCapiClusterKubeconfig(clusterName string, log *zap.SugaredLogger) ([]byte, error) {
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

func getCapiClusterK8sClient(clusterName string, log *zap.SugaredLogger) (client *kubernetes.Clientset, err error) {
	capiK8sConfig, err := getCapiClusterKubeconfig(clusterName, log)
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

func TriggerCapiClusterCreation(clusterName, templateName string, log *zap.SugaredLogger) error {
	err := clusterTemplateGenerate(clusterName, templateName, log)
	if err != nil {
		log.Errorf("unable to generate template for cluster : %v", zap.Error(err))
		return err
	}

	clusterTemplateData, err := os.ReadFile(ClusterTemplateGeneratedFilePath)
	if err != nil {
		return nil
	}
	err = resource.CreateOrUpdateResourceFromBytes(clusterTemplateData, log)
	if err != nil {
		log.Errorf("Error creating cluster from template ", zap.Error(err))
		return err
	}
	log.Infof("Wait for 10 seconds before verification")
	time.Sleep(30 * time.Second)
	return nil
}

func DeployClusterResourceSets(clusterName, templateName string, log *zap.SugaredLogger) error {
	log.Info("Preparing to deploy Clusterresourcesets...")
	//var cmdArgs []string
	//var bcmd helpers.BashCommand
	//ocicmd := fmt.Sprintf("oci network vcn list --compartment-id %s --display-name %s | jq -r '.data[0].id'", OCICompartmentID, clusterName)
	//cmdArgs = append(cmdArgs, "/bin/bash", "-c", ocicmd)
	//bcmd.CommandArgs = cmdArgs
	//vcndata := helpers.Runner(&bcmd, log)
	//if vcndata.CommandError != nil {
	//	return vcndata.CommandError
	//}
	//OCIVcnID = strings.Trim(vcndata.StandardOut.String(), "\n")
	//
	//cmdArgs = []string{}
	//ocicmd = fmt.Sprintf("oci network subnet list --compartment-id %s --vcn-id %s --display-name service-lb | jq -r '.data[0].id'", OCICompartmentID, OCIVcnID)
	//cmdArgs = append(cmdArgs, "/bin/bash", "-c", ocicmd)
	//bcmd.CommandArgs = cmdArgs
	//subnetData := helpers.Runner(&bcmd, log)
	//if subnetData.CommandError != nil {
	//	return subnetData.CommandError
	//}
	//
	//OCISubnetID = strings.Trim(subnetData.StandardOut.String(), "\n")

	oci, err := NewClient(GetOCIConfigurationProvider(log))
	if err != nil {
		log.Error("Unable to create OCI client %v", zap.Error(err))
		return err
	}

	OCIVcnID, err = oci.GetVcnIDByNane(context.TODO(), OCICompartmentID, clusterName)
	if err != nil {
		return err
	}

	OCISubnetID, err = oci.GetSubnetIDByNane(context.TODO(), OCICompartmentID, OCIVcnID, "service-lb")
	if err != nil {
		return err
	}

	os.Setenv(OCIVCNKey, OCIVcnID)
	os.Setenv(OCISubnetKey, OCISubnetID)

	err = clusterTemplateGenerate(clusterName, templateName, log)
	if err != nil {
		log.Errorf("unable to generate template for clusterresourcesets : %v", zap.Error(err))
		return err
	}

	clusterTemplateData, err := os.ReadFile(ClusterTemplateGeneratedFilePath)
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
	return nil
}

func ensureMachinesAreProvisioned(namespace, clusterName string, log *zap.SugaredLogger) error {
	machinesFetched, err := getUnstructuredDataList("cluster.x-k8s.io", "v1beta1", "machines", namespace, log)
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
		return nil
	}
	return fmt.Errorf("All machines are not in 'Running' state")
}

func MonitorCapiClusterDeletion(clusterName string, log *zap.SugaredLogger) error {
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

func MonitorCapiClusterCreation(clusterName string, log *zap.SugaredLogger) error {
	klusterData, err := getCluster(OCNENamespace, clusterName, log)
	if err != nil {
		return err
	}

	controlPlaneName := fmt.Sprintf("%s-control-plane", clusterName)
	ocneCP, err := getOCNEControlPlane(OCNENamespace, controlPlaneName, log)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tCluster\tInitialized\tReplicas\tUpdated\tUnavailable\tReady\tAge")
	fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v",
		ocneCP.Metadata.Name, ocneCP.Metadata.Labels.ClusterXK8SIoClusterName, ocneCP.Status.Initialized, ocneCP.Status.Replicas,
		ocneCP.Status.UpdatedReplicas, ocneCP.Status.UnavailableReplicas, ocneCP.Status.ReadyReplicas, time.Until(ocneCP.Metadata.CreationTimestamp).Abs()))
	writer.Flush()
	err = ensureMachinesAreProvisioned(OCNENamespace, clusterName, log)
	if err != nil {
		return err
	}

	// OCNE cluster is ready when both control plane and worker nodes are up
	if klusterData.Status.ControlPlaneReady && klusterData.Status.InfrastructureReady {
		log.Infof("Cluster '%s' phase is => '%s'. All machines are also in '%s' state.", clusterName, klusterData.Status.Phase, klusterData.Status.Phase)
		return nil
	}
	return fmt.Errorf("cluster '%s' phase is => '%s'", clusterName, klusterData.Status.Phase)
}

func triggerCapiClusterDeletion(clusterName, nameSpaceName string, log *zap.SugaredLogger) error {
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

/*
func showModuleInfo(clusterName string, log *zap.SugaredLogger) error {

	k8sconfig, err := getCapiClusterK8sConfig(clusterName, log)
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return err
	}

	dclient, err := dynamic.NewForConfig(k8sconfig)
	if err != nil {
		log.Errorf("unable to create dynamic client for workload cluster %v", zap.Error(err))
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "platform.verrazzano.io",
		Version:  "v1alpha1",
		Resource: "module",
	}

	calicoModuleFetched, err := dclient.Resource(gvr).Get(context.TODO(), "calico", metav1.GetOptions{})
	if err != nil {
		log.Errorf("unable to fetch moduledata from %s due to '%v' for module calico", clusterName, zap.Error(err))
		return err
	}

	var calico Module
	modBinaryData, err := json.Marshal(calicoModuleFetched)
	if err != nil {
		log.Error("json marshalling error ", zap.Error(err))
		return err
	}

	err = json.Unmarshal(modBinaryData, &calico)
	if err != nil {
		log.Error("json unmarshalling error ", zap.Error(err))
		return err
	}

	log.Infof("Values for module '%s' => %+v", calico.Metadata.Name, calico.Spec.Values)

	ccmModuleFetched, err := dclient.Resource(gvr).Get(context.TODO(), "oci-ccm", metav1.GetOptions{})
	if err != nil {
		log.Errorf("unable to fetch moduledata from %s due to '%v' for module ccm ", clusterName, zap.Error(err))
		return err
	}

	var ccm Module
	ccmBinaryData, err := json.Marshal(ccmModuleFetched)
	if err != nil {
		log.Error("json marshalling error ", zap.Error(err))
		return err
	}

	err = json.Unmarshal(ccmBinaryData, &ccm)
	if err != nil {
		log.Error("json unmarshalling error ", zap.Error(err))
		return err
	}

	log.Infof("Values for module '%s' => %+v", ccm.Metadata.Name, ccm.Spec.Values)
	return nil
}
*/

func showNodeInfo(client *kubernetes.Clientset, clustername string, log *zap.SugaredLogger) error {
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

func showPodInfo(client *kubernetes.Clientset, clusterName string, log *zap.SugaredLogger) error {
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

/*
func showSecretsInfo(client *kubernetes.Clientset, clusterName string, log *zap.SugaredLogger) error {
	nsList, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get list of namespaces from cluster '%s'", clusterName))
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Namespace\tName\tType\tAge")
	for _, ns := range nsList.Items {
		secretList, err := client.CoreV1().Secrets(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get list of secrets from cluster '%s'", clusterName))
		}
		for _, secret := range secretList.Items {
			secretData, err := client.CoreV1().Secrets(ns.Name).Get(context.TODO(), secret.Name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					log.Infof("No secrets in namespace '%s'", ns.Name)
				} else {
					return errors.Wrap(err, fmt.Sprintf("failed to get secret '%s' from cluster '%s'", secretData.Name, clusterName))
				}
			}

			fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v",
				secretData.GetNamespace(), secretData.GetName(), secretData.Type, time.Until(secretData.GetCreationTimestamp().Time)))

		}
	}
	writer.Flush()
	return nil
}

func showConfigMapsInfo(client *kubernetes.Clientset, clusterName string, log *zap.SugaredLogger) error {
	nsList, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get list of namespaces from cluster '%s'", clusterName))
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Namespace\tName\tAge")
	for _, ns := range nsList.Items {
		configmapList, err := client.CoreV1().ConfigMaps(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get list of secrets from cluster '%s'", clusterName))
		}
		for _, configmap := range configmapList.Items {
			configmapData, err := client.CoreV1().ConfigMaps(ns.Name).Get(context.TODO(), configmap.Name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					log.Infof("No configmaps in namespace '%s'", ns.Name)
				} else {
					return errors.Wrap(err, fmt.Sprintf("failed to get configmap '%s' from cluster '%s'", configmapData.Name, clusterName))
				}
			}

			fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v",
				configmapData.GetNamespace(), configmapData.GetName(), time.Until(configmapData.GetCreationTimestamp().Time)))

		}
	}
	writer.Flush()
	return nil
}
*/

func displayWorkloadClusterResources(clusterName string, log *zap.SugaredLogger) error {
	client, err := getCapiClusterK8sClient(clusterName, log)
	if err != nil {
		return errors.Wrap(err, "Failed to get k8s client for workload cluster")
	}

	log.Infof("----------- Node in workload cluster ---------------------")
	err = showNodeInfo(client, clusterName, log)
	if err != nil {
		return err
	}

	log.Infof("----------- Pods running on workload cluster ---------------------")
	return showPodInfo(client, clusterName, log)
}

func processOCIPrivateKeysBase64(file, key string, log *zap.SugaredLogger) error {
	_, err := os.Stat(file)
	if err != nil {
		log.Errorf("file '%s' not found", file)
		return err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		log.Errorf("failed reading file contents: %v", err)
		return err
	}

	return os.Setenv(key, base64.StdEncoding.EncodeToString(data))
}

func processOCISSHKeys(file, key string, log *zap.SugaredLogger) error {
	_, err := os.Stat(file)
	if err != nil {
		log.Errorf("file '%s' not found", file)
		return err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		log.Errorf("failed reading file contents: %v", err)
		return err
	}

	return os.Setenv(key, string(data))
}

func processOCIPrivateKeysSingleLine(file, key string, log *zap.SugaredLogger) error {
	_, err := os.Stat(file)
	if err != nil {
		log.Errorf("file '%s' not found", file)
		return err
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), "testkey")
	if err != nil {
		log.Errorf("Failed to create temporary file : %v", zap.Error(err))
		return err
	}

	var cmdArgs []string
	var bcmd helpers.BashCommand
	ocicmd := "awk 'NF {sub(/\\r/, \"\"); printf \"%s\\\\n\",$0;}' " + file + "> " + tmpFile.Name()
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", ocicmd)
	bcmd.CommandArgs = cmdArgs
	keydata := helpers.Runner(&bcmd, log)
	if keydata.CommandError != nil {
		return keydata.CommandError
	}

	bdata, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		log.Errorf("failed reading file contents: %v", err)
		return err
	}

	return os.Setenv(key, string(bdata))
}

func createNamespace(namespace string, log *zap.SugaredLogger) error {
	log.Infof("creating namespace '%s'", namespace)
	k8s, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}

	nsObj := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = k8s.CoreV1().Namespaces().Create(context.TODO(), nsObj, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("failed to create namespace %v", zap.Error(err))
		return err
	}
	return nil

}

func deleteNamespace(namespace string, log *zap.SugaredLogger) error {
	log.Infof("deleting namespace '%s'", namespace)
	k8s, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	return k8s.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
}

func updateOCINSG(clusterName, nsgDisplayName string, rule *SecurityRuleDetails, log *zap.SugaredLogger) error {
	log.Infof("Updating NSG rules for cluster '%s' for nsg '%s'", clusterName, nsgDisplayName)
	//var cmdArgs []string
	//var bcmd helpers.BashCommand
	//ocicmd := fmt.Sprintf("oci network nsg list --compartment-id %s --vcn-id %s --display-name %s | jq -r '.data[0].id'", OCICompartmentID, OCIVcnID, nsgDisplayName)
	//cmdArgs = append(cmdArgs, "/bin/bash", "-c", ocicmd)
	//bcmd.CommandArgs = cmdArgs
	//nsgData := helpers.Runner(&bcmd, log)
	//if nsgData.CommandError != nil {
	//	return nsgData.CommandError
	//}
	//
	//nsgOCID := strings.Trim(nsgData.StandardOut.String(), "\n")
	//
	//cmdArgs = []string{}
	//ocicmd = fmt.Sprintf("oci network nsg rules add --nsg-id %s --from-json file://%s", nsgOCID, nsgRuleTemplatePath)
	//cmdArgs = append(cmdArgs, "/bin/bash", "-c", ocicmd)
	//bcmd.CommandArgs = cmdArgs
	//_ = helpers.Runner(&bcmd, log)
	//if nsgData.CommandError != nil {
	//	log.Errorf("failure to update nsg %v", zap.Error(nsgData.CommandError))
	//	return nsgData.CommandError
	//}

	oci, err := NewClient(GetOCIConfigurationProvider(log))
	if err != nil {
		log.Error("Unable to create OCI client %v", zap.Error(err))
		return err
	}

	nsgID, err := oci.GetNsgIDByNane(context.TODO(), OCICompartmentID, OCIVcnID, nsgDisplayName)
	if err != nil {
		return err
	}

	return oci.UpdateNSG(context.TODO(), nsgID, rule)
}

func setImageID(key string, log *zap.SugaredLogger) error {
	oci, err := NewClient(GetOCIConfigurationProvider(log))
	if err != nil {
		log.Error("Unable to create OCI client %v", zap.Error(err))
		return err
	}
	id, err := oci.GetImageIdByName(context.TODO(), OCICompartmentID, "Oracle-Linux-8.7-2023", "Oracle Linux", "8")
	if err != nil {
		log.Error("Unable to fetch image id %v", zap.Error(err))
		return err
	}
	return os.Setenv(key, id)
}
