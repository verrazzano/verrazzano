// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"context"
	"strings"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var dashboardList = []string{
	"manifests/dashboards/vmi_dashboard_provider.yml",
	"manifests/dashboards/weblogic/weblogic_dashboard.json",
	"manifests/dashboards/coherence/elastic-data-summary-dashboard.json",
	"manifests/dashboards/coherence/persistence-summary-dashboard.json",
	"manifests/dashboards/coherence/cache-details-dashboard.json",
	"manifests/dashboards/coherence/members-summary-dashboard.json",
	"manifests/dashboards/coherence/kubernetes-summary-dashboard.json",
	"manifests/dashboards/coherence/coherence-dashboard-main.json",
	"manifests/dashboards/coherence/caches-summary-dashboard.json",
	"manifests/dashboards/coherence/service-details-dashboard.json",
	"manifests/dashboards/coherence/proxy-servers-summary-dashboard.json",
	"manifests/dashboards/coherence/federation-details-dashboard.json",
	"manifests/dashboards/coherence/federation-summary-dashboard.json",
	"manifests/dashboards/coherence/services-summary-dashboard.json",
	"manifests/dashboards/coherence/http-servers-summary-dashboard.json",
	"manifests/dashboards/coherence/proxy-server-detail-dashboard.json",
	"manifests/dashboards/coherence/alerts-dashboard.json",
	"manifests/dashboards/coherence/member-details-dashboard.json",
	"manifests/dashboards/coherence/machines-summary-dashboard.json",
	"manifests/dashboards/helidon/helidon_dashboard.json",
	"manifests/dashboards/system/system_dashboard.json",
	"manifests/dashboards/system/opensearch_dashboard.json",
}

func createGrafanaConfigMaps(ctx spi.ComponentContext) error {
	if !vzconfig.IsGrafanaEnabled(ctx.EffectiveCR()) {
		return nil
	}

	// Create the ConfigMap for Grafana Dashboards
	dashboards := systemDashboardsCM()
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), dashboards, func() error {
		for _, dashboard := range dashboardList {
			content, err := Asset(dashboard)
			if err != nil {
				return ctx.Log().ErrorfNewErr("failed to create grafana configmaps: %v", err)
			}
			dashboards.Data[dashboardName(dashboard)] = string(content)
		}
		return nil
	})
	return err
}

// dashboardName individual dashboards live in the configmap as files of the format:
// 'dashboard-<component>-<dashboard>.json'
func dashboardName(dashboard string) string {
	dashboardNoManifest := strings.Replace(dashboard, "manifests/", "", -1)
	return strings.Replace(dashboardNoManifest, "/", "-", -1)
}

func systemDashboardsCM() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-dashboards",
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string]string{},
	}
}
