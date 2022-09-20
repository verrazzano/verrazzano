// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiconversion

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

const (
	ingressNGINXLabelValue = "ingress-nginx"
	ingressNGINXLabelKey   = "app.kubernetes.io/name"
)

type IngressNGINXReplicasModifierV1beta1 struct {
	replicas uint32
}

type IngressNGINXDefaultModifierV1beta1 struct {
}

func (u IngressNGINXDefaultModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	cr.Spec.Components.IngressNGINX = &v1beta1.IngressNginxComponent{}
}

var t = framework.NewTestFramework("update ingressNginx")

var nodeCount uint32

var _ = t.BeforeSuite(func() {
	var err error
	nodeCount, err = pkg.GetNodeCount()
	if err != nil {
		Fail(err.Error())
	}
})

var _ = t.AfterSuite(func() {
	m := IngressNGINXDefaultModifierV1beta1{}
	err := update.UpdateCRV1beta1(m)
	if err != nil {
		Fail(err.Error())
	}

	cr := update.GetCR()
	expectedRunning := uint32(1)
	if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
		expectedRunning = 2
	}
	update.ValidatePods(ingressNGINXLabelValue, ingressNGINXLabelKey, constants.IngressNamespace, expectedRunning, false)

})

func (u IngressNGINXReplicasModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.IngressNGINX == nil {
		cr.Spec.Components.IngressNGINX = &v1beta1.IngressNginxComponent{}
	}
	ingressNginxReplicaOverridesYaml := fmt.Sprintf(`controller:
            defaultBackend:
              replicaCount: %v`, u.replicas)
	cr.Spec.Components.IngressNGINX.ValueOverrides = createOverridesOrDie(ingressNginxReplicaOverridesYaml)
}

func createOverridesOrDie(yamlString string) []v1beta1.Overrides {
	data, err := yaml.YAMLToJSON([]byte(yamlString))
	if err != nil {
		t.Logs.Errorf("Failed to convert yaml to JSON: %s", yamlString)
		panic(err)
	}
	return []v1beta1.Overrides{
		{
			ConfigMapRef: nil,
			SecretRef:    nil,
			Values: &apiextensionsv1.JSON{
				Raw: data,
			},
		},
	}
}

var _ = t.Describe("Update ingressNGINX", Label("f:platform-lcm.update"), func() {
	t.Describe("ingressNginx update replicas with v1beta1 client", Label("f:platform-lcm.ingressNginx-update-replicas"), func() {
		t.It("ingressNginx explicit replicas", func() {
			m := IngressNGINXReplicasModifierV1beta1{replicas: nodeCount}
			err := update.UpdateCRV1beta1(m)
			if err != nil {
				Fail(err.Error())
			}
			expectedRunning := nodeCount
			update.ValidatePods(ingressNGINXLabelValue, ingressNGINXLabelKey, constants.IngressNamespace, expectedRunning, false)

		})
	})
})
