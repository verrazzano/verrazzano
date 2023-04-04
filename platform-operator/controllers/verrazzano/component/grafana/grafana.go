// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"fmt"
	"path"

	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/types"
)

const grafanaDeployment = "vmi-system-grafana"

// isGrafanaInstalled checks that the Grafana deployment exists
func isGrafanaInstalled(ctx spi.ComponentContext) bool {
	prefix := newPrefix(ctx.GetComponent())
	deployments := newDeployments()
	return ready.DoDeploymentsExist(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// isGrafanaReady checks that the deployment has the minimum number of replicas available and
// that the admin secret is ready
func isGrafanaReady(ctx spi.ComponentContext) bool {
	var expectedReplicas int32 = 1
	if ctx.EffectiveCR().Spec.Components.Grafana != nil && ctx.EffectiveCR().Spec.Components.Grafana.Replicas != nil {
		if *ctx.EffectiveCR().Spec.Components.Grafana.Replicas < 1 {
			return true
		}
	}
	prefix := newPrefix(ctx.GetComponent())
	deployments := newDeployments()
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, expectedReplicas, prefix) && common.IsGrafanaAdminSecretReady(ctx)
}

// newPrefix creates a component prefix string
func newPrefix(component string) string {
	return fmt.Sprintf("Component %s", component)
}

// creates a slice of NamespacedName with the Grafana deployment name
func newDeployments() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      grafanaDeployment,
			Namespace: ComponentNamespace,
		},
	}
}

// applyDatasourcesConfigmap applies a configmap containing Grafana datasources. If Thanos Query is enabled
// we set Thanos as the default datasource, otherwise Prometheus is the default datasource.
func applyDatasourcesConfigmap(ctx spi.ComponentContext) error {
	// create template key/value map
	args := make(map[string]interface{})
	args["namespace"] = constants.VerrazzanoSystemNamespace
	args["name"] = datasourcesConfigMapName

	cr := ctx.EffectiveCR()
	promEnabled := vzcr.IsPrometheusEnabled(cr) && vzcr.IsPrometheusOperatorEnabled(cr)
	args["isPrometheusEnabled"] = promEnabled
	if promEnabled {
		args["prometheusURL"] = "http://prometheus-operator-kube-p-prometheus.verrazzano-monitoring"
		args["prometheusPort"] = 9090
	}

	thanosQueryEnabled, err := isThanosQueryFrontendEnabled(ctx)
	if err != nil {
		return err
	}
	args["isThanosQueryEnabled"] = thanosQueryEnabled
	if thanosQueryEnabled {
		args["thanosQueryURL"] = "http://thanos-query-frontend.verrazzano-monitoring"
		args["thanosQueryPort"] = 9090
	}

	// substitute template values in the datasources configmap template and apply the resulting YAML
	fpath := path.Join(config.GetThirdPartyManifestsDir(), "grafana", "datasources-configmap.yaml")
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), "")
	err = yamlApplier.ApplyFT(fpath, args)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to substitute template values in Grafana datasources configmap: %v", err)
	}
	return nil
}

// isThanosQueryFrontendEnabled returns true if the Thanos component is enabled and Thanos Query Frontend is
// enabled in the Helm chart
func isThanosQueryFrontendEnabled(ctx spi.ComponentContext) (bool, error) {
	const queryFrontendEnabledHelmKey = "queryFrontend.enabled"

	if !vzcr.IsThanosEnabled(ctx.EffectiveCR()) {
		return false, nil
	}

	thanosComp := thanos.NewComponent().(thanos.ThanosComponent)
	vals, err := thanosComp.GetComputedValues(ctx)
	if err != nil {
		return false, errors.Errorf("Unable to fetch computed Helm values for Thanos: %v", err)
	}
	enabledVal, err := vals.PathValue(queryFrontendEnabledHelmKey)
	if err != nil {
		return false, errors.Errorf("Unable to find Helm key %s in Thanos chart: %v", queryFrontendEnabledHelmKey, err)
	}

	enabled, ok := enabledVal.(bool)
	if !ok {
		return false, errors.Errorf("Thanos chart value %s expected to be of type bool", queryFrontendEnabledHelmKey)
	}

	return enabled, nil
}
