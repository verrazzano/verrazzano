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

func GetCluster(namespace, clusterName string, log *zap.SugaredLogger) (*Cluster, error) {
	clusterFetched, err := getUnstructuredData("cluster.x-k8s.io", "v1beta1", "clusters", clusterName, namespace, "capi-cluster", log)
	if err != nil {
		log.Errorf("Unable to fetch CAPI cluster '%s' due to '%v'", clusterName, zap.Error(err))
		return nil, err
	}

	if clusterFetched == nil {
		log.Infof("No CAPI clusters with name '%s' in namespace '%s' was detected", clusterName, namespace)
	}

	var capiCluster Cluster
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

func showCapiCluster(clusterName string, log *zap.SugaredLogger) error {
	log.Info("Start templating ...")
	klusterData, err := GetCluster("default", clusterName, log)
	if err != nil {

	}

	// OCNE cluster is ready when both control plane and worker nodes are up
	if klusterData.Status.ControlPlaneReady && klusterData.Status.InfrastructureReady {
		return nil
	}
	return
}
