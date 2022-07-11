package helpers

import (
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const verrazzanoResource = "verrazzano_resources.json"

var compMap = map[string][]string{
	"oam-kubernetes-runtime":          {constants.VerrazzanoSystemNamespace},
	"kiali-server":                    {constants.VerrazzanoSystemNamespace},
	"weblogic-operator":               {constants.VerrazzanoSystemNamespace},
	"verrazzano-authproxy":            {constants.VerrazzanoSystemNamespace},
	"istio":                           {constants.IstioSystemNamespace},
	"external-dns":                    {constants2.CertManagerNamespace},
	"verrazzano-application-operator": {constants.VerrazzanoSystemNamespace},
	"coherence-operator":              {constants.VerrazzanoSystemNamespace},
	"ingress-controller":              {constants.IngressNginxNamespace},
	"mysql":                           {constants.KeycloakNamespace},
	"cert-manager":                    {constants2.CertManagerNamespace},
	"rancher":                         {common.CattleSystem}, // TODO vz-6833 add multiple namespaces
	"prometheus-pushgateway":          {constants.VerrazzanoMonitoringNamespace},
	"prometheus-adapter":              {constants.VerrazzanoMonitoringNamespace},
	"kube-state-metrics":              {constants.VerrazzanoMonitoringNamespace},
	"prometheus-node-exporter":        {constants.VerrazzanoMonitoringNamespace},
	"prometheus-operator":             {constants.VerrazzanoMonitoringNamespace},
	"keycloak":                        {constants.KeycloakNamespace},
	"verrazzano-monitoring-operator":  {constants.VerrazzanoSystemNamespace},
	"grafana":                         {constants.VerrazzanoSystemNamespace},
	"jaeger-operator":                 {constants.VerrazzanoMonitoringNamespace},
	"opensearch-dashboards":           {constants.VerrazzanoSystemNamespace},
	"opensearch":                      {constants.VerrazzanoSystemNamespace},
	"velero":                          {constants.VeleroNameSpace},
	"verrazzano-console":              {constants.VerrazzanoSystemNamespace},
	"verrazzano":                      {constants.VerrazzanoSystemNamespace},
	"fluentd":                         {constants.VerrazzanoSystemNamespace},
}

// Read the Verrazzano resource and return the list of components which did not reach Ready state
func GetComponentsNotReady(client client.Client) ([]string, error) {

	var compsNotReady = make([]string, 0)
	// Get the controller runtime client
	vzRes, err := FindVerrazzanoResource(client)
	if err != nil {
		return compsNotReady, err
	}

	if vzRes.Status.State != installv1alpha1.VzStateReady {
		// Verrazzano installation is not complete, find out the list of components which are not ready
		for _, compStatusDetail := range vzRes.Status.Components {
			if compStatusDetail.State != installv1alpha1.CompStateReady {
				continue
			}
			compsNotReady = append(compsNotReady, compStatusDetail.Name)
		}
	}
	return compsNotReady, nil
}

func getNameSpacesByComponent(componentName string) []string {
	value, exists := compMap[componentName]
	if !exists {
		return nil
	}
	return value
}

func GetAllUniqueNameSpacesForFailedComponents(client client.Client) ([]string, error) {
	var nsList []string
	var nsListMap map[string]bool
	allComponents, err := GetComponentsNotReady(client)
	if err != nil {
		return nsList, err
	}

	for _, eachComp := range allComponents {
		for _, eachNameSpace := range getNameSpacesByComponent(eachComp) {
			if !nsListMap[eachNameSpace] {
				nsListMap[eachNameSpace] = true
				nsList = append(nsList, eachNameSpace)
			}
		}
	}
	return nsList, err
}
