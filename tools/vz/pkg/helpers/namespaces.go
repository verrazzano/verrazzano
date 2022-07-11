package helpers

import (
	encjson "encoding/json"
	"fmt"
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	//files "github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"io/ioutil"
	"os"
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
func GetComponentsNotReady(clusterRoot string) ([]string, error) {
	var compsNotReady = make([]string, 0)
	vzResourcesPath := fmt.Sprintf("%s/%s", clusterRoot, verrazzanoResource) //files.FindFileInClusterRoot(clusterRoot, verrazzanoResource)

	fileInfo, e := os.Stat(vzResourcesPath)
	if e != nil || fileInfo.Size() == 0 {
		// The cluster dump taken by the latest script is expected to contain the verrazzano_resources.json.
		// In order to support cluster dumps taken in earlier release, return nil rather than an error.
		return nil, nil
	}

	file, err := os.Open(vzResourcesPath)
	if err != nil {
		return compsNotReady, err
	}
	defer file.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return compsNotReady, err
	}

	var vzResourceList installv1alpha1.VerrazzanoList
	err = encjson.Unmarshal(fileBytes, &vzResourceList)
	if err != nil {
		return compsNotReady, err
	}

	// There should be only one Verrazzano resource, so the first item from the list should be good enough
	for _, vzRes := range vzResourceList.Items {
		if vzRes.Status.State != installv1alpha1.VzStateReady {

			// Verrazzano installation is not complete, find out the list of components which are not ready
			for _, compStatusDetail := range vzRes.Status.Components {
				if compStatusDetail.State != installv1alpha1.CompStateReady {
					if compStatusDetail.State == installv1alpha1.CompStateDisabled {
						continue
					}
					compsNotReady = append(compsNotReady, compStatusDetail.Name)
				}
			}
			return compsNotReady, nil
		}
	}
	return compsNotReady, nil
}
