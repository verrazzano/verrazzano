// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
)

const defaultImageKey = "image"
const slash = "/"
const tagSep = ":"

const (
	// Pod Substring for finding the platform operator pod
	platformOperatorPodNameSearchString = "verrazzano-platform-operator"
)

// Bom contains information related to the bill of materials along with structures to process it.
// The bom file is verrazzano-bom.json and it mainly has image information.
type Bom struct {
	// bomDoc contains the contents of the JSON bom, in go structure format.
	bomDoc BomDoc

	// subComponentMap is a map of subcomponents keyed by subcomponent name.
	subComponentMap map[string]*BomSubComponent
}

// BomDoc contains product metadata for components installed by Verrazzano.
// Currently, this metadata only describes images and image repositories.
// This information is needed by the helm charts and used during install/upgrade.
type BomDoc struct {
	// Registry is the top level registry URI which contains the image repositories.
	// An example is ghcr.io
	Registry string `json:"registry"`

	// Version is the verrazzano version corresponding to the build
	Version string `json:"version"`

	// Components is the array of component boms
	Components []BomComponent `json:"components"`

	// SupportedKubernetesVersions is the array of supported Kubernetes versions
	SupportedKubernetesVersions []string `json:"supportedKubernetesVersions"`
}

// BomComponent represents a high level component, such as Istio.
// Each component has one or more subcomponents.
type BomComponent struct {
	// The name of the component, for example: Istio
	Name string `json:"name"`
	// Version of the component
	Version string `json:"version,omitempty"`
	// SubComponents is the array of subcomponents in the component
	SubComponents []BomSubComponent `json:"subcomponents"`
}

// BomSubComponent contains the bom information for a single helm chart.
// Istio is an example of a component with several subcomponents.
type BomSubComponent struct {
	Name string `json:"name"`

	// Repository is the name of the repository within a registry.  This is combined
	// with the registry Value to form the image URL prefix, for example: ghcr.io/verrazzano,
	// where ghci.io is the registry and Verrazzano is the repository name.
	Repository string `json:"repository"`

	// Override the registry within a subcomponent
	Registry string `json:"registry"`

	// Images is the array of images for this subcomponent
	Images []BomImage `json:"images"`
}

// BomImage describes a single image used by one of the helm charts.  This structure
// is needed to render the helm chart manifest, so the image fields are correctly populated
// in the resulting YAML.  There is no helm standard which specifies the keys for
// image information used by a template, each product usually has a custom way to do this.
// The helm keys fields in this structure specify those custom keys.
type BomImage struct {
	// ImageName specifies the name of the image tag, such as `nginx-ingress-controller`
	ImageName string `json:"image"`

	// ImageTag specifies the name of the image tag, such as `0.46.0-20210510134749-abc2d2088`
	ImageTag string `json:"tag"`

	// Registry is the image registry. It can be used to override the subcomponent registry
	Registry string `json:"registry,omitempty"`

	// Repository is the image repository. It can be used to override the subcomponent repository
	Repository string `json:"repository,omitempty"`

	// HelmRegistryKey is the helm template Key which identifies the registry for an image.  An example is
	// `image.registry` in external-dns.  The default is empty string.
	HelmRegistryKey string `json:"helmRegKey"`

	// HelmRepoKey is the helm template Key which stores the value of the repository for an image.
	HelmRepoKey string `json:"helmRepoKey"`

	// HelmImageKey is the helm template Key which identifies the base image name, without the registry or parent repo
	// parts of the path.  For example, if the full image name is myreg.io/foo/bar/myimage:v1.0, the value of this key
	// will be "myimage".  See the Istio proxyv2 entry in the BOM file for an example.
	HelmImageKey string `json:"helmImageKey"`

	// HelmTagKey is the helm template Key which stores the value of the image tag.  For example,
	// if the full image name is myreg.io/foo/bar/myimage:v1.0, the value of this key will be "v1.0"
	HelmTagKey string `json:"helmTagKey"`

	// HelmFullImageKey is the helm path Key which identifies the image name without the registry or tag.  For example,
	// if the full image name is myreg.io/foo/bar/myimage:v1.0, the value of this key will be
	// "foo/bar/myimage".
	HelmFullImageKey string `json:"helmFullImageKey"`

	// HelmRegistryAndRepoKey is a helm Key which stores the registry and repo parts of the image path.  For example,
	// if the full image name is myreg.io/foo/bar/myimage:v1.0 the value of this key will be "myreg.io/foo/bar".
	// See `image.repository` in the external-dns component
	HelmRegistryAndRepoKey string `json:"helmRegistryAndRepoKey"`
}

// keyVal defines the Key, Value pair used to override a single helm Value
type KeyValue struct {
	Key       string
	Value     string
	SetString bool // for --set-string
	SetFile   bool // for --set-file
	IsFile    bool // for -f
}

// Create a new BOM instance from a JSON file
func NewBom(bomPath string) (Bom, error) {
	jsonBom, err := os.ReadFile(bomPath)
	if err != nil {
		return Bom{}, err
	}
	return newBOMFromJSON(jsonBom)
}

// newBOMFromJSON Create a new BOM instance from a JSON payload
func newBOMFromJSON(jsonBom []byte) (Bom, error) {
	bom := Bom{
		subComponentMap: make(map[string]*BomSubComponent),
	}
	err := bom.init(string(jsonBom))
	if err != nil {
		return Bom{}, err
	}
	return bom, nil
}

// Initialize the BomInfo.  Load the Bom from the JSON file and build
// a map of subcomponents
func (b *Bom) init(jsonBom string) error {
	// Convert the json into a to bom
	if err := json.Unmarshal([]byte(jsonBom), &b.bomDoc); err != nil {
		return err
	}

	// Build a map of subcomponents
	for _, comp := range b.bomDoc.Components {
		for i, sub := range comp.SubComponents {
			b.subComponentMap[sub.Name] = &comp.SubComponents[i]
		}
	}
	return nil
}

// GetRegistry Gets the registry name
func (b *Bom) GetRegistry() string {
	return b.bomDoc.Registry
}

// GetVersion gets the BOM product version
func (b *Bom) GetVersion() string {
	return b.bomDoc.Version
}

// GetComponent gets the BOM component
func (b *Bom) GetComponent(componentName string) (*BomComponent, error) {
	for _, comp := range b.bomDoc.Components {
		if comp.Name == componentName {
			return &comp, nil
		}
	}
	return nil, errors.New("unknown component " + componentName)
}

func (b *Bom) GetComponentVersion(componentName string) (string, error) {
	component, err := b.GetComponent(componentName)
	if err != nil {
		return "", err
	}
	if len(component.Version) == 0 {
		return "", fmt.Errorf("Did not find valid version for component %s: %s", component, component.Version)
	}
	return component.Version, nil
}

// GetSubcomponent gets the bom subcomponent
func (b *Bom) GetSubcomponent(subComponentName string) (*BomSubComponent, error) {
	sc, ok := b.subComponentMap[subComponentName]
	if !ok {
		return nil, errors.New("unknown subcomponent " + subComponentName)
	}
	return sc, nil
}

// GetSubcomponentImages the imageBoms for a subcomponent
func (b *Bom) GetSubcomponentImages(subComponentName string) ([]BomImage, error) {
	sc, err := b.GetSubcomponent(subComponentName)
	if err != nil {
		return nil, err
	}
	return sc.Images, nil
}

func (b *Bom) FindImage(sc *BomSubComponent, imageName string) (BomImage, error) {
	for _, image := range sc.Images {
		if image.ImageName == imageName {
			return image, nil
		}
	}
	return BomImage{}, fmt.Errorf("Image %s not found for sub-component %s", imageName, sc.Name)
}

// GetSubcomponentImageCount returns the number of subcomponent images
func (b *Bom) GetSubcomponentImageCount(subComponentName string) int {
	imageBom, ok := b.subComponentMap[subComponentName]
	if !ok {
		return 0
	}
	return len(imageBom.Images)
}

// BuildImageOverrides builds the image overrides array for the subComponent.
// Each override has an array of 1-n Helm Key:Value pairs
func (b *Bom) BuildImageOverrides(subComponentName string) ([]KeyValue, error) {
	kvs, _, err := b.BuildImageStrings(subComponentName)
	return kvs, err
}

// GetImageNameList build the image names and return a list of image names
func (b *Bom) GetImageNameList(subComponentName string) ([]string, error) {
	_, images, err := b.BuildImageStrings(subComponentName)
	return images, err
}

// BuildImageStrings builds the image overrides array for the subComponent.
// Each override has an array of 1-n Helm Key:Value pairs
// Also return the set of fully constructed image names
func (b *Bom) BuildImageStrings(subComponentName string) ([]KeyValue, []string, error) {
	var fullImageNames []string
	sc, ok := b.subComponentMap[subComponentName]
	if !ok {
		return nil, nil, errors.New("unknown subcomponent " + subComponentName)
	}

	// Loop through the images used by this subcomponent, building
	// the list of Key:Value pairs.  At the very least, this will build
	// a single Value for the fully qualified image name in the format of
	// registry/repository/image.tag
	var kvs []KeyValue
	for _, imageBom := range sc.Images {
		partialImageNameBldr := strings.Builder{}
		registry := b.ResolveRegistry(sc, imageBom)
		repo := b.ResolveRepo(sc, imageBom)

		// Normally, the registry is the first segment of the image name, for example "ghcr.io/"
		// However, there are exceptions like in external-dns, where the registry is a separate helm field,
		// in which case the registry is omitted from the image full name.
		if imageBom.HelmRegistryKey != "" {
			kvs = append(kvs, KeyValue{
				Key:   imageBom.HelmRegistryKey,
				Value: registry,
			})
		} else {
			partialImageNameBldr.WriteString(registry)
			partialImageNameBldr.WriteString(slash)
		}

		// Either write the repo name Key Value, or append it to the full image path
		if imageBom.HelmRepoKey != "" {
			kvs = append(kvs, KeyValue{
				Key:   imageBom.HelmRepoKey,
				Value: repo,
			})
		} else {
			if len(repo) > 0 {
				partialImageNameBldr.WriteString(repo)
				partialImageNameBldr.WriteString(slash)
			}
		}

		// If the Registry/Repo key is defined then set it
		if imageBom.HelmRegistryAndRepoKey != "" {
			regAndRep := registry + "/" + repo
			kvs = append(kvs, KeyValue{
				Key:   imageBom.HelmRegistryAndRepoKey,
				Value: regAndRep,
			})
		}

		// Either write the image name Key Value, or append it to the full image path
		if imageBom.HelmImageKey != "" {
			kvs = append(kvs, KeyValue{
				Key:   imageBom.HelmImageKey,
				Value: imageBom.ImageName,
			})
		} else {
			partialImageNameBldr.WriteString(imageBom.ImageName)
		}

		// Either write the tag name Key Value, or append it to the full image path
		if imageBom.HelmTagKey != "" {
			kvs = append(kvs, KeyValue{
				Key:   imageBom.HelmTagKey,
				Value: imageBom.ImageTag,
			})
		} else {
			partialImageNameBldr.WriteString(tagSep)
			partialImageNameBldr.WriteString(imageBom.ImageTag)
		}

		// This partial image path may be a subset of the full image name or the entire image path
		partialImagePath := partialImageNameBldr.String()

		// If the image path Key is present the create the kv with the partial image path
		if imageBom.HelmFullImageKey != "" {
			kvs = append(kvs, KeyValue{
				Key:   imageBom.HelmFullImageKey,
				Value: partialImagePath,
			})
		}
		// Default the image Key if there are no specified tags.  Keycloak theme needs this
		if len(kvs) == 0 {
			kvs = append(kvs, KeyValue{
				Key:   defaultImageKey,
				Value: partialImagePath,
			})
		}
		// Add the full image name to the list
		fullImageName := fmt.Sprintf("%s/%s/%s:%s", registry, repo, imageBom.ImageName, imageBom.ImageTag)
		fullImageNames = append(fullImageNames, fullImageName)
	}
	return kvs, fullImageNames, nil
}

// ResolveRegistry resolves the registry name using the ENV var if it exists.
func (b *Bom) ResolveRegistry(sc *BomSubComponent, img BomImage) string {
	// Get the registry ENV override, if it doesn't exist use the default
	registry := os.Getenv(constants.RegistryOverrideEnvVar)
	if registry == "" {
		registry = b.bomDoc.Registry
		if len(sc.Registry) > 0 {
			registry = sc.Registry
		}
		if len(img.Registry) > 0 {
			registry = img.Registry
		}
	}
	return registry
}

// ResolveRepo resolves the repository name using the ENV var if it exists.
func (b *Bom) ResolveRepo(sc *BomSubComponent, img BomImage) string {
	// Get the repo ENV override.  This needs to get prepended to the bom repo
	userRepo := os.Getenv(constants.ImageRepoOverrideEnvVar)
	repo := sc.Repository
	if len(img.Repository) > 0 {
		repo = img.Repository
	}

	if userRepo != "" {
		repo = userRepo + slash + repo
	}
	return repo
}

// FindKV searches an array of KeyValue structs for a Key and returns the Value if found, or returns an empty string
func FindKV(kvs []KeyValue, key string) string {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

// GetSupportedKubernetesVersion gets supported Kubernetes versions
func (b *Bom) GetSupportedKubernetesVersion() []string {
	return b.bomDoc.SupportedKubernetesVersions
}

// GetBOMDoc gets the BOM from the platform operator in the cluster and build the BOM structure from it
func GetBOMDoc(kubeClient kubernetes.Interface, config *rest.Config) (*BomDoc, error) {

	pods, err := kubeClient.CoreV1().Pods(constants.VerrazzanoInstallNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pod *corev1.Pod
	for i := range pods.Items {
		if pods.Items[i].Labels["app"] == "verrazzano-platform-operator" && pods.Items[i].Status.Phase == corev1.PodRunning {
			pod = &pods.Items[i]
			break
		}
	}
	if pod == nil {
		return nil, fmt.Errorf("failed to find a running verrazzano-platform-operator pod in the verrazzano-install namespace")
	}

	catCommand := []string{"cat", "/verrazzano/platform-operator/verrazzano-bom.json"}
	stdout, stderr, err := k8sutil.ExecPod(kubeClient, config, pod, "verrazzano-platform-operator", catCommand)
	if err != nil || stderr != "" || len(stdout) == 0 {
		return nil, fmt.Errorf("error retrieving compatible kubernetes versions from the verrazzano-platform-operator pod, stdout: %s, stderr: %s, err: %v",
			stdout, stderr, err)
	}

	var bomDoc BomDoc
	err = json.Unmarshal([]byte(stdout), &bomDoc)
	return &bomDoc, err
}

func ValidateKubernetesVersionSupported(kubernetesVersionString string, supportedVersionsString []string) error {
	kubernetesVersion, err := semver.NewSemVersion(kubernetesVersionString)
	if err != nil {
		return fmt.Errorf("invalid kubernetes version %s, error %v", kubernetesVersionString, err.Error())
	}

	for _, supportedVersionString := range supportedVersionsString {
		supportedVersion, err := semver.NewSemVersion(supportedVersionString)
		if err != nil {
			return fmt.Errorf("invalid supported kubernetes version %s, error %v", supportedVersion.ToString(), err.Error())
		}

		if kubernetesVersion.IsEqualToOrPatchVersionOf(supportedVersion) {
			return nil
		}
	}

	return fmt.Errorf("kubernetes version %s not supported, supported versions are %v", kubernetesVersion.ToString(), supportedVersionsString)
}
