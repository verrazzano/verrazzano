// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package servicemonitor

import (
	"fmt"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	metrics "github.com/verrazzano/verrazzano/pkg/metrics"
	extract "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/traits"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	prometheusClusterNameLabel = "verrazzano_cluster"
)

func CreateServiceMonitor(conversionComponent *types.ConversionComponents) (err error) {
	var log vzlog.VerrazzanoLogger
	trait := conversionComponent.MetricsTrait

	serviceMonitor := promoperapi.ServiceMonitor{}

	// Creating a service monitor with name and namespace
	pmName, err := createServiceMonitorName(trait, 0)
	if err != nil {
		return err
	}

	// Fetch workload resource using information from the trait
	var workload *unstructured.Unstructured
	workload, err = extract.FetchWorkloadFromTrait(trait)
	if err != nil {
		return err
	}

	//fetch trait defaults
	traitDefaults, supported, err := extract.FetchTraitDefaults(workload)
	if err != nil {
		return err
	}
	if !supported || traitDefaults == nil {
		return err
	}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := extract.FetchSourceCredentialsSecretIfRequired(trait, traitDefaults, workload)
	if err != nil {
		return err
	}

	//fetch if trait uses Istio
	useHTTPS, err := extract.UseHTTPSForScrapeTarget(trait)
	if err != nil {
		return err
	}
	//fetch if workload is WebLogic
	wlsWorkload, err := extract.IsWLSWorkload(workload)
	if err != nil {
		return err
	}

	vzPromLabels := !wlsWorkload

	//populate servicemonitor scrape info
	scrapeInfo := metrics.ScrapeInfo{
		Ports:              len(getPortSpecs(trait, traitDefaults)),
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

	serviceMonitor.SetName(pmName)
	serviceMonitor.SetNamespace(workload.GetNamespace())
	metrics.PopulateServiceMonitor(scrapeInfo, &serviceMonitor, log)
	PrintServiceMonitor(&serviceMonitor)
	return nil

}
func createServiceMonitorName(trait *vzapi.MetricsTrait, portNum int) (string, error) {
	sname, err := createJobOrServiceMonitorName(trait, portNum)
	if err != nil {
		return "", err
	}
	return strings.Replace(sname, "_", "-", -1), nil
}
func createJobOrServiceMonitorName(trait *vzapi.MetricsTrait, portNum int) (string, error) {
	namespace := getNamespaceFromObjectMetaOrDefault(trait.ObjectMeta)
	app, found := trait.Labels[oam.LabelAppName]
	if !found {
		return "", fmt.Errorf("metrics trait missing application name label")
	}
	comp, found := trait.Labels[oam.LabelAppComponent]
	if !found {
		return "", fmt.Errorf("metrics trait missing component name label")
	}
	portStr := ""
	if portNum > 0 {
		portStr = fmt.Sprintf("_%d", portNum)
	}

	finalName := fmt.Sprintf("%s_%s_%s%s", app, namespace, comp, portStr)
	// Check for Kubernetes name length requirement
	if len(finalName) > 63 {
		finalName = fmt.Sprintf("%s_%s%s", app, namespace, portStr)
		if len(finalName) > 63 {
			return finalName[:63], nil
		}
	}
	return finalName, nil
}
func getNamespaceFromObjectMetaOrDefault(meta metav1.ObjectMeta) string {
	name := meta.Namespace
	if name == "" {
		return "default"
	}
	return name
}

// ScrapeInfo captures the information needed to construct the service monitor for a generic workload


func PrintServiceMonitor(serviceMonitor *promoperapi.ServiceMonitor)(error){
	fmt.Println("virtual-service", serviceMonitor)
	directoryPath := "/Users/adalua/GolandProjects/verrazzano/tools/oam-converter/"
	fileName := "sm.yaml"
	filePath := filepath.Join(directoryPath, fileName)

	virtualServiceYaml, err := yaml.Marshal(serviceMonitor)
	if err != nil {
		fmt.Printf("Failed to marshal: %v\n", err)
		return err
	}
	// Write the YAML content to the file
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return err
	}
	defer file.Close()

	// Append the YAML content to the file
	_, err = file.Write(virtualServiceYaml)
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
		return err
	}
	_, err = file.WriteString("---\n")
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
		return err
	}
	return nil
}

func getPortSpecs(trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec) []vzapi.PortSpec {
	ports := trait.Spec.Ports
	if len(ports) == 0 {
		// create a port spec from the existing port
		ports = []vzapi.PortSpec{{Port: trait.Spec.Port, Path: trait.Spec.Path}}
	} else {
		// if there are existing ports and a port/path setting, add the latter to the ports
		if trait.Spec.Port != nil {
			// add the port to the ports
			path := trait.Spec.Path
			if path == nil {
				path = traitDefaults.Path
			}
			portSpec := vzapi.PortSpec{
				Port: trait.Spec.Port,
				Path: path,
			}
			ports = append(ports, portSpec)
		}
	}
	return ports
}
