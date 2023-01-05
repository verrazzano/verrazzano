// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/templates"
)

const (
	CommandName = "status"
	helpShort   = "Status of the Verrazzano installation and access endpoints"
	helpLong    = `The command 'status' returns summary information about a Verrazzano installation`
	helpExample = `
vz status
vz status --context minikube
vz status --kubeconfig ~/.kube/config --context minikube`
)

// The component output is disabled pending the resolution some issues with
// the content of the Verrazzano status block
var componentOutputEnabled = false

type TemplateInput struct {
	Endpoints         map[string]string
	Components        map[string]string
	ComponentsEnabled bool

	Name                string
	Namespace           string
	Version             string
	State               string
	Profile             string
	AvailableComponents string
}

// statusOutputTemplate - template for output of status command
const statusOutputTemplate = `
Verrazzano Status
  Name: {{.Name}}
  Namespace: {{.Namespace}}
  Profile: {{.Profile}}
  Version: {{.Version}}
  State: {{.State}}
{{- if .AvailableComponents }}
  Available Components: {{.AvailableComponents}}
{{- end }}
  Profile: {{.Profile}}
  Access Endpoints:
{{- range $key, $value := .Endpoints }}
    {{ $key }}: {{ $value }}
{{- end }}
{{- if .ComponentsEnabled }}
  Components:
{{- range $key, $value := .Components }}
    {{ $key }}: {{ $value }}
{{- end }}
{{- end }}
`

func NewCmdStatus(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdStatus(cmd, vzHelper)
	}
	cmd.Example = helpExample

	return cmd
}

// runCmdStatus - run the "vz status" command
func runCmdStatus(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// Get the VZ resource
	vz, err := helpers.FindVerrazzanoResource(client)
	if err != nil {
		return err
	}
	templateValues := TemplateInput{
		Endpoints:           getEndpoints(vz.Status.VerrazzanoInstance),
		Components:          getComponents(vz.Status.Components),
		Name:                vz.Name,
		Namespace:           vz.Namespace,
		Version:             vz.Status.Version,
		State:               string(vz.Status.State),
		AvailableComponents: getAvailableComponents(vz.Status.Available),
		Profile:             getProfile(vz.Spec.Profile),
	}
	result, err := templates.ApplyTemplate(statusOutputTemplate, templateValues)
	if err != nil {
		return fmt.Errorf("Failed to generate %s command output: %s", CommandName, err.Error())
	}
	fmt.Fprintf(vzHelper.GetOutputStream(), result)

	return nil
}

func getAvailableComponents(available *string) string {
	if available == nil {
		return ""
	}
	return *available
}

func getProfile(profile v1beta1.ProfileType) string {
	if profile == "" {
		return string(vzapi.Prod)
	}
	return string(profile)
}

func getEndpoints(instance *v1beta1.InstanceInfo) map[string]string {
	values := map[string]string{}
	instanceValue := reflect.Indirect(reflect.ValueOf(instance))
	instanceType := reflect.TypeOf(instance).Elem()
	for i := 0; i < instanceType.NumField(); i++ {
		fieldValue := instanceValue.Field(i)
		v, ok := fieldValue.Interface().(*string)
		if ok && v != nil {
			k := getJSONTagNameForField(instanceType.Field(i))
			values[k] = *v
		}
	}
	return values
}

func getJSONTagNameForField(field reflect.StructField) string {
	return strings.Split(field.Tag.Get("json"), ",")[0]
}

// addComponents - add the component status information
func getComponents(components v1beta1.ComponentStatusMap) map[string]string {
	values := map[string]string{}
	if componentOutputEnabled {
		for _, component := range components {
			ok, c := registry.FindComponent(component.Name)
			if !ok {
				continue
			}
			values[c.GetJSONName()] = string(component.State)
		}
	}
	return values
}
