// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	corev1 "k8s.io/api/core/v1"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/templates"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStatusCmd(t *testing.T) {
	name := "verrazzano"
	namespace := "test"
	version := "1.2.3"
	consoleURL := "https://verrazzano.default.10.107.141.8.nip.io"
	keycloakURL := "https://keycloak.default.10.107.141.8.nip.io"
	rancherURL := "https://rancher.default.10.107.141.8.nip.io"
	osURL := "https://elasticsearch.vmi.system.10.107.141.8.nip.io"
	kibanaURL := "https://kibana.vmi.system.10.107.141.8.nip.io"
	grafanaURL := "https://grafana.vmi.system.10.107.141.8.nip.io"
	prometheusURL := "https://prometheus.vmi.system.10.107.141.8.nip.io"
	kialiURL := "https://kiali.vmi.system.10.107.141.8.nip.io"

	vz := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.Dev,
		},
		Status: vzapi.VerrazzanoStatus{
			Version: version,
			VerrazzanoInstance: &vzapi.InstanceInfo{
				ConsoleURL:    &consoleURL,
				KeyCloakURL:   &keycloakURL,
				RancherURL:    &rancherURL,
				ElasticURL:    &osURL,
				KibanaURL:     &kibanaURL,
				GrafanaURL:    &grafanaURL,
				PrometheusURL: &prometheusURL,
				KialiURL:      &kialiURL,
			},
			Conditions: nil,
			State:      vzapi.VzStateInstalling,
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}

	// Template map for status command output
	templateMap := map[string]string{
		"verrazzano_name":    name,
		"verrazzano_version": version,
		"verrazzano_state":   string(vzapi.VzStateInstalling),
		"console_url":        consoleURL,
		"keycloak_url":       keycloakURL,
		"rancher_url":        rancherURL,
		"os_url":             osURL,
		"kibana_url":         kibanaURL,
		"grafana_url":        grafanaURL,
		"prometheus_url":     prometheusURL,
		"kiali_url":          kialiURL,
		"install_profile":    string(vzapi.Dev),
	}

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&vz).Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	statusCmd.SetArgs([]string{fmt.Sprintf("--%s", nameFlag), name, fmt.Sprintf("--%s", namespaceFlag), namespace})
	err := statusCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	expectedResult, err := templates.ApplyTemplate(statusOutputTemplate, templateMap)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)

	// Run the status command with the incorrect namespace, expect that the Verrazzano resource is not found
	errBuf.Reset()
	buf.Reset()
	statusCmd.SetArgs([]string{fmt.Sprintf("--%s", nameFlag), name, fmt.Sprintf("--%s", namespaceFlag), "default"})
	err = statusCmd.Execute()
	assert.Error(t, err)
	assert.True(t, strings.Contains(errBuf.String(), "Failed to find Verrazzano with name \"verrazzano\" in namespace \"default\""))

	// Run the status command with the incorrect name, expect that the Verrazzano resource is not found
	errBuf.Reset()
	buf.Reset()
	statusCmd.SetArgs([]string{fmt.Sprintf("--%s", nameFlag), "bad", fmt.Sprintf("--%s", namespaceFlag), namespace})
	err = statusCmd.Execute()
	assert.Error(t, err)
	assert.True(t, strings.Contains(errBuf.String(), "Failed to find Verrazzano with name \"bad\" in namespace \"test\""))
}

func makeVerrazzanoComponentStatusMap() vzapi.ComponentStatusMap {
	statusMap := make(vzapi.ComponentStatusMap)
	for _, comp := range registry.GetComponents() {
		if comp.IsOperatorInstallSupported() {
			statusMap[comp.Name()] = &vzapi.ComponentStatusDetails{
				Name: comp.Name(),
				Conditions: []vzapi.Condition{
					{
						Type:   vzapi.CondInstallComplete,
						Status: corev1.ConditionTrue,
					},
				},
				State: vzapi.CompStateReady,
			}
		}
	}
	return statusMap
}
