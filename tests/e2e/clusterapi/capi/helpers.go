package capi

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	capicomponent "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusterapi"
	"go.uber.org/zap"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"path/filepath"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"strings"
	"text/template"
)

const (
	clusterctlYamlTemplate = `
images:
  cluster-api:
    repository: {{.APIRepository}}
    tag: {{.APITag}}

  infrastructure-oci:
    repository: {{.OCIRepository}}
    tag: {{.OCITag}}

  bootstrap-ocne:
    repository: {{.OCNEBootstrapRepository}}
    tag: {{.OCNEBootstrapTag}}
  control-plane-ocne:
    repository: {{.OCNEControlPlaneRepository}}
    tag: {{.OCNEControlPlaneTag}}

providers:
  - name: "cluster-api"
    url: "/verrazzano/capi/cluster-api/{{.APIVersion}}/core-components.yaml"
    type: "CoreProvider"
  - name: "oci"
    url: "/verrazzano/capi/infrastructure-oci/{{.OCIVersion}}/infrastructure-components.yaml"
    type: "InfrastructureProvider"
  - name: "ocne"
    url: "/verrazzano/capi/bootstrap-ocne/{{.OCNEBootstrapVersion}}/bootstrap-components.yaml"
    type: "BootstrapProvider"
  - name: "ocne"
    url: "/verrazzano/capi/control-plane-ocne/{{.OCNEControlPlaneVersion}}/control-plane-components.yaml"
    type: "ControlPlaneProvider"
`
	defaultClusterAPIDir = "$HOME/.cluster-api"
)

func getImageOverride(bomFile bom.Bom, component string, imageName string) (image *capicomponent.ImageConfig, err error) {
	version, err := bomFile.GetComponentVersion(component)
	if err != nil {
		return nil, err
	}

	images, err := bomFile.GetImageNameList(component)
	if err != nil {
		return nil, err
	}

	var repository string
	var tag string

	for _, image := range images {
		if len(imageName) == 0 || strings.Contains(image, imageName) {
			imageSplit := strings.Split(image, ":")
			tag = imageSplit[1]
			index := strings.LastIndex(imageSplit[0], "/")
			repository = imageSplit[0][:index]
			break
		}
	}

	if len(repository) == 0 || len(tag) == 0 {
		return nil, fmt.Errorf("Failed to find image override for %s/%s", component, imageName)
	}

	return &capicomponent.ImageConfig{Version: version, Repository: repository, Tag: tag}, nil
}

// getImageOverrides returns the CAPI provider image overrides and versions from the Verrazzano bom
func getImageOverrides() (*TemplateInput, error) {

	bomFile, err := bom.NewBom("../../../../platform-operator/verrazzano-bom.json")
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Failed to get the BOM file for the capi image overrides: %v", zap.Error(err)))
	}

	templateInput := &TemplateInput{}
	imageConfig, err := getImageOverride(bomFile, "capi-cluster-api", "")
	if err != nil {
		return nil, err
	}
	templateInput.APIVersion = imageConfig.Version
	templateInput.APIRepository = imageConfig.Repository
	templateInput.APITag = imageConfig.Tag

	imageConfig, err = getImageOverride(bomFile, "capi-oci", "")
	if err != nil {
		return nil, err
	}
	templateInput.OCIVersion = imageConfig.Version
	templateInput.OCIRepository = imageConfig.Repository
	templateInput.OCITag = imageConfig.Tag

	imageConfig, err = getImageOverride(bomFile, "capi-ocne", "cluster-api-ocne-bootstrap-controller")
	if err != nil {
		return nil, err
	}
	templateInput.OCNEBootstrapVersion = imageConfig.Version
	templateInput.OCNEBootstrapRepository = imageConfig.Repository
	templateInput.OCNEBootstrapTag = imageConfig.Tag

	imageConfig, err = getImageOverride(bomFile, "capi-ocne", "cluster-api-ocne-control-plane-controller")
	if err != nil {
		return nil, err
	}
	templateInput.OCNEControlPlaneVersion = imageConfig.Version
	templateInput.OCNEControlPlaneRepository = imageConfig.Repository
	templateInput.OCNEControlPlaneTag = imageConfig.Tag

	return templateInput, nil
}

// applyTemplate applies the CAPI provider image overrides and versions to the clusterctl.yaml template
func applyTemplate(templateContent string, params interface{}) (bytes.Buffer, error) {
	// Parse the template file
	capiYaml, err := template.New("capi").Parse(templateContent)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Apply the replacement parameters to the template
	var buf bytes.Buffer
	err = capiYaml.Execute(&buf, &params)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Return the result containing the processed template
	return buf, nil
}

func createClusterctlYaml(log *zap.SugaredLogger) error {
	// Get the image overrides and versions for the CAPI images.
	templateInput, err := getImageOverrides()
	if err != nil {
		return err
	}

	// Apply the image overrides and versions to generate clusterctl.yaml
	result, err := applyTemplate(clusterctlYamlTemplate, templateInput)
	if err != nil {
		return err
	}

	if _, err := os.Stat(defaultClusterAPIDir); os.IsNotExist(err) {
		log.Infof("Directory %s does not exist", defaultClusterAPIDir)
		err = os.MkdirAll(defaultClusterAPIDir, 0755)
		if err != nil {
			log.Errorf("Unable to create directory %v", zap.Error(err))
			return err
		}
	} else {
		log.Infof("Directory %s does exists !", defaultClusterAPIDir)
	}

	// Create the clusterctl.yaml used when initializing CAPI.
	return os.WriteFile(defaultClusterAPIDir+"/clusterctl.yaml", result.Bytes(), 0600)
}

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

/*
func newRestClient(restConfig rest.Config, gv schema.GroupVersion) (rest.Interface, error) {
	restConfig.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	restConfig.GroupVersion = &gv
	if len(gv.Group) == 0 {
		restConfig.APIPath = "/api"
	} else {
		restConfig.APIPath = "/apis"
	}

	return rest.RESTClientFor(&restConfig)
}


func createObject(kubeClientset kubernetes.Interface, restConfig rest.Config, obj runtime.Object) error {
    // Create a REST mapper that tracks information about the available resources in the cluster.
    groupResources, err := restmapper.GetAPIGroupResources(kubeClientset.Discovery())
    if err != nil {
        return err
    }
    rm := restmapper.NewDiscoveryRESTMapper(groupResources)

    // Get some metadata needed to make the REST request.
    gvk := obj.GetObjectKind().GroupVersionKind()
    gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
    mapping, err := rm.RESTMapping(gk, gvk.Version)
    if err != nil {
        return err
    }

    // Create a client specifically for creating the object.
    restClient, err := newRestClient(restConfig, mapping.GroupVersionKind.GroupVersion())
    if err != nil {
        return err
    }

    // Use the REST helper to create the object in the "default" namespace.
    restHelper := resource.NewHelper(restClient, mapping)
	_, err = restHelper.Create("default", false, obj)
	if err != nil {
		return err
	}
	return nil
}
*/

func triggerCapiClusterCreation(templatePath string, log *zap.SugaredLogger) error {
	//client, err := k8sutil.GetKubernetesClientset()
	//if err != nil {
	//	return err
	//}
	log.Infof("+++ Generated File Path name = %s +++", templatePath)
	sch := runtime.NewScheme()
	_ = scheme.AddToScheme(sch)
	_ = apiextv1beta1.AddToScheme(sch)

	decode := serializer.NewCodecFactory(sch).UniversalDeserializer().Decode
	stream, _ := os.ReadFile(templatePath)
	obj, gKV, _ := decode(stream, nil, nil)
	//if gKV.Kind == "CustomResourceDefinition" {
	//	pod := obj.(*apiextv1beta1.CustomResourceDefinition)
	//	spew.Dump(pod)
	//}
	log.Infof("++ Object = %+v +++", obj)
	log.Infof("++ gkv = %+v +++", gKV)
	return nil

}
