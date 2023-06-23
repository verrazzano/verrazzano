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
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	CapiDefaultNameSpace = "default"
	nodeLabel            = "node-role.kubernetes.io/node"
	controlPlaneLabel    = "node-role.kubernetes.io/control-plane"
)

var capiInitFunc = clusterapi.New

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

func clusterTemplateGenerate(clusterName, templatePath string, log *zap.SugaredLogger) error {
	log.Infof("Generate called for clustername '%s'...", clusterName)
	capiClient, err := capiInitFunc("")
	if err != nil {
		return err
	}
	log.Info("Fetching kubeconfig ...")
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig url due to %v", zap.Error(err))
		return err
	}

	log.Info("Start templating ...")

	templateOptions := clusterapi.GetClusterTemplateOptions{
		Kubeconfig: clusterapi.Kubeconfig{
			Path:    kubeconfigPath,
			Context: ""},
		URLSource: &clusterapi.URLSourceOptions{
			URL: templatePath,
		},
		ClusterName: clusterName,
	}

	template, err := capiClient.GetClusterTemplate(templateOptions)
	if err != nil {
		log.Errorf("GetClusterTemplate error = %v", zap.Error(err))
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

func triggerCapiClusterCreation(templatePath string, log *zap.SugaredLogger) error {
	clusterTemplateData, err := os.ReadFile(templatePath)
	if err != nil {
		return nil
	}
	err = resource.CreateOrUpdateResourceFromBytes(clusterTemplateData, log)
	//err = common.DynamicSSA(context.TODO(), string(clusterTemplateData), log)
	if err != nil {
		log.Errorf("Error creating cluster from template ", zap.Error(err))
		return err
	}
	log.Infof("Wait for 10 seconds before verification")
	time.Sleep(30 * time.Second)
	return nil
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
	fmt.Fprintln(writer, "Name\tCluster\tNodename\tProviderID\tPhase")

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
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v",
			machine.Metadata.Name, machine.Metadata.Labels.ClusterXK8SIoClusterName, machine.Status.NodeRef.Name,
			machine.Spec.ProviderID, machine.Status.Phase))

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
	return fmt.Errorf("All nodes are not in 'Running' state")
}

func monitorCapiClusterDeletion(clusterName string, log *zap.SugaredLogger) error {
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

	kluster, err := dclient.Resource(gvr).Namespace(CapiDefaultNameSpace).Get(context.TODO(), clusterName, metav1.GetOptions{})
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

	return fmt.Errorf("Cluster '%s' is in '%s' state.", clusterName, capiCluster.Status.Phase)
}

func monitorCapiClusterCreation(clusterName string, log *zap.SugaredLogger) error {
	klusterData, err := getCluster(CapiDefaultNameSpace, clusterName, log)
	if err != nil {
		return err
	}

	controlPlaneName := fmt.Sprintf("%s-control-plane", clusterName)
	ocneCP, err := getOCNEControlPlane(CapiDefaultNameSpace, controlPlaneName, log)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tCluster\tInitialized\tReplicas\tUpdated\tUnavailable\tReady")
	fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v",
		ocneCP.Metadata.Name, ocneCP.Metadata.Labels.ClusterXK8SIoClusterName, ocneCP.Status.Initialized, ocneCP.Status.Replicas,
		ocneCP.Status.UpdatedReplicas, ocneCP.Status.UnavailableReplicas, ocneCP.Status.ReadyReplicas))
	writer.Flush()
	err = ensureMachinesAreProvisioned(CapiDefaultNameSpace, clusterName, log)
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

func getCapiClusterKubeconfig(clusterName string, log *zap.SugaredLogger) ([]byte, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return nil, err
	}

	secret, err := clientset.CoreV1().Secrets(CapiDefaultNameSpace).Get(context.TODO(), fmt.Sprintf("%s-kubeconfig", clusterName), metav1.GetOptions{})
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
	log.Info(tmpFile.Name())

	if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		return nil, errors.Wrap(err, "failed to write to destination file")
	}

	k8sRestConfig, err := k8sutil.GetKubeConfigGivenPathAndContext(tmpFile.Name(), fmt.Sprintf("%s-admin@%s", clusterName, clusterName))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get k8s rest config")
	}

	return k8sutil.GetKubernetesClientsetWithConfig(k8sRestConfig)
}

func getCapiClusterK8sConfig(clusterName string, log *zap.SugaredLogger) (config *rest.Config, err error) {
	capiK8sConfig, err := getCapiClusterKubeconfig(clusterName, log)
	if err != nil {
		return nil, err
	}
	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create temporary file")
	}
	log.Info(tmpFile.Name())

	if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		return nil, errors.Wrap(err, "failed to write to destination file")
	}

	return k8sutil.GetKubeConfigGivenPathAndContext(tmpFile.Name(), fmt.Sprintf("%s-admin@%s", clusterName, clusterName))
}

func ensureCapiAccess(clusterName string, log *zap.SugaredLogger) error {
	client, err := getCapiClusterK8sClient(clusterName, log)
	if err != nil {
		return errors.Wrap(err, "Failed to get k8s client for workload cluster")
	}

	var cmdArgs []string
	var bcmd helpers.BashCommand
	ocicmd := fmt.Sprintf("oci network vcn list --compartment-id %s --display-name %s | jq -r '.data[0].id'", OCICompartmentID, ClusterName)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", ocicmd)
	bcmd.CommandArgs = cmdArgs
	vcndata := helpers.Runner(&bcmd, log)
	if vcndata.CommandError != nil {
		return vcndata.CommandError
	}

	cmdArgs = []string{}
	ocicmd = fmt.Sprintf("oci network subnet list --compartment-id %s --vcn-id %s --display-name service-lb | jq -r '.data[0].id'", OCICompartmentID, vcndata.StandardOut.String())
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", ocicmd)
	bcmd.CommandArgs = cmdArgs
	subnetData := helpers.Runner(&bcmd, log)
	if subnetData.CommandError != nil {
		return subnetData.CommandError
	}

	OCIVcnID = vcndata.StandardOut.String()
	OCISubnetID = subnetData.StandardOut.String()

	log.Infof("+++ VCN ID = %s", OCIVcnID)
	log.Infof("+++ Subnet ID = %s", OCISubnetID)

	log.Infof("----------- Node in workload cluster ---------------------")
	err = showNodeInfo(client, clusterName, log)
	if err != nil {
		return err
	}

	log.Infof("----------- Pods running on workload cluster ---------------------")
	return showPodInfo(client, clusterName, log)

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

func showNodeInfo(client *kubernetes.Clientset, clustername string, log *zap.SugaredLogger) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tRole\tVersion\tInternalIP\tExternalIP\tOSImage\tKernelVersion\tContainerRuntime")
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
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v",
			node.GetName(), role, node.Status.NodeInfo.KubeletVersion, internalIP, "None", node.Status.NodeInfo.OSImage, node.Status.NodeInfo.KernelVersion,
			node.Status.NodeInfo.ContainerRuntimeVersion))
	}
	writer.Flush()
	return nil
}

func showPodInfo(client *kubernetes.Clientset, clustername string, log *zap.SugaredLogger) error {
	nsList, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get list of namespaces from cluster '%s'", clustername))
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tNamespace\tStatus\tIP\tNode")
	for _, ns := range nsList.Items {
		podList, err := client.CoreV1().Pods(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get list of pods from cluster '%s'", clustername))
		}
		for _, pod := range podList.Items {
			podData, err := client.CoreV1().Pods(ns.Name).Get(context.TODO(), pod.Name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					log.Infof("No pods in namespace '%s'", ns.Name)
				} else {
					return errors.Wrap(err, fmt.Sprintf("failed to get pod '%s' from cluster '%s'", pod.Name, clustername))
				}
			}
			fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v",
				podData.GetName(), podData.GetNamespace(), podData.Status.Phase, podData.Status.PodIP, podData.Spec.NodeName))

		}
	}
	writer.Flush()
	return nil
}

func deployClusterResourceSets(clustername, templateName string, log *zap.SugaredLogger) error {
	err := clusterTemplateGenerate(clustername, templateName, log)
	if err != nil {
		return errors.Wrap(err, "unable to generate template for clusterresourcesets")
	}

	clusterTemplateData, err := os.ReadFile(ClusterTemplateGeneratedFilePath)
	if err != nil {
		return errors.Wrap(err, "unable to get read  file")
	}

	config, err := getCapiClusterK8sConfig(clustername, log)
	if err != nil {
		return errors.Wrap(err, "unable to get kubeconfig for workload cluster")
	}
	return resource.CreateOrUpdateResourceFromBytesUsingConfig(clusterTemplateData, config)
}
