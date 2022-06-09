// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"context"
	"fmt"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	workloadSourceLabel = "__meta_kubernetes_pod_label_app_verrazzano_io_workload"
)

// GetMetricsTemplate returns the MetricsTemplate given in the MetricsBinding
func GetMetricsTemplate(ctx context.Context, client k8sclient.Client, metricsBinding *vzapi.MetricsBinding, log *zap.SugaredLogger) (*vzapi.MetricsTemplate, error) {
	template := vzapi.MetricsTemplate{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       vzconst.MetricsTemplateKind,
			APIVersion: vzconst.MetricsTemplateAPIVersion,
		},
	}

	templateSpec := metricsBinding.Spec.MetricsTemplate
	namespacedName := types.NamespacedName{Name: templateSpec.Name, Namespace: templateSpec.Namespace}
	err := client.Get(ctx, namespacedName, &template)
	if err != nil {
		newErr := fmt.Errorf("Failed to get the MetricsTemplate %s: %v", templateSpec.Name, err)
		log.Errorf(newErr.Error())
		return nil, newErr
	}

	return &template, nil
}

// HandleDefaultMetricsTemplate handles pre-Verrazzano 1.4 metrics bindings that use the default
// metrics template, by creating/updating a service monitor that does the same work as the default template
func HandleDefaultMetricsTemplate(ctx context.Context, client k8sclient.Client, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	log.Infof("Default metrics template used by metrics binding %s/%s, service monitor time!", metricsBinding.Namespace, metricsBinding.Name)

	// Create the Service monitor from information gathered from the Metrics Binding
	scrapeInfo, err := createScrapeInfo(ctx, client, metricsBinding, log)
	if err != nil {
		return err
	}
	serviceMonitor := promoperapi.ServiceMonitor{}
	serviceMonitor.SetName(metricsBinding.Name)
	serviceMonitor.SetNamespace(metricsBinding.Namespace)
	_, err = controllerutil.CreateOrUpdate(ctx, client, &serviceMonitor, func() error {
		return PopulateServiceMonitor(scrapeInfo, &serviceMonitor, log)
	})
	if err != nil {
		return log.ErrorfNewErr("Failed to create or update the service monitor for the Metrics Binding %s/%s: %v", metricsBinding.Namespace, metricsBinding.Name, err)
	}
	return nil
}

// HandleCustomMetricsTemplate handles pre-Verrazzano 1.4 metrics bindings that use a custom
// metrics template, by updating the additionalScrapeConfigs secret for the Prometheus CR to collect
// metrics as specified by the custom template. TODO should this also handle user-specified configmap?
func HandleCustomMetricsTemplate(ctx context.Context, client k8sclient.Client, metricsBinding *vzapi.MetricsBinding, log *zap.SugaredLogger) error {
	log.Infof("Custom metrics template used by metrics binding %s/%s, edit additionalScrapeConfigs", metricsBinding.Namespace, metricsBinding.Name)
	return nil
}

func createScrapeInfo(ctx context.Context, client k8sclient.Client, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (ScrapeInfo, error) {
	var scrapeInfo ScrapeInfo

	// Get the workload object from the Metrics Binding to populate the Service Monitor
	workload := metricsBinding.Spec.Workload
	workloadObject := unstructured.Unstructured{}
	workloadObject.SetKind(workload.TypeMeta.Kind)
	workloadObject.SetAPIVersion(workload.TypeMeta.APIVersion)
	workloadName := types.NamespacedName{Namespace: metricsBinding.Namespace, Name: workload.Name}
	err := client.Get(ctx, workloadName, &workloadObject)
	if err != nil {
		return scrapeInfo, log.ErrorfNewErr("Failed to get the workload %s from the Metrics Binding %s/%s: %v", workload.Name, metricsBinding.Namespace, metricsBinding.Name, err)
	}

	// Get the namespace for the Metrics Binding to check if Istio is enabled
	workloadNamespace := k8scorev1.Namespace{}
	err = client.Get(context.TODO(), k8sclient.ObjectKey{Name: metricsBinding.GetNamespace()}, &workloadNamespace)
	if err != nil {
		return scrapeInfo, log.ErrorfNewErr("Failed to get metrics binding namespace %s: %v", metricsBinding.GetName(), err)
	}

	// Verify if Istio is enabled from the Namespace annotations
	value, ok := workloadNamespace.Labels[constants.LabelIstioInjection]
	istioEnabled := ok && value == "enabled"
	scrapeInfo.IstioEnabled = &istioEnabled

	// Match the Verrazzano workload application labels that get applied by the Metrics Binding labeler
	value, ok = workloadObject.GetLabels()[constants.MetricsWorkloadLabel]
	if !ok {
		return scrapeInfo, log.ErrorfNewErr("Failed to find the annotation %s on the target workload", constants.MetricsWorkloadLabel)
	}
	scrapeInfo.KeepLabels = map[string]string{workloadSourceLabel: value}

	// Add a port to the Service Monitor endpoints
	scrapeInfo.Ports = 1

	return scrapeInfo, nil
}
