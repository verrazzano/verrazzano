// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integration

import (
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/helm"
	"github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/event"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"path/filepath"
)

// Reconcile reconciles the Verrazzano CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	log := vzlog.DefaultLogger()

	cm := &corev1.ConfigMap{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, cm); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error, don't requeue
		return result.NewResult()
	}
	ev := event.ConfigMapToEvent(cm)
	return r.applyIntegrationCharts(log, ev)
}

// applyIntegrationCharts applies all the integration charts for components that are enabled
func (r Reconciler) applyIntegrationCharts(log vzlog.VerrazzanoLogger, ev *event.LifecycleEvent) result.Result {
	var retError error

	// Get the chart directories
	chartDirs, err := getChartDirs(config.GetIntegrationChartsDir())
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Do helm install of all integration charts
	for _, chartDir := range chartDirs {
		chartInfo, err := helm.GetChartInfo(chartDir)
		if err != nil {
			log.ErrorfThrottled("Failed to read Chart.yaml for chart %s", chartDir)
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
			ChartPath:    chartDir,
			ChartVersion: chartInfo.Version,
			Overrides:    []helm.HelmOverrides{},
		}
		_, err = helm.UpgradeRelease(log, opts, true, false)
		if err != nil {
			retError = err
		}
		return result.NewResultShortRequeueDelayIfError(err)
	}

	if retError != nil {
		return result.NewResultShortRequeueDelayIfError(retError)
	}
	return result.NewResult()
}

// deleteIntegrationRelease deletes the integration release
func (r Reconciler) deleteIntegrationRelease(log vzlog.VerrazzanoLogger, ev event.LifecycleEvent) result.Result {

	return result.NewResult()
}

// getChartDirs returns the directory names of all the integration changes for files that match a regular expression.
func getChartDirs(rootDirectory string) ([]string, error) {
	chartDirs := []string{}
	walkFunc := func(fileName string, fileInfo os.FileInfo, err error) error {
		if !fileInfo.IsDir() {
			chartDirs = append(chartDirs, fileName)
		}
		return nil
	}

	err := filepath.Walk(rootDirectory, walkFunc)
	if err != nil {
		return nil, err
	}
	return chartDirs, err
}

func getReleaseName(moduleName string) string {
	return fmt.Sprintf("%s-%s", moduleName, "-integration")
}
