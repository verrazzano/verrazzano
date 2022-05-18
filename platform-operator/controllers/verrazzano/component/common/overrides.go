package common

import (
	"context"
	"fmt"
	os2 "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
)

// RetrieveInstallOverrideResources takes the list of Overrides and returns a list of key value pairs
func RetrieveInstallOverrideResources(ctx spi.ComponentContext, overrides []v1alpha1.Overrides, name string) ([]string, error) {
	var files []string
	var file *os.File
	for _, override := range overrides {
		// Check if ConfigMapRef is populated and gather helm file
		if override.ConfigMapRef != nil {
			// Get the ConfigMap
			configMap := &v1.ConfigMap{}
			selector := override.ConfigMapRef
			nsn := types.NamespacedName{Name: selector.Name, Namespace: ctx.EffectiveCR().Namespace}
			optional := selector.Optional
			err := ctx.Client().Get(context.TODO(), nsn, configMap)
			if err != nil {
				if optional == nil || !*optional {
					err := ctx.Log().ErrorfNewErr("Could not get Configmap %s from namespace %s: %v", nsn.Name, nsn.Namespace, err)
					return files, err
				}
				ctx.Log().Debugf("Optional Configmap %s from namespace %s not found", nsn.Name, nsn.Namespace)
				continue
			}

			// Get resource data
			fieldData, ok := configMap.Data[selector.Key]
			if !ok {
				if optional == nil || !*optional {
					err := ctx.Log().ErrorfNewErr("Could not get Data field %s from Resource %s from namespace %s", selector.Key, nsn.Name, nsn.Namespace)
					return files, err
				}
				ctx.Log().Debugf("Optional Resource %s from namespace %s missing Data key %s", nsn.Name, nsn.Namespace, selector.Key)
			}

			// Create the temp file for the data
			file, err = os2.CreateTempFile(ctx.Log(), fmt.Sprintf("install-overrides-%s-*.yaml", name), []byte(fieldData))
			if err != nil {
				return files, err
			}
			files = append(files, file.Name())
		}
		// Check if SecretRef is populated and gather helm file
		if override.SecretRef != nil {
			// Get the Secret
			sec := &v1.Secret{}
			selector := override.SecretRef
			nsn := types.NamespacedName{Name: selector.Name, Namespace: ctx.EffectiveCR().Namespace}
			optional := selector.Optional
			err := ctx.Client().Get(context.TODO(), nsn, sec)
			if err != nil {
				if optional == nil || !*optional {
					err := ctx.Log().ErrorfNewErr("Could not get Secret %s from namespace %s: %v", nsn.Name, nsn.Namespace, err)
					return files, err
				}
				ctx.Log().Debugf("Optional Secret %s from namespace %s not found", nsn.Name, nsn.Namespace)
				continue
			}

			dataStrings := map[string]string{}
			for key, val := range sec.Data {
				dataStrings[key] = string(val)
			}

			// Get resource data
			fieldData, ok := dataStrings[selector.Key]
			if !ok {
				if optional == nil || !*optional {
					err := ctx.Log().ErrorfNewErr("Could not get Data field %s from Resource %s from namespace %s", selector.Key, nsn.Name, nsn.Namespace)
					return files, err
				}
				ctx.Log().Debugf("Optional Resource %s from namespace %s missing Data key %s", nsn.Name, nsn.Namespace, selector.Key)
			}

			// Create the temp file for the data
			file, err = os2.CreateTempFile(ctx.Log(), "install-overrides-%s-*.yaml", []byte(fieldData))
			if err != nil {
				return files, err
			}
			files = append(files, file.Name())
		}
	}
	return files, nil
}
