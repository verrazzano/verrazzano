package verrazzano

import (
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
)

var componentNameToNamespacesMap = map[string][]string{
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
