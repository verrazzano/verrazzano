// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package servicemonitor

import (
	"context"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	metrics "github.com/verrazzano/verrazzano/pkg/metrics"
	utils "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/controllers"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	prometheusClusterNameLabel = "verrazzano_cluster"
)

func CreateServiceMonitor(conversionComponent *types.ConversionComponents) (*promoperapi.ServiceMonitor,error) {
	var ctx context.Context
	var log vzlog.VerrazzanoLogger
	var cli client.Client
	trait := conversionComponent.MetricsTrait

	serviceMonitor := promoperapi.ServiceMonitor{}

	// Creating a service monitor with name and namespace
	pmName, err := utils.CreateServiceMonitorName(trait, conversionComponent.AppName, conversionComponent.ComponentName, 0)
	if err != nil {
		return &serviceMonitor, err
	}

	// Fetch workload resource using information from the trait
	var workload *unstructured.Unstructured

	if conversionComponent.Helidonworkload != nil {
		workload = conversionComponent.Helidonworkload
	}
	if conversionComponent.Coherenceworkload != nil {
		workload = conversionComponent.Coherenceworkload
	}
	if conversionComponent.Weblogicworkload != nil {
		workload = conversionComponent.Weblogicworkload
	}

	//fetch trait defaultss
	traitDefaults, supported, err := utils.FetchTraitDefaults(workload)
	if err != nil {
		return &serviceMonitor, err
	}
	if !supported || traitDefaults == nil {
		return &serviceMonitor, err
	}
	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := utils.FetchSourceCredentialsSecretIfRequired(ctx, trait, traitDefaults, workload, cli)
	if err != nil {
		return &serviceMonitor, err
	}

	//fetch if trait uses Istio
	useHTTPS, err := utils.UseHTTPSForScrapeTarget(trait)
	if err != nil {
		return &serviceMonitor, err
	}
	//fetch if workload is WebLogic
	wlsWorkload, err := utils.IsWLSWorkload(workload)
	if err != nil {
		return &serviceMonitor, err
	}

	vzPromLabels := !wlsWorkload

	//populate servicemonitor scrape info
	scrapeInfo := metrics.ScrapeInfo{
		Ports:              len(utils.GetPortSpecs(trait, traitDefaults)),
		BasicAuthSecret:    secret,
		IstioEnabled:       &useHTTPS,
		VZPrometheusLabels: &vzPromLabels,
		ClusterName:        "default",
	}

	// Fill in the scrape info if it is populated in the trait
	if trait.Spec.Path != nil {
		scrapeInfo.Path = trait.Spec.Path
	}

	// Populate the keep labels to match the oam pod labels
	scrapeInfo.KeepLabels = map[string]string{
		"__meta_kubernetes_pod_label_app_oam_dev_name":      trait.Labels[oam.LabelAppName],
		"__meta_kubernetes_pod_label_app_oam_dev_component": trait.Labels[oam.LabelAppComponent],
	}

	serviceMonitor.APIVersion = "monitoring.coreos.com/v1"
	serviceMonitor.Kind = "ServiceMonitor"
	serviceMonitor.SetName(pmName)
	serviceMonitor.SetNamespace(workload.GetNamespace())
	metrics.PopulateServiceMonitor(scrapeInfo, &serviceMonitor, log)
	return &serviceMonitor, nil

}
