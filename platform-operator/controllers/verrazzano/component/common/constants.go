// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import "github.com/verrazzano/verrazzano/pkg/constants"

const (

	// IstioComponentName is the name of the Istio component
	IstioComponentName = "istio"

	// IstioNamespace is the default Istio namespace
	IstioNamespace = constants.IstioSystemNamespace

	// PrometheusOperatorComponentName is the name of the Prometheus Operator component
	PrometheusOperatorComponentName = "prometheus-operator"

	// PrometheusOperatorComponentNamespace is the namespace of the component
	PrometheusOperatorComponentNamespace = constants.VerrazzanoMonitoringNamespace

	OpenSearchComponentName = "opensearch"

	OpenSearchDashboardsComponentName = "opensearch-dashboards"
)
