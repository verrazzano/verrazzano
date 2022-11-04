// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package validators

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// runValidatorTest Attempt to use an illegal overrides value on the Jaeger operator configuration
func runValidatorTest() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(err.Error())
		return
	}
	cr, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfigPath)
	if err != nil {
		Fail(err.Error())
		return
	}
	// Attempt to make an illegal edit to the Jaeger configuration to ensure its component validation is working properly
	trueValue := true
	if cr.Spec.Components.JaegerOperator == nil {
		cr.Spec.Components.JaegerOperator = &v1beta1.JaegerOperatorComponent{}
	}
	cr.Spec.Components.JaegerOperator.Enabled = &trueValue
	illegalOverride := `{"nameOverride": "testjaeger"}`
	illegalValuesObj := &apiextensionsv1.JSON{
		Raw: []byte(illegalOverride),
	}
	cr.Spec.Components.JaegerOperator.InstallOverrides.ValueOverrides = append(
		cr.Spec.Components.JaegerOperator.InstallOverrides.ValueOverrides,
		v1beta1.Overrides{Values: illegalValuesObj})

	t.Logs.Infof("Attempting to set an illegal override value for Jaeger component: %v", string(illegalValuesObj.Raw))
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		Fail(err.Error())
		return
	}
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		Fail(err.Error())
		return
	}
	vzClient := client.VerrazzanoV1beta1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	if err != nil {
		t.Logs.Infof("Update error: %s", err.Error())
	}
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(ContainSubstring("the Jaeger Operator Helm chart value nameOverride cannot be overridden"))
}
