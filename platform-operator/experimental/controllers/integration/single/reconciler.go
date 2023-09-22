// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package single

import (
	ctx "context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/helm"
	"github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/event"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"path"
)

// Create ModuleIntegrateAllRequestEvent for these modules
var requireIntegrateAll = map[string]bool{
	certmanager.ComponentName:              true,
	fluentoperator.ComponentName:           true,
	istio.ComponentName:                    true,
	common.PrometheusOperatorComponentName: true,
}

// Reconcile reconciles the IntegrateSingleRequestEvent (in the form of a configmap)
// to perform integration for a single module. Certain modules, such as prometheus-operator,
// require that all integration charts for other modules be installed/upgraded. So in addition
// to applying the chart for a single module, this reconciler may create second event,
// the IntegrateCascadeRequestEvent which processed by the cascade integration controller.
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	log := vzlog.DefaultLogger()

	// Get the configmap and convert into an event
	cm := &corev1.ConfigMap{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, cm); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error, don't requeue
		return result.NewResult()
	}
	ev, err := event.ConfigMapToModuleIntegrationEvent(log, cm)
	if err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Either delete the Helm release or apply the integration chart to install/update the release
	if ev.Action == event.Deleted {
		res := r.deleteIntegrationRelease(log, ev)
		if res.ShouldRequeue() {
			return res
		}
	} else {
		res := r.applyIntegrationChart(log, ev)
		if res.ShouldRequeue() {
			return res
		}
	}

	// If needed, create an IntegrateCascadeRequestEvent using the same payload as the module event that was just processed
	// this is needed to integrate related modules affected by this module
	if ev.Cascade {
		_, ok := requireIntegrateAll[ev.ModuleName]
		if ok {
			res := event.CreateModuleIntegrationCascadeEvent(log, r.Client, ev)
			if res.ShouldRequeue() {
				return res
			}
		}
	}

	// Delete the event.  This is safe to do since the integration controller
	// is the only controller processing IntegrateSingleRequestEvent events
	if err := r.Client.Delete(ctx.TODO(), cm); err != nil {
		log.ErrorfThrottled("Failed to delete event configmap %s", cm.Name)
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// applyIntegrationChart applies the integration chart for the module if chart exists
func (r Reconciler) applyIntegrationChart(log vzlog.VerrazzanoLogger, ev *event.ModuleIntegrationEvent) result.Result {
	// Get the chart directories
	itegrationChartsDir := config.GetIntegrationChartsDir()

	// Nothing to do if an integration chart doesn't exist for this module
	moduleChartDir := path.Join(itegrationChartsDir, ev.ModuleName)
	_, err := os.Stat(moduleChartDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result.NewResult()
		}
		log.ErrorfThrottled("Failed to check if integration chart exists for module %s: %v", ev.ModuleName, err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Get the chart.yaml for this module
	chartInfo, err := helm.GetChartInfo(moduleChartDir)
	if err != nil {
		log.ErrorfThrottled("Failed to read Chart.yaml for chart %s", moduleChartDir)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Perform a Helm install using the helm upgrade --install command
	// Block until helm finishes (wait = true)
	var opts = &helm.HelmReleaseOpts{
		ReleaseName:  getReleaseName(ev.ResourceNSN.Name),
		Namespace:    ev.TargetNamespace,
		ChartPath:    moduleChartDir,
		ChartVersion: chartInfo.Version,
		Overrides:    []helm.HelmOverrides{},
	}
	_, err = helm.UpgradeRelease(log, opts, true, false)
	if err != nil {
		return result.NewResultShortRequeueDelayIfError(err)
	}
	return result.NewResult()
}

// deleteIntegrationRelease deletes the integration release
func (r Reconciler) deleteIntegrationRelease(log vzlog.VerrazzanoLogger, ev *event.ModuleIntegrationEvent) result.Result {
	relName := getReleaseName(ev.ResourceNSN.Name)
	relNamespace := ev.TargetNamespace

	// Check if release is installed
	installed, err := helm.IsReleaseInstalled(relName, relNamespace)
	if err != nil {
		log.ErrorfThrottled("Failed checking if integration Helm release %s is installed: %v", relName, err.Error())
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if !installed {
		return result.NewResult()
	}

	// Uninstall release
	err = helm.Uninstall(log, relName, ev.TargetNamespace, false)
	if err != nil {
		log.ErrorfThrottled("Failed deleting integration Helm release %s: %v", relName, err.Error())
		return result.NewResultShortRequeueDelayWithError(err)
	}

	return result.NewResult()
}

func getReleaseName(moduleName string) string {
	return fmt.Sprintf("%s-%s", moduleName, "integration")
}
