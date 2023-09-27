// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"reflect"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	VolumeSource             *corev1.VolumeSource            `json:"volumeSource,omitempty"`
	MySQLInstallArgs         []vzapi.InstallArgs             `json:"mysqlInstallArgs,omitempty"`
	DefaultVolumeSource      *corev1.VolumeSource            `json:"defaultVolumeSource,omitempty"`
	VolumeClaimSpecTemplates []vzapi.VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty"`
}

var emptyConfig = valuesConfig{}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c mysqlComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		DefaultVolumeSource:      effectiveCR.Spec.DefaultVolumeSource,
		VolumeClaimSpecTemplates: effectiveCR.Spec.VolumeClaimSpecTemplates,
	}

	keycloak := effectiveCR.Spec.Components.Keycloak
	if keycloak != nil {
		configSnippet.VolumeSource = keycloak.MySQL.VolumeSource.DeepCopy()
		configSnippet.MySQLInstallArgs = keycloak.MySQL.MySQLInstallArgs
	}

	if reflect.DeepEqual(emptyConfig, configSnippet) {
		return nil, nil
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c mysqlComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.GetModuleInstalledWatches([]string{common.IstioComponentName, MySQLOperatorComponentName, fluentoperator.ComponentName})
}
