// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package servicemonitor

import (
	"context"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	operator "github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	metrics "github.com/verrazzano/verrazzano/pkg/metrics"
	utils "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/controllers"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateServiceMonitor(conversionComponent *types.ConversionComponents) (*promoperapi.ServiceMonitor, error) {
	var log vzlog.VerrazzanoLogger
	var traitDefaults *vzapi.MetricsTraitSpec
	cfg, _ := config.GetConfig()
	cli, _ := client.New(cfg, client.Options{})
	trait := conversionComponent.MetricsTrait

	//TODO:Fix namespace with servicemonitor name and if trait uses Istio

	// Creating a service monitor with name and namespace
	serviceMonitor := promoperapi.ServiceMonitor{}
	pmName, err := utils.CreateServiceMonitorName(trait, conversionComponent.AppName, conversionComponent.ComponentName, 0)
	if err != nil {
		return &serviceMonitor, err
	}
	// Fetch workload resource as well as trait defaults using information from the trait
	var workload *unstructured.Unstructured

	if conversionComponent.Helidonworkload != nil {
		workload = conversionComponent.Helidonworkload
		traitDefaults, err = utils.NewTraitDefaultsForGenericWorkload()
		if err != nil {
			return &serviceMonitor, err
		}
	} else if conversionComponent.Coherenceworkload != nil {
		workload = conversionComponent.Coherenceworkload
		traitDefaults, err = utils.NewTraitDefaultsForCOHWorkload(workload)
		if err != nil {
			return &serviceMonitor, err
		}
	} else if conversionComponent.Weblogicworkload != nil {
		workload = conversionComponent.Weblogicworkload
		traitDefaults, err = utils.NewTraitDefaultsForWLSDomainWorkload(workload)
		if err != nil {
			return &serviceMonitor, err
		}
	} else {
		workload = conversionComponent.Genericworkload
		traitDefaults, err = utils.NewTraitDefaultsForGenericWorkload()
		if err != nil {
			return &serviceMonitor, err
		}
	}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := operator.FetchSourceCredentialsSecretIfRequired(context.TODO(), trait, traitDefaults, workload, cli)
	if err != nil {
		return &serviceMonitor, err
	}

	//fetch if trait uses Istio
	useHTTPS := types.InputArgs.IstioEnabled

	//fetch if workload is WebLogic
	wlsWorkload, err := operator.IsWLSWorkload(workload)
	if err != nil {
		return &serviceMonitor, err
	}

	vzPromLabels := !wlsWorkload

	//populate servicemonitor scrape info
	scrapeInfo := metrics.ScrapeInfo{
		Ports:              len(operator.GetPortSpecs(trait, traitDefaults)),
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
