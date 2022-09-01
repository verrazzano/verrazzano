// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/templates"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestStatusCmd tests the status command
// GIVEN an environment with a single VZ resource
//  WHEN I run the command vz status
//  THEN expect a successful status report
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
	jaegerURL := "https://jaeger.default.10.107.141.8.nip.io"

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
				JaegerURL:     &jaegerURL,
			},
			Conditions: nil,
			State:      vzapi.VzStateReconciling,
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}

	// Template map for status command output
	templateMap := map[string]string{
		"verrazzano_name":      name,
		"verrazzano_namespace": namespace,
		"verrazzano_version":   version,
		"verrazzano_state":     string(vzapi.VzStateReconciling),
		"console_url":          consoleURL,
		"keycloak_url":         keycloakURL,
		"rancher_url":          rancherURL,
		"os_url":               osURL,
		"kibana_url":           kibanaURL,
		"grafana_url":          grafanaURL,
		"prometheus_url":       prometheusURL,
		"kiali_url":            kialiURL,
		"jaeger_url":           jaegerURL,
		"install_profile":      string(vzapi.Dev),
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz).Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	expectedResult, err := templates.ApplyTemplate(statusOutputTemplate, templateMap)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

// TestStatusCmdDefaultProfile tests the status command
// GIVEN an environment with a single VZ resource and the default prod profile
//  WHEN I run the command vz status
//  THEN expect a successful status report
func TestStatusCmdDefaultProfile(t *testing.T) {
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
	jaegerURL := "https://jaeger.default.10.107.141.8.nip.io"

	vz := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
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
				JaegerURL:     &jaegerURL,
			},
			Conditions: nil,
			State:      vzapi.VzStateReconciling,
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}

	// Template map for status command output
	templateMap := map[string]string{
		"verrazzano_name":      name,
		"verrazzano_namespace": namespace,
		"verrazzano_version":   version,
		"verrazzano_state":     string(vzapi.VzStateReconciling),
		"console_url":          consoleURL,
		"keycloak_url":         keycloakURL,
		"rancher_url":          rancherURL,
		"os_url":               osURL,
		"kibana_url":           kibanaURL,
		"grafana_url":          grafanaURL,
		"prometheus_url":       prometheusURL,
		"kiali_url":            kialiURL,
		"jaeger_url":           jaegerURL,
		"install_profile":      string(vzapi.Prod),
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz).Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	expectedResult, err := templates.ApplyTemplate(statusOutputTemplate, templateMap)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

// TestVZNotFound tests the status command
// GIVEN an environment with a no VZ resources exist
//  WHEN I run the command vz status
//  THEN expect an error of no VZ resources found
func TestVZNotFound(t *testing.T) {

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects().Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Failed to find any Verrazzano resources", err.Error())
}

// TestStatusMultipleVZ tests the status command
// GIVEN an environment with a two VZ resources
//  WHEN I run the command vz status
//  THEN expect an error of only expecting one VZ
func TestStatusMultipleVZ(t *testing.T) {
	name := "verrazzano"
	namespace1 := "test1"
	namespace2 := "test2"

	vz1 := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace1,
			Name:      name,
		},
	}
	vz2 := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace2,
			Name:      name,
		},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz1, &vz2).Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Expected to only find one Verrazzano resource, but found 2", err.Error())
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
