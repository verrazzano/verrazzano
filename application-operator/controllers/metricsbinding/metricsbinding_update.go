// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vztemplate "github.com/verrazzano/verrazzano/application-operator/controllers/template"
	"github.com/verrazzano/verrazzano/application-operator/internal/metrics"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/metricsutils"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	k8scontroller "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

// reconcileBindingCreateOrUpdate completes the reconcile process for an object that is being created or updated
func (r *Reconciler) reconcileBindingCreateOrUpdate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (k8scontroller.Result, error) {
	log.Debugw("Reconcile for created or updated object", "resource", metricsBinding.GetName())

	// Requeue with a delay to account for situations where the scrape config
	// has changed but without the MetricsBinding changing.
	var seconds = rand.IntnRange(45, 90)
	var requeueDuration = time.Duration(seconds) * time.Second

	// Handle the case where the workload uses the default metrics template - in this case, we will
	// delete the metrics binding if processing succeeds, since this is a one-time conversion of
	// legacy apps using default metrics template, to ServiceMonitor. If it's not using VMI config map,
	// we treat it like custom metrics setup
	if isLegacyDefaultMetricsBinding(metricsBinding) {
		log.Debug("Legacy default MetricsBinding found, creating a Service Monitor and deleting the MetricsBinding")
		if err := r.handleDefaultMetricsTemplate(ctx, metricsBinding, log); err != nil {
			return k8scontroller.Result{Requeue: true}, err
		}
		log.Infof("Deleting legacy default MetricsBinding %s/%s", metricsBinding.Namespace, metricsBinding.Name)
		if err := r.deleteMetricsBinding(metricsBinding, log); err != nil {
			return k8scontroller.Result{Requeue: true}, err
		}
		// Requeue with a delay to account for situations where the scrape config
		// has changed but without the MetricsBinding changing.
		return reconcile.Result{Requeue: true, RequeueAfter: requeueDuration}, nil
	}

	// Update the MetricsBinding to add workload as owner ref
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, metricsBinding, func() error {
		return r.updateMetricsBinding(metricsBinding, log)
	})
	if err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Handle the case where the workload uses a custom metrics template
	if err = r.handleCustomMetricsTemplate(ctx, metricsBinding, log); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Requeue with a delay to account for situations where the scrape config
	// has changed but without the MetricsBinding changing.
	return reconcile.Result{Requeue: true, RequeueAfter: requeueDuration}, nil
}

// handleDefaultMetricsTemplate handles pre-Verrazzano 1.4 metrics bindings that use the default
// metrics template, by creating/updating a service monitor that does the same work as the default template
func (r *Reconciler) handleDefaultMetricsTemplate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	log.Infof("Default metrics template used by metrics binding %s/%s, creating service monitor", metricsBinding.Namespace, metricsBinding.Name)

	// Create the Service monitor from information gathered from the Metrics Binding
	scrapeInfo, err := r.createScrapeInfo(ctx, metricsBinding, log)
	if err != nil {
		return err
	}
	serviceMonitor := promoperapi.ServiceMonitor{}
	serviceMonitor.SetName(metricsBinding.Name)
	serviceMonitor.SetNamespace(metricsBinding.Namespace)
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &serviceMonitor, func() error {
		return metrics.PopulateServiceMonitor(scrapeInfo, &serviceMonitor, log)
	})
	if err != nil {
		return log.ErrorfNewErr("Failed to create or update the service monitor for the Metrics Binding %s/%s: %v", metricsBinding.Namespace, metricsBinding.Name, err)
	}
	return nil
}

// handleCustomMetricsTemplate handles pre-Verrazzano 1.4 metrics bindings that use a custom
// metrics template, by updating the additionalScrapeConfigs secret for the Prometheus CR to collect
// metrics as specified by the custom template.
func (r *Reconciler) handleCustomMetricsTemplate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	log.Debugf("Custom metrics template used by metrics binding %s/%s, edit additionalScrapeConfigs", metricsBinding.Namespace, metricsBinding.Name)

	var workloadNamespaceUnstructured *unstructured.Unstructured
	var err error
	// Get the Namespace of the Metrics Binding as an unstructured resource so that it can be applied
	// to the template
	if workloadNamespaceUnstructured, err = r.createWorkloadNamespaceUnstructured(metricsBinding, log); err != nil {
		return err
	}

	// Get the workload object, so that it can be applied to the template
	var workloadObject *unstructured.Unstructured
	if workloadObject, err = r.getWorkloadObject(metricsBinding); err != nil {
		return log.ErrorfNewErr("Failed to get the workload object for metrics binding %s: %v", metricsBinding.GetName(), err)
	}

	createdJobName := createJobName(metricsBinding)
	scrapeConfigString, err := r.createScrapeConfigForMetricsBinding(metricsBinding, workloadObject, workloadNamespaceUnstructured, createdJobName, log)
	if err != nil {
		return err
	}

	// Format scrape config into readable container
	var configYaml []byte
	if configYaml, err = yaml.YAMLToJSON([]byte(scrapeConfigString)); err != nil {
		return log.ErrorfNewErr("Failed to convert scrape config YAML to JSON: %v", err)
	}
	g
	var newScrapeConfig *gabs.Container
	if newScrapeConfig, err = gabs.ParseJSON(configYaml); err != nil {
		return log.ErrorfNewErr("Failed to convert scrape config JSON to container: %v", err)
	}

	// Collect the data from the ConfigMap or the Secret
	configMapExists, err := r.updateScrapeConfigInConfigMap(ctx, metricsBinding, createdJobName, newScrapeConfig, log)
	if !configMapExists {
		_, err = r.updateScrapeConfigInConfigSecret(ctx, metricsBinding, createdJobName, newScrapeConfig, log)
	}

	return err
}

// updateScrapeConfigInConfigSecret updates the scrape config in the PrometheusConfigSecret if one
// is specified in the metrics binding. Returns true if there is a config secret, and any error that occurred
func (r *Reconciler) updateScrapeConfigInConfigSecret(ctx context.Context, metricsBinding *vzapi.MetricsBinding,
	createdJobName string, newScrapeConfig *gabs.Container, log vzlog.VerrazzanoLogger) (bool, error) {
	secret, key := getPromConfigSecret(metricsBinding)
	if secret == nil {
		return false, nil
	}
	log.Debugf("Secret %s/%s found in the MetricsBinding, attempting scrape config update", secret.GetNamespace(), secret.GetName())
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		var err error
		var data *gabs.Container
		if data, err = getConfigDataFromSecret(secret, key); err != nil {
			return log.ErrorfNewErr("Failed to get the Secret data: %v", err)
		}
		var promConfig *gabs.Container
		if promConfig, err = metricsutils.EditScrapeJob(data, createdJobName, newScrapeConfig); err != nil {
			return log.ErrorfNewErr("Failed to edit the scrape job: %v", err)
		}
		var newPromConfigData []byte
		if newPromConfigData, err = yaml.JSONToYAML(promConfig.Bytes()); err != nil {
			return log.ErrorfNewErr("Failed to convert scrape config JSON to YAML: %v", err)
		}
		secret.Data[key] = newPromConfigData
		return nil
	})
	return true, err
}

// updateScrapeConfigInConfigMap updates the scrape config in the Prometheus ConfigMap if one
// is specified in the metrics binding. Returns true if there is a config map, and any error that occurred
func (r *Reconciler) updateScrapeConfigInConfigMap(ctx context.Context,
	metricsBinding *vzapi.MetricsBinding, jobName string, newScrapeConfig *gabs.Container, log vzlog.VerrazzanoLogger) (bool, error) {
	var data *gabs.Container
	configMap := getPromConfigMap(metricsBinding)
	if configMap == nil {
		return false, nil
	}
	log.Debugf("ConfigMap %s/%s found in the MetricsBinding, attempting scrape config update", configMap.GetNamespace(), configMap.GetName())
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		var err error
		if data, err = getConfigData(configMap); err != nil {
			return log.ErrorfNewErr("Failed to get the ConfigMap data: %v", err)
		}
		if err = metricsutils.EditScrapeJobInPrometheusConfig(data, prometheusScrapeConfigsLabel, jobName, newScrapeConfig); err != nil {
			return log.ErrorfNewErr("Failed to edit the scrape job: %v", err)
		}
		var newPromConfigData []byte
		if newPromConfigData, err = yaml.JSONToYAML(data.Bytes()); err != nil {
			return log.ErrorfNewErr("Failed to convert scrape config JSON to YAML: %v", err)
		}
		configMap.Data[prometheusConfigKey] = string(newPromConfigData)
		return nil
	})
	return true, err
}

func (r *Reconciler) createWorkloadNamespaceUnstructured(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (*unstructured.Unstructured, error) {
	workloadNamespace := k8scorev1.Namespace{}
	log.Debugf("Getting the workload namespace %s from the MetricsBinding", metricsBinding.GetNamespace())
	err := r.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: metricsBinding.GetNamespace()}, &workloadNamespace)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to get metrics binding namespace %s: %v", metricsBinding.GetName(), err)
	}

	// Create an unstructured resource from the Namespace, so it can be applied to the template
	workloadNamespaceUnstructuredMap, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(&workloadNamespace)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to get the unstructured for namespace %s: %v", workloadNamespace.GetName(), err)
	}
	return &unstructured.Unstructured{Object: workloadNamespaceUnstructuredMap}, nil
}

func (r *Reconciler) createScrapeInfo(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (metrics.ScrapeInfo, error) {
	log.Debugf("Attempting to create the ServiceMonitor information from the MetricsBinding %s/%s", metricsBinding.Namespace, metricsBinding.Name)
	var scrapeInfo metrics.ScrapeInfo

	// Get the workload object from the Metrics Binding to populate the Service Monitor
	workload := metricsBinding.Spec.Workload
	workloadObject := unstructured.Unstructured{}
	workloadObject.SetKind(workload.TypeMeta.Kind)
	workloadObject.SetAPIVersion(workload.TypeMeta.APIVersion)
	workloadName := types.NamespacedName{Namespace: metricsBinding.Namespace, Name: workload.Name}
	log.Debugf("Getting the workload resource %s/%s from the MetricsBinding", workloadName.Namespace, workloadName.Name)
	err := r.Client.Get(ctx, workloadName, &workloadObject)
	if err != nil {
		return scrapeInfo, log.ErrorfNewErr("Failed to get the workload %s from the MetricsBinding %s/%s: %v", workload.Name, metricsBinding.Namespace, metricsBinding.Name, err)
	}

	// Get the namespace for the Metrics Binding to check if Istio is enabled
	workloadNamespace := k8scorev1.Namespace{}
	log.Debugf("Getting the workload namespace %s from the MetricsBinding", metricsBinding.GetNamespace())
	err = r.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: metricsBinding.GetNamespace()}, &workloadNamespace)
	if err != nil {
		return scrapeInfo, log.ErrorfNewErr("Failed to get MetricsBinding namespace %s: %v", metricsBinding.GetName(), err)
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

	// Add the cluster name to the scrape info
	scrapeInfo.ClusterName = clusters.GetClusterName(ctx, r.Client)

	return scrapeInfo, nil
}

// updateMetricsBinding updates the Metrics Binding Owner Reference from the target workload,
// adds a finalizer, and updates the PrometheusConfigSecret field if the metrics binding was using
// the legacy default prometheus config map
func (r *Reconciler) updateMetricsBinding(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	// Add the finalizer
	controllerutil.AddFinalizer(metricsBinding, finalizerName)

	// Retrieve the workload object from the MetricsBinding
	workloadObject, err := r.getWorkloadObject(metricsBinding)
	if err != nil {
		return log.ErrorfNewErr("Failed to get the Workload from the MetricsBinding %s: %v", metricsBinding.Spec.Workload.Name, err)
	}

	// Return error if UID is not found
	if len(workloadObject.GetUID()) == 0 {
		err = fmt.Errorf("could not get UID from workload resource: %s, %s", workloadObject.GetKind(), workloadObject.GetName())
		return log.ErrorfNewErr("Failed to find UID for workload %s: %v", workloadObject.GetName(), err)
	}

	// Set the owner reference for the MetricsBinding so that it gets deleted with the workload
	log.Debugf("Updating the MetricsBinding OwnerReference to the target workload %s/%s", workloadObject.GetNamespace(), workloadObject.GetName())
	trueValue := true
	metricsBinding.SetOwnerReferences([]k8smetav1.OwnerReference{
		{
			Name:               workloadObject.GetName(),
			APIVersion:         workloadObject.GetAPIVersion(),
			Kind:               workloadObject.GetKind(),
			UID:                workloadObject.GetUID(),
			Controller:         &trueValue,
			BlockOwnerDeletion: &trueValue,
		},
	})

	// If the config map specified is the legacy VMI prometheus config map, modify it to use
	// the additionalScrapeConfigs config map for the Prometheus Operator
	if isLegacyVmiPrometheusConfigMapName(metricsBinding.Spec.PrometheusConfigMap) {
		log.Infof("Metrics Binding %s/%s uses legacy VMI prometheus config map - updating to use the Prometheus operator secret %s/%s",
			metricsBinding.Namespace, metricsBinding.Name, vzconst.PrometheusOperatorNamespace, vzconst.PromAdditionalScrapeConfigsSecretName)
		metricsBinding.Spec.PrometheusConfigMap = vzapi.NamespaceName{}
		metricsBinding.Spec.PrometheusConfigSecret = vzapi.SecretKey{
			Namespace: vzconst.PrometheusOperatorNamespace,
			Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
			Key:       vzconst.PromAdditionalScrapeConfigsSecretKey,
		}
	}

	return nil
}

// getMetricsTemplate returns the MetricsTemplate given in the MetricsBinding
func (r *Reconciler) getMetricsTemplate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (*vzapi.MetricsTemplate, error) {
	template := vzapi.MetricsTemplate{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       vzconst.MetricsTemplateKind,
			APIVersion: vzconst.MetricsTemplateAPIVersion,
		},
	}

	templateSpec := metricsBinding.Spec.MetricsTemplate
	namespacedName := types.NamespacedName{Name: templateSpec.Name, Namespace: templateSpec.Namespace}
	err := r.Client.Get(ctx, namespacedName, &template)
	if err != nil {
		newErr := fmt.Errorf("Failed to get the MetricsTemplate %s: %v", templateSpec.Name, err)
		return nil, log.ErrorfNewErr(newErr.Error())
	}
	return &template, nil
}

// getWorkloadObject returns the workload object based on the definition in the MetricsBinding
func (r *Reconciler) getWorkloadObject(metricsBinding *vzapi.MetricsBinding) (*unstructured.Unstructured, error) {
	// Retrieve the owner from the workload field of the MetricsBinding
	owner := metricsBinding.Spec.Workload
	workloadObject := unstructured.Unstructured{}
	workloadObject.SetKind(owner.TypeMeta.Kind)
	workloadObject.SetAPIVersion(owner.TypeMeta.APIVersion)
	workloadName := types.NamespacedName{Namespace: metricsBinding.GetNamespace(), Name: owner.Name}
	err := r.Client.Get(context.Background(), workloadName, &workloadObject)
	if err != nil {
		return nil, err
	}
	return &workloadObject, nil
}

// deleteMetricsBinding deletes the Metrics Binding object from the cluster
func (r *Reconciler) deleteMetricsBinding(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	// Remove the finalizer from the metrics binding
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, metricsBinding, func() error {
		controllerutil.RemoveFinalizer(metricsBinding, finalizerName)
		return nil
	})
	if err != nil {
		return log.ErrorfNewErr("Failed to remove the finalizer from the Metrics Binding %s/%s: %s", metricsBinding.Namespace, metricsBinding.Name, err)
	}

	// Delete the binding once the finalizer has been removed
	err = r.Delete(context.Background(), metricsBinding)
	if err != nil {
		return log.ErrorfNewErr("Failed to delete the Metrics Binding %s/%s from the cluster: %v", metricsBinding.Namespace, metricsBinding.Name, err)
	}
	return err
}

func (r *Reconciler) createScrapeConfigForMetricsBinding(
	metricsBinding *vzapi.MetricsBinding, workloadObject *unstructured.Unstructured,
	workloadNamespaceUnstructured *unstructured.Unstructured, jobName string, log vzlog.VerrazzanoLogger) (string, error) {
	// Get the Metrics Template from the Metrics Binding
	template, err := r.getMetricsTemplate(context.Background(), metricsBinding, log)
	if err != nil {
		return "", err
	}

	// Organize inputs for template processor
	log.Debugf("Creating the template inputs from the workload %s and namespace %s", workloadObject.GetName(), metricsBinding.GetNamespace())
	templateInputs := map[string]interface{}{
		"workload":  workloadObject.Object,
		"namespace": workloadNamespaceUnstructured.Object,
	}

	// Get scrape config from the template processor and process the template inputs
	templateProcessor := vztemplate.NewProcessor(r.Client, template.Spec.PrometheusConfig.ScrapeConfigTemplate)
	scrapeConfigString, err := templateProcessor.Process(templateInputs)
	if err != nil {
		return "", log.ErrorfNewErr("Failed to process metrics template %s: %v", template.GetName(), err)
	}

	// Prepend job name to the scrape config
	scrapeConfigString = formatJobName(jobName) + scrapeConfigString
	return scrapeConfigString, nil
}
