// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
)

var dashboardList = []string{
	"dashboards/vmi_dashboard_provider.yml",
	"dashboards/weblogic/weblogic_dashboard.json",
	"dashboards/coherence/elastic-data-summary-dashboard.json",
	"dashboards/coherence/persistence-summary-dashboard.json",
	"dashboards/coherence/cache-details-dashboard.json",
	"dashboards/coherence/members-summary-dashboard.json",
	"dashboards/coherence/kubernetes-summary-dashboard.json",
	"dashboards/coherence/coherence-dashboard-main.json",
	"dashboards/coherence/caches-summary-dashboard.json",
	"dashboards/coherence/service-details-dashboard.json",
	"dashboards/coherence/proxy-servers-summary-dashboard.json",
	"dashboards/coherence/federation-details-dashboard.json",
	"dashboards/coherence/federation-summary-dashboard.json",
	"dashboards/coherence/services-summary-dashboard.json",
	"dashboards/coherence/http-servers-summary-dashboard.json",
	"dashboards/coherence/proxy-server-detail-dashboard.json",
	"dashboards/coherence/alerts-dashboard.json",
	"dashboards/coherence/member-details-dashboard.json",
	"dashboards/coherence/machines-summary-dashboard.json",
	"dashboards/helidon/helidon_dashboard.json",
	"dashboards/system/system_dashboard.json",
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
				return err
			}
			dashboardName := strings.Replace(dashboard, "/", "-", -1)
			dashboards.Data[dashboardName] = string(content)
		}
		return nil
	})
	return err
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
