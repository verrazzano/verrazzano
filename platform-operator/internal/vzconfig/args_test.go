// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"fmt"
	"testing"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	nodePort                 = installv1beta1.NodePort
	compName                 = "istio"
	ExternalIPArg            = "gateways.istio-ingressgateway.externalIPs"
	specServiceJSONPath      = "spec.components.ingressGateways.0.k8s.service"
	externalIPJsonPathSuffix = "externalIPs.0"
	typeJSONPathSuffix       = "type"
	externalIPJsonPath       = specServiceJSONPath + "." + externalIPJsonPathSuffix
)

// TestCheckExternalIPsArgs tests CheckExternalIPsArgs
// GIVEN a v1alpha1 VZ CR with ExternalIP IstioOverrides
// THEN return an error
// IF the override key is not found or the IP is not valid
func TestCheckExternalIPsArgs(t *testing.T) {
	asserts := assert.New(t)

	vz := getVZWithIstioOverride("0.0.0.0")
	err := CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName)
	asserts.NoError(err)

	vz = getVZWithIstioOverride("0.0.0")
	err = CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName)
	asserts.Error(err)
	asserts.Contains(err.Error(), "Must be a proper IP address format")

	vz = getVZWithIstioOverride("")
	err = CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName)
	asserts.Error(err)
	asserts.Contains(err.Error(), "not found for component")
}

// TestCheckExternalIPsOverridesArgs tests CheckExternalIPsOverridesArgs
// GIVEN a v1beta1 VZ CR with ExternalIP IstioOverrides
// THEN return an error
// IF the IP is not valid
func TestCheckExternalIPsOverridesArgs(t *testing.T) {
	asserts := assert.New(t)

	vz := getv1beta1VZWithIstioOverride("0.0.0.0")
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName)
	asserts.NoError(err)

	vz = getv1beta1VZWithIstioOverride("0.0.0")
	err = CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName)
	asserts.Error(err)
	asserts.Contains(err.Error(), "Must be a proper IP address format")
}

// TestCheckExternalIPsOverridesArgsWithPaths tests CheckExternalIPsOverridesArgsWithPaths
// GIVEN a v1beta1 VZ CR with ExternalIP IstioOverrides
// THEN return an error
// IF the IP is not valid
func TestCheckExternalIPsOverridesArgsWithPaths(t *testing.T) {
	asserts := assert.New(t)

	vz := getv1beta1VZWithIstioOverride("0.0.0.0")
	err := CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName)
	asserts.NoError(err)

	vz = getv1beta1VZWithIstioOverride("0.0.0")
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName)
	asserts.Error(err)
	asserts.Contains(err.Error(), "Must be a proper IP address format")
}

// getIstioOverride returns an Istio override in json format
func getIstioOverride(externalIP string) []byte {
	override := fmt.Sprintf(`spec:
  components:
    ingressGateways:
      - name: istio-ingressgateway
        k8s:
          service:
            type: NodePort
            externalIPs:
            - %s`, externalIP)

	json, _ := yaml.ToJSON([]byte(override))
	return json
}

// getVZWithIstioOverride returns v1aplha1 vz CR with Istio Component overrides
func getVZWithIstioOverride(externalIP string) vzapi.Verrazzano {
	if len(externalIP) == 0 {
		return vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Istio: &vzapi.IstioComponent{},
				},
			},
		}
	}
	vz := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: getIstioOverride(externalIP),
								},
							},
						},
					},
				},
			},
		},
	}
	return vz
}

// getv1beta1VZWithIstioOverride returns v1beta1 vz CR with Istio Component overrides
func getv1beta1VZWithIstioOverride(externalIP string) installv1beta1.Verrazzano {
	vz := installv1beta1.Verrazzano{
		Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					InstallOverrides: installv1beta1.InstallOverrides{
						ValueOverrides: []installv1beta1.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: getIstioOverride(externalIP),
								},
							},
						},
					},
				},
			},
		},
	}
	return vz
}
