// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// updateOperatorYAMLPrivateRegistry edits the operator yaml file after copying it to a new temp file
func updateOperatorYAMLPrivateRegistry(operatorFilename string, imageRegistry string, imagePrefix string) (string, error) {
	var err error

	fileObj, err := os.Open(operatorFilename)
	defer func() { fileObj.Close() }()
	if err != nil {
		return operatorFilename, err
	}
	objectsInYAML, err := k8sutil.Unmarshall(bufio.NewReader(fileObj))
	if err != nil {
		return "", err
	}
	vpoDeployIdx, vpoWebhookDeployIdx := findVPODeploymentIndices(objectsInYAML)
	vpoDeploy := &objectsInYAML[vpoDeployIdx]
	vpoWebhookDeploy := &objectsInYAML[vpoWebhookDeployIdx]

	vpoDeployUpdated, err := updatePrivateRegistryVPODeploy(vpoDeploy, imageRegistry, imagePrefix, true)
	if err != nil {
		return "", err
	}
	vpoWebhookDeployUpdated, err := updatePrivateRegistryVPODeploy(vpoWebhookDeploy, imageRegistry, imagePrefix, false)
	if err != nil {
		return "", err
	}
	if !vpoDeployUpdated && !vpoWebhookDeployUpdated {
		return operatorFilename, nil
	}
	objectsInYAML[vpoDeployIdx] = *vpoDeploy
	objectsInYAML[vpoWebhookDeployIdx] = *vpoWebhookDeploy
	bytesToWrite, err := k8sutil.Marshal(objectsInYAML)
	if err != nil {
		return "", err
	}

	var tempFile *os.File
	if tempFile, err = vzos.CreateTempFile("vz-operator-file", nil); err != nil {
		return "", err
	}
	editedOperatorFile := tempFile.Name()
	if err := os.WriteFile(editedOperatorFile, bytesToWrite, fs.ModeAppend); err != nil {
		return "", err
	}
	return editedOperatorFile, nil
}

func findVPODeploymentIndices(objectsInYAML []unstructured.Unstructured) (int, int) {
	vpoDeployIdx := -1
	vpoWebhookDeployIdx := -1
	for idx, yamlObj := range objectsInYAML {
		if yamlObj.GetObjectKind().GroupVersionKind().Kind == "Deployment" {
			if yamlObj.GetName() == constants.VerrazzanoPlatformOperator {
				vpoDeployIdx = idx
			} else if yamlObj.GetName() == constants.VerrazzanoPlatformOperatorWebhook {
				vpoWebhookDeployIdx = idx
			}
		}
	}
	return vpoDeployIdx, vpoWebhookDeployIdx
}

// updatePrivateRegistryVPODeploy updates the private registry information in the
// given verrazzano-platform-operator (or webhook) deployment YAML. Returns true if vpoDeploy was modified
func updatePrivateRegistryVPODeploy(vpoDeploy *unstructured.Unstructured, imageRegistry string, imagePrefix string, addRegistryEnvVars bool) (bool, error) {
	vpoDeployObj := vpoDeploy.Object
	containersFields := containersFields()
	initContainersFields := initContainersFields()

	containers, found, err := unstructured.NestedSlice(vpoDeployObj, containersFields...)
	if err != nil || !found {
		return false, fmt.Errorf("Failed to find containers in verrazzano-platform-operator deployment")
	}

	initContainers, found, err := unstructured.NestedSlice(vpoDeployObj, initContainersFields...)
	if err != nil || !found {
		return false, fmt.Errorf("Failed to find initContainers in verrazzano-platform-operator deployment")
	}

	updated := false
	for idx := range initContainers {
		// Use indexing on the slice to get a reference to the initCtr so we can edit it. By default
		// range returns copies so our edits won't stick
		initCtr := initContainers[idx].(map[string]interface{})
		ctrUpdated := updatePrivateRegistryOnContainer(initCtr, imageRegistry, imagePrefix)
		updated = updated || ctrUpdated
	}

	for idx := range containers {
		// Use indexing on the slice to get a reference to the container so we can edit it. By default
		// range returns copies so our edits won't stick
		container := containers[idx].(map[string]interface{})
		ctrUpdated := updatePrivateRegistryOnContainer(container, imageRegistry, imagePrefix)
		updated = updated || ctrUpdated
		if container["name"] == constants.VerrazzanoPlatformOperator {
			envUpdated := addRegistryEnvVarsToContainer(container, imageRegistry, imagePrefix)
			updated = updated || envUpdated
		}
	}

	if updated {
		if err := unstructured.SetNestedSlice(vpoDeployObj, containers, containersFields...); err != nil {
			return false, err
		}
		if err := unstructured.SetNestedSlice(vpoDeployObj, initContainers, initContainersFields...); err != nil {
			return false, err
		}
		vpoDeploy.SetUnstructuredContent(vpoDeployObj)
		return true, nil
	}
	return false, nil
}

func addRegistryEnvVarsToContainer(container map[string]interface{}, imageRegistry string, imagePrefix string) bool {
	foundRegistry := false
	foundPrefix := false
	updated := false
	env := container["env"].([]interface{})
	if env == nil {
		env = make([]interface{}, 2)
	}
	for idx := range env {
		envVar := env[idx].(map[string]interface{})
		if envVar["name"] == vpoconst.RegistryOverrideEnvVar {
			foundRegistry = true
			if envVar["value"] != imageRegistry {
				envVar["value"] = imageRegistry
				updated = true
			}
		}
		if envVar["name"] == vpoconst.ImageRepoOverrideEnvVar {
			foundPrefix = true
			if envVar["value"] != imagePrefix {
				envVar["value"] = imagePrefix
				updated = true
			}
		}
	}
	if !foundRegistry {
		env = append(env, map[string]interface{}{
			"name":  vpoconst.RegistryOverrideEnvVar,
			"value": imageRegistry,
		})
		updated = true
	}
	if !foundPrefix {
		env = append(env, map[string]interface{}{
			"name":  vpoconst.ImageRepoOverrideEnvVar,
			"value": imagePrefix,
		})
		updated = true
	}
	container["env"] = env
	return updated
}

func updatePrivateRegistryOnContainer(container map[string]interface{}, imageRegistry string, imagePrefix string) bool {
	curImage := container["image"].(string)
	suffixPattern := fmt.Sprintf("/verrazzano/%s", constants.VerrazzanoPlatformOperator)
	imageSuffix := curImage[strings.LastIndex(curImage, suffixPattern)+1:]
	newImage := fmt.Sprintf("%s/%s/%s", imageRegistry, imagePrefix, imageSuffix)
	if newImage == curImage {
		// no update needed
		return false
	}
	container["image"] = newImage
	return true
}

func containersFields() []string {
	return []string{"spec", "template", "spec", "containers"}
}

func initContainersFields() []string {
	return []string{"spec", "template", "spec", "initContainers"}
}
