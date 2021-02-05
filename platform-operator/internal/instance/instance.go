// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"strings"

	//clusterPkg "github.com/verrazzano/verrazzano-operator/pkg/api/clusters"
	//"github.com/verrazzano/verrazzano-operator/pkg/util"
	"go.uber.org/zap"
)

// This is global for the operator
//var verrazzanoURI string

// SetVerrazzanoURI set the verrazzanoURI variable
//func SetVerrazzanoURI(s string) {
//	verrazzanoURI = s
//}

// GetInstanceInfo returns a single instance identified by the secret Kubernetes UID.
func GetInstanceInfo(name string, version string, envName string, dnsSuffix string) *v1alpha1.InstanceInfo {
	zap.S().Infow("GET /instance")

	//clusters, err := clusterPkg.GetClusters()
	//if err != nil {
	//	msg := fmt.Sprintf("Error getting clusters : %s", err.Error())
	//	zap.S().Errorw(msg)
	//	http.Error(w, msg, http.StatusInternalServerError)
	//	return
	//}
	//
	//var mgmtCluster clusterPkg.Cluster
	//for _, c := range clusters {
	//	if c.Name == "local" {
	//		mgmtCluster = c
	//		break
	//	}
	//}

	vzURI := getVerrazzanoURI(envName, dnsSuffix)

	return &v1alpha1.InstanceInfo{
		ID:               "0",
		Name:             name,
		//MgmtCluster:      mgmtCluster.Name,
		//MgmtPlatform:     mgmtCluster.Type,
		//Status:           mgmtCluster.Status,
		Version:          version,
		VzAPIURL:         deriveURL(vzURI,"api"),
		RancherURL:       deriveURL(vzURI,"rancher"),
		ElasticURL:       GetElasticURL(vzURI),
		KibanaURL:        GetKibanaURL(vzURI),
		GrafanaURL:       GetGrafanaURL(vzURI),
		PrometheusURL:    GetPrometheusURL(vzURI),
		KeyCloakURL:      GetKeyCloakURL(vzURI),
		//IsUsingSharedVMI: util.SharedVMIDefault(),
	}
}

func getVerrazzanoURI(name string, suffix string) string {
	return fmt.Sprintf("%s.%s", name, suffix)
}

//func getVersion() string {
//	return "0.1.0"
//}

// Derive the URL from the verrazzano URI by prefixing with the given URL segment
func deriveURL(verrazzanoURI string, component string) string {
	if len(strings.TrimSpace(verrazzanoURI)) > 0 {
		return "https://" + component + "." + verrazzanoURI
	}
	return ""
}

// GetVerrazzanoName returns the environment name portion of the verrazzanoUri
//func GetVerrazzanoName() string {
//	segs := strings.Split(verrazzanoURI, ".")
//	if len(segs) > 1 {
//		return segs[0]
//	}
//	return ""
//}

// GetKeyCloakURL returns Keycloak URL
func GetKeyCloakURL(verrazzanoURI string) string {
	return deriveURL(verrazzanoURI,"keycloak")
}

// GetKibanaURL returns Kibana URL
func GetKibanaURL(verrazzanoURI string) string {
	return deriveURL(verrazzanoURI,"kibana.vmi.system")
}

// GetGrafanaURL returns Grafana URL
func GetGrafanaURL(verrazzanoURI string) string {
	return deriveURL(verrazzanoURI,"grafana.vmi.system")
}

// GetPrometheusURL returns Prometheus URL
func GetPrometheusURL(verrazzanoURI string) string {
	return deriveURL(verrazzanoURI,"prometheus.vmi.system")
}

// GetElasticURL returns Elasticsearch URL
func GetElasticURL(verrazzanoURI string) string {
	return deriveURL(verrazzanoURI,"elasticsearch.vmi.system")
}

// GetConsoleURL returns the Verrazzano Console URL
func GetConsoleURL(verrazzanoURI string) string {
	return deriveURL(verrazzanoURI,"console")
}
