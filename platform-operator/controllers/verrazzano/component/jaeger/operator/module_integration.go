// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"reflect"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	DefaultVolumeSource      *corev1.VolumeSource            `json:"defaultVolumeSource,omitempty" patchStrategy:"replace"`
	VolumeClaimSpecTemplates []vzapi.VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

var emptyConfig = valuesConfig{}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c jaegerOperatorComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		DefaultVolumeSource:      effectiveCR.Spec.DefaultVolumeSource,
		VolumeClaimSpecTemplates: effectiveCR.Spec.VolumeClaimSpecTemplates,
	}

	if reflect.DeepEqual(emptyConfig, configSnippet) {
		return nil, nil
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c jaegerOperatorComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{common.IstioComponentName, nginx.ComponentName, cmconstants.CertManagerComponentName, opensearch.ComponentName, fluentoperator.ComponentName, keycloak.ComponentName, common.PrometheusOperatorComponentName}),
		watch.GetModuleUpdatedWatches([]string{common.IstioComponentName, nginx.ComponentName, cmconstants.CertManagerComponentName, opensearch.ComponentName, fluentoperator.ComponentName}),
		watch.GetCreateSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
		watch.GetUpdateSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
		watch.GetDeleteSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
	)
}
