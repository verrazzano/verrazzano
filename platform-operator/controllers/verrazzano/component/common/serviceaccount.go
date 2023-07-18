// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
)

func GetAuthProxyPrincipal() string {
	return fmt.Sprintf("cluster.local/ns/%s/sa/verrazzano-authproxy", constants.VerrazzanoSystemNamespace)
}

func GetVMOPrincipal() string {
	return fmt.Sprintf("cluster.local/ns/%s/sa/verrazzano-monitoring-operator", constants.VerrazzanoSystemNamespace)
}

func GetKialiPrincipal() string {
	return fmt.Sprintf("cluster.local/ns/%s/sa/vmi-system-kiali", constants.VerrazzanoSystemNamespace)
}

func GetJaegerPrincipal() string {
	return fmt.Sprintf("cluster.local/ns/%s/sa/jaeger-operator-jaeger", constants.VerrazzanoMonitoringNamespace)
}

func GetThanosQueryPrincipal() string {
	return fmt.Sprintf("cluster.local/ns/%s/sa/thanos-query", constants.VerrazzanoMonitoringNamespace)
}
