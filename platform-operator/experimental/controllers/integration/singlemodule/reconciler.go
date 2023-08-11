// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package singlemodule

import (
	ctx "context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/helm"
	"github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/certmanager"
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
	certmanager.ComponentName:    true,
	fluentoperator.ComponentName: true,
	istio.ComponentName:          true,
}

// Reconcile reconciles the Verrazzano CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	log := vzlog.DefaultLogger()

	cm := &corev1.ConfigMap{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, cm); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error, don't requeue
		return result.NewResult()
	}
	ev := event.ConfigMapToModuleIntegrationEvent(cm)
	res := r.applyIntegrationCharts(log, ev)
	if res.ShouldRequeue() {
		return res
	}

	// If needed, create an IntegrateOthersRequestEvent using the same payload as the module event that was just processed
	// this is needed to integrate related modules affected by this module
	if ev.Cascade {
		_, ok := requireIntegrateAll[ev.ModuleName]
		if ok {
			ev.EventType = event.IntegrateOthersRequestEvent
			ev.Cascade = false
			res := event.CreateEvent(r.Client, ev)
			if res.ShouldRequeue() {
				return res
			}
		}
	}

	// Delete the event.  This is safe to do since the integration controller
	// is the only controller processing IntegrateSingleRequestEvent events
	if err := r.Client.Delete(ctx.TODO(), cm); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// applyIntegrationCharts applies all the integration charts for components that are enabled
func (r Reconciler) applyIntegrationCharts(log vzlog.VerrazzanoLogger, ev *event.ModuleIntegrationEvent) result.Result {
	var retError error

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
	if err != nil {
		return result.NewResult()
	}
	var opts = &helm.HelmReleaseOpts{
		ReleaseName:  getReleaseName(ev.ResourceNSN.Name),
		Namespace:    ev.TargetNamespace,
		ChartPath:    moduleChartDir,
		ChartVersion: chartInfo.Version,
		Overrides:    []helm.HelmOverrides{},
	}
	_, err = helm.UpgradeRelease(log, opts, true, false)
	if err != nil {
		return result.NewResultShortRequeueDelayIfError(retError)
	}
	return result.NewResult()
}

// deleteIntegrationRelease deletes the integration release
func (r Reconciler) deleteIntegrationRelease(log vzlog.VerrazzanoLogger, ev event.ModuleIntegrationEvent) result.Result {
	return result.NewResult()
}

func getReleaseName(moduleName string) string {
	return fmt.Sprintf("%s-%s", moduleName, "integration")
}
