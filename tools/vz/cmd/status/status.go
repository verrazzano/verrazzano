// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/templates"

	"github.com/spf13/cobra"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"k8s.io/apimachinery/pkg/types"
)

const (
	CommandName   = "status"
	namespaceFlag = "namespace"
	nameFlag      = "name"
)

var namespace string
var name string

// statusOutputTemplate - template for output of status command
const statusOutputTemplate = `
Status of Verrazzano {{.verrazzano_name}}
  Version Installed: {{.verrazzano_version}}
  Access Endpoints:
{{- if .console_url}}
    Console URL: {{.console_url}}
{{- end}}
{{- if .grafana_url}}
    Grafana URL: {{.grafana_url}}
{{- end}}
{{- if .keycloak_url}}
    Keycloak URL: {{.keycloak_url}}
{{- end}}
{{- if .kiali_url}}
    Kiali URL: {{.kiali_url}}
{{- end}}
{{- if .kibana_url}}
    Kibana URL: {{.kibana_url}}
{{- end}}
{{- if .os_url}}
    OpenSearch URL: {{.os_url}}
{{- end}}
{{- if .prometheus_url}}
    Prometheus URL: {{.prometheus_url}}
{{- end}}
{{- if .rancher_url}}
    Rancher URL: {{.rancher_url}}
{{- end}}
`

func NewCmdStatus(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := helpers.NewCommand(vzHelper, CommandName, "Status of the Verrazzano install and access endpoints", "Status of the Verrazzano install and access endpoints")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdStatus(cmd, args, vzHelper)
	}

	// Add flags specific to this command and its sub-commands
	cmd.PersistentFlags().StringVarP(&namespace, namespaceFlag, "n", "default", "The namespace of the Verrazzano resource")
	cmd.PersistentFlags().StringVar(&name, nameFlag, "", "The name of the Verrazzano resource")

	return cmd
}

// runCmdStatus - run the "vz status" command
func runCmdStatus(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// Get the VZ resource
	vz := vzapi.Verrazzano{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	if err != nil {
		return fmt.Errorf("Failed to find Verrazzano with name %s in namespace %s: %v", name, namespace, err.Error())
	}

	// Report the status information
	templateValues := map[string]string{
		"verrazzano_name":    vz.Name,
		"verrazzano_version": vz.Status.Version,
	}
	addAccessEndpoints(vz.Status.VerrazzanoInstance, templateValues)
	result, err := templates.ApplyTemplate(statusOutputTemplate, templateValues)
	if err != nil {
		return err
	}
	fmt.Fprintf(vzHelper.GetOutputStream(), result)

	return nil
}

// addAccessEndpoints - add access endpoints to the display output
func addAccessEndpoints(instance *vzapi.InstanceInfo, values map[string]string) {
	if instance != nil {
		if instance.ConsoleURL != nil {
			values["console_url"] = *instance.ConsoleURL
		}
		if instance.KeyCloakURL != nil {
			values["keycloak_url"] = *instance.KeyCloakURL
		}
		if instance.RancherURL != nil {
			values["rancher_url"] = *instance.RancherURL
		}
		if instance.ElasticURL != nil {
			values["os_url"] = *instance.ElasticURL
		}
		if instance.KibanaURL != nil {
			values["kibana_url"] = *instance.KibanaURL
		}
		if instance.GrafanaURL != nil {
			values["grafana_url"] = *instance.GrafanaURL
		}
		if instance.PrometheusURL != nil {
			values["prometheus_url"] = *instance.PrometheusURL
		}
		if instance.KialiURL != nil {
			values["kiali_url"] = *instance.KialiURL
		}
	}
}
