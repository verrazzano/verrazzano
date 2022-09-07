// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
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

// statusOutputTemplate - template for output of status command
const statusOutputTemplate = `
Verrazzano Status
  Name: {{.verrazzano_name}}
  Namespace: {{.verrazzano_namespace}}
  Version: {{.verrazzano_version}}
  State: {{.verrazzano_state}}
  Profile: {{.install_profile}}
  Access Endpoints:
{{- if .console_url}}
    Console URL: {{.console_url}}
{{- end}}
{{- if .grafana_url}}
    Grafana URL: {{.grafana_url}}
{{- end}}
{{- if .jaeger_url}}
    Jaeger URL: {{.jaeger_url}}
{{- end}}
{{- if .keycloak_url}}
    Keycloak URL: {{.keycloak_url}}
{{- end}}
{{- if .kiali_url}}
    Kiali URL: {{.kiali_url}}
{{- end}}
{{- if .osd_url}}
    Opensearch Dashboards URL: {{.osd_url}}
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
{{- if .components_enabled}}
  Components:
{{- end}}
{{- if .comp_certmanager_state}}
    Cert Manager: {{.comp_certmanager_state}}
{{- end}}
{{- if .comp_coherenceoperator_state}}
    Coherence Operator: {{.comp_coherenceoperator_state}}
{{- end}}
{{- if .comp_externaldns_state}}
    External DNS: {{.comp_externaldns_state}}
{{- end}}
{{- if .comp_grafana_state}}
    Grafana: {{.comp_grafana_state}}
{{- end}}
{{- if .comp_ingresscontroller_state}}
    Ingress Controller: {{.comp_ingresscontroller_state}}
{{- end}}
{{- if .comp_istio_state}}
    Istio: {{.comp_istio_state}}
{{- end}}
{{- if .comp_jaegeroperator_state}}
    Jaeger Operator: {{.comp_jaegeroperator_state}}
{{- end}}
{{- if .comp_keycloak_state}}
    Keycloak: {{.comp_keycloak_state}}
{{- end}}
{{- if .comp_kialiserver_state}}
    Kiali Server: {{.comp_kialiserver_state}}
{{- end}}
{{- if .comp_kubestatemetrics_state}}
    Kube State Metrics: {{.comp_kubestatemetrics_state}}
{{- end}}
{{- if .comp_mysql_state}}
    MySQL: {{.comp_mysql_state}}
{{- end}}
{{- if .comp_oamkubernetesruntime_state}}
    OAM Kubernetes Runtime: {{.comp_oamkubernetesruntime_state}}
{{- end}}
{{- if .comp_opensearch_state}}
    OpenSearch: {{.comp_opensearch_state}}
{{- end}}
{{- if .comp_opensearchdashboards_state}}
    OpenSearch Dashboards: {{.comp_opensearchdashboards_state}}
{{- end}}
{{- if .comp_prometheusadapter_state}}
    Prometheus Adapter: {{.comp_prometheusadapter_state}}
{{- end}}
{{- if .comp_prometheusnodeexporter_state}}
    Prometheus Node Exporter: {{.comp_prometheusnodeexporter_state}}
{{- end}}
{{- if .comp_prometheusoperator_state}}
    Prometheus Operator: {{.comp_prometheusoperator_state}}
{{- end}}
{{- if .comp_prometheuspushgateway_state}}
    Prometheus Pushgateway: {{.comp_prometheuspushgateway_state}}
{{- end}}
{{- if .comp_rancher_state}}
    Rancher: {{.comp_rancher_state}}
{{- end}}
{{- if .comp_verrazzano_state}}
    Verrazzano: {{.comp_verrazzano_state}}
{{- end}}
{{- if .comp_verrazzanoapplicationoperator_state}}
    Verrazzano Application Operator: {{.comp_verrazzanoapplicationoperator_state}}
{{- end}}
{{- if .comp_verrazzanoauthproxy_state}}
    Verrazzano AuthProxy: {{.comp_verrazzanoauthproxy_state}}
{{- end}}
{{- if .comp_verrazzanomonitoringoperator_state}}
    Verrazzano Monitoring Operator: {{.comp_verrazzanomonitoringoperator_state}}
{{- end}}
{{- if .comp_weblogicoperator_state}}
    WebLogic Operator: {{.comp_weblogicoperator_state}}
{{- end}}
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

	// Report the status information
	templateValues := map[string]string{
		"verrazzano_name":      vz.Name,
		"verrazzano_namespace": vz.Namespace,
		"verrazzano_version":   vz.Status.Version,
		"verrazzano_state":     string(vz.Status.State),
	}
	if vz.Spec.Profile == "" {
		templateValues["install_profile"] = string(vzapi.Prod)
	} else {
		templateValues["install_profile"] = string(vz.Spec.Profile)
	}
	addAccessEndpoints(vz.Status.VerrazzanoInstance, templateValues)
	addComponents(vz.Status.Components, templateValues)
	result, err := templates.ApplyTemplate(statusOutputTemplate, templateValues)
	if err != nil {
		return fmt.Errorf("Failed to generate %s command output: %s", CommandName, err.Error())
	}
	fmt.Fprintf(vzHelper.GetOutputStream(), result)

	return nil
}

// addAccessEndpoints - add access endpoints to the display output
func addAccessEndpoints(instance *v1beta1.InstanceInfo, values map[string]string) {
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
		if instance.OpenSearchURL != nil {
			values["os_url"] = *instance.OpenSearchURL
		}
		if instance.OpenSearchURL != nil {
			values["osd_url"] = *instance.OpenSearchDashboardsURL
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
		if instance.JaegerURL != nil {
			values["jaeger_url"] = *instance.JaegerURL
		}
	}
}

// addComponents - add the component status information
func addComponents(components v1beta1.ComponentStatusMap, values map[string]string) {
	if componentOutputEnabled {
		for _, component := range components {
			// Generate key/value for output template - remove dashes from component name, not a valid template character
			stateKey := fmt.Sprintf("comp_%s_state", component.Name)
			values[strings.ReplaceAll(stateKey, "-", "")] = string(component.State)
		}
	}
}
