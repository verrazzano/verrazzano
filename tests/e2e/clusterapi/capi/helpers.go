package capi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"os"
	"path/filepath"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	CapiDefaultNameSpace = "default"
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
		Kubeconfig: clusterapi.Kubeconfig{kubeconfigPath, ""},
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
	time.Sleep(10 * time.Second)
	return nil
}

// getUnstructuredData common utility to fetch unstructured data
func getUnstructuredData(group, version, resource, resourceName, nameSpaceName, component string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
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
		log.Infof("Fetching '%s' '%s' '%s' in namespace '%s'", component, resource, resourceName, nameSpaceName)
		dataFetched, err = dclient.Resource(gvr).Namespace(nameSpaceName).Get(context.TODO(), resourceName, metav1.GetOptions{})
	} else {
		log.Infof("Fetching '%s' '%s' '%s'", component, resource, resourceName)
		dataFetched, err = dclient.Resource(gvr).Get(context.TODO(), resourceName, metav1.GetOptions{})
	}
	if err != nil {
		log.Errorf("Unable to fetch %s %s %s due to '%v'", component, resource, resourceName, zap.Error(err))
		return nil, err
	}
	return dataFetched, nil
}

// getUnstructuredData common utility to fetch list of unstructured data
func getUnstructuredDataList(group, version, resource, nameSpaceName, component string, log *zap.SugaredLogger) (*unstructured.UnstructuredList, error) {
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

	log.Infof("Fetching %s %s", component, resource)
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	dataFetched, err := dclient.Resource(gvr).Namespace(nameSpaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Unable to fetch %s %s due to '%v'", component, resource, zap.Error(err))
		return nil, err
	}
	return dataFetched, nil
}

func getCluster(namespace, clusterName string, log *zap.SugaredLogger) (*Cluster, error) {
	var capiCluster Cluster
	clusterFetched, err := getUnstructuredData("cluster.x-k8s.io", "v1beta1", "clusters", clusterName, namespace, "capi-cluster", log)
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
	ocnecpFetched, err := getUnstructuredData("controlplane.cluster.x-k8s.io", "v1alpha1", "ocnecontrolplanes", controlPlaneName, namespace, "ocne-control-plane", log)
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
	machinesFetched, err := getUnstructuredDataList("cluster.x-k8s.io", "v1beta1", "machines", namespace, "capi-machines", log)
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
	klusterData, err := getCluster(CapiDefaultNameSpace, clusterName, log)
	if err != nil {
		return err
	}
	if klusterData == nil {
		return nil
	}
	return fmt.Errorf("Cluster data not empty. Still present")
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
	log.Infof("Control plane details:")
	log.Infof("Name:%v, Cluster:%v, Initialized:%v, Replicas:%v, Updated:%v, Unavaliable:%v, Ready:%v",
		ocneCP.Metadata.Name, ocneCP.Metadata.Labels.ClusterXK8SIoClusterName, ocneCP.Status.Initialized, ocneCP.Status.Replicas,
		ocneCP.Status.UpdatedReplicas, ocneCP.Status.UnavailableReplicas, ocneCP.Status.ReadyReplicas)

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tCluster\tInitialized\tReplicas\tUpdated\tUnavaliable\tReady")
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
	return fmt.Errorf("Cluster '%s' phase is => '%s'", clusterName, klusterData.Status.Phase)
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

func ensureCapiAccess(clusterName string, log *zap.SugaredLogger) error {

	capiK8sConfig, err := getCapiClusterKubeconfig(clusterName, log)
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		return errors.Wrap(err, "Failed to create temporary file")
	}
	log.Info(tmpFile.Name())

	if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		return errors.Wrap(err, "failed to write to destination file")
	}

	k8sRestConfig, err := k8sutil.GetKubeConfigGivenPathAndContext(tmpFile.Name(), fmt.Sprintf("%s-admin@%s", clusterName))
	if err != nil {
		return errors.Wrap(err, "Failed to get k8s rest config")
	}

	client, err := k8sutil.GetKubernetesClientsetWithConfig(k8sRestConfig)
	if err != nil {
		return errors.Wrap(err, "Failed to get k8s client")
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tVersion")
	nodeList, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	for _, node := range nodeList.Items {
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v",
			node.GetName(), node.Status.NodeInfo.KernelVersion))
	}
	writer.Flush()
	return nil
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
