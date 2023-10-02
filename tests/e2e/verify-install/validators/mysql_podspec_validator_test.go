// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package validators

import (
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Used by this test to make an illegal edit, to ensure that the apply is rejected but still generate the
	// warnings we're looking for
	illegalEnvironmentName = "asfsdafsdafadsfdsafdsafdsafadsfadsfadsfadsfdsafadsfsdafdsafasdfasdfsadfasdfasdfdasfasdfasdfasadsfasdfdasfasdfdsafdsafasd"

	podSpecJSON = `
{
  "podSpec": {
    "affinity": {
      "podAffinity": {
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 200,
            "podAffinityTerm": {
              "labelSelector": {
                "matchLabels": {
                  "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-mysql-server",
                  "app.kubernetes.io/name": "mysql-innodbcluster-mysql-server"
                }
              },
              "topologyKey": "kubernetes.io/hostname"
            }
          }
        ]
      }
    }
  }
}`

	warningSubstring = "Modifications to MySQL server pod specs do not trigger an automatic restart of the stateful set"
)

type warningInfo struct {
	code  int
	agent string
	text  string
}

type mysqlPodSpecUpdater struct {
	warnings []warningInfo
}

func (j *mysqlPodSpecUpdater) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	// Attempt to make an illegal edit to the Jaeger configuration to ensure its component validation is working properly
	trueValue := true
	if cr.Spec.Components.JaegerOperator == nil {
		cr.Spec.Components.JaegerOperator = &v1beta1.JaegerOperatorComponent{}
	}
	// Include an illegal edit to prevent the edit from succeeding
	cr.Spec.EnvironmentName = illegalEnvironmentName
	if cr.Spec.Components.Keycloak == nil {
		cr.Spec.Components.Keycloak = &v1beta1.KeycloakComponent{}
	}
	cr.Spec.Components.Keycloak.Enabled = &trueValue
	illegalValuesObj := generateJSONValuesObj()
	cr.Spec.Components.Keycloak.MySQL.InstallOverrides.ValueOverrides = append(
		cr.Spec.Components.Keycloak.MySQL.InstallOverrides.ValueOverrides,
		v1beta1.Overrides{Values: illegalValuesObj})
}

func (j *mysqlPodSpecUpdater) ModifyCR(cr *v1alpha1.Verrazzano) {
	// Attempt to make an illegal edit to the Jaeger configuration to ensure its component validation is working properly
	trueValue := true
	if cr.Spec.Components.JaegerOperator == nil {
		cr.Spec.Components.JaegerOperator = &v1alpha1.JaegerOperatorComponent{}
	}
	// Include an illegal edit to prevent the edit from succeeding
	cr.Spec.EnvironmentName = illegalEnvironmentName
	if cr.Spec.Components.Keycloak == nil {
		cr.Spec.Components.Keycloak = &v1alpha1.KeycloakComponent{}
	}
	cr.Spec.Components.Keycloak.Enabled = &trueValue
	illegalValuesObj := generateJSONValuesObj()
	cr.Spec.Components.Keycloak.MySQL.InstallOverrides.ValueOverrides = append(
		cr.Spec.Components.Keycloak.MySQL.InstallOverrides.ValueOverrides,
		v1alpha1.Overrides{Values: illegalValuesObj})
}

func (j *mysqlPodSpecUpdater) HandleWarningHeader(code int, agent string, text string) {
	j.warnings = append(j.warnings, warningInfo{code: code, agent: agent, text: text})
}

var _ update.CRModifier = &mysqlPodSpecUpdater{}
var _ update.CRModifierV1beta1 = &mysqlPodSpecUpdater{}
var _ rest.WarningHandler = &mysqlPodSpecUpdater{}

func generateJSONValuesObj() *apiextensionsv1.JSON {
	illegalOverride := podSpecJSON
	illegalValuesObj := &apiextensionsv1.JSON{
		Raw: []byte(illegalOverride),
	}
	return illegalValuesObj
}

// runValidatorTestV1Beta1 Attempt to generate the MySQL values webhook warning, with an illegal edit so the overall edit is rejected
func runMySQLPodspecEditWarningTestV1Beta1() {
	updater := &mysqlPodSpecUpdater{}
	checkExpectations(update.UpdateCRV1beta1(updater, client.DryRunAll), updater)
}

// runValidatorTestV1Alpha1 Attempt to use an illegal overrides value on the Jaeger operator configuration using the v1alpha1 API
func runMySQLPodspecEditWarningTestV1Alpha1() {
	updater := &mysqlPodSpecUpdater{}
	checkExpectations(update.UpdateCR(updater, client.DryRunAll), updater)
}

func checkExpectations(err error, updater *mysqlPodSpecUpdater) {
	Eventually(func() []string {
		t.Logs.Infof("Verifies that an update to the MySQL overrides containing a podSpec value issues a warning to " +
			"the user; also makes an illegal edit to avoid mutating the system but also generate the warning")
		if err == nil {
			t.Logs.Info("Did not get an error on illegal update")
			return []string{}
		}
		if err != nil {
			t.Logs.Infof("Update error: %s", err.Error())
		}
		var warningText []string
		for _, warning := range updater.warnings {
			warningText = append(warningText, warning.text)
			t.Logs.Infof("Warning: %v", warning)
		}
		return warningText
	}, waitTimeout, pollingInterval).Should(ContainElements(ContainSubstring(warningSubstring)))
}
