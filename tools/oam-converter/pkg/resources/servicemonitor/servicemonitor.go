// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package servicemonitor

import (
	"fmt"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	extract "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/traits"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
)

const (
	prometheusClusterNameLabel = "verrazzano_cluster"
)

func CreateServiceMonitor(trait *vzapi.MetricsTrait) (monitor promoperapi.ServiceMonitor, err error) {

	serviceMonitor := promoperapi.ServiceMonitor{}

	// Creating a service monitor with name and namespace
	pmName, err := createServiceMonitorName(trait, 0)
	if err != nil {
		return serviceMonitor, err
	}

	// Fetch workload resource using information from the trait
	var workload *unstructured.Unstructured
	workload, err = extract.FetchWorkloadFromTrait(trait)
	if err != nil {
		return serviceMonitor, err
	}

	//fetch trait defaults
	traitDefaults, supported, err := extract.FetchTraitDefaults(workload)
	if err != nil {
		return serviceMonitor, err
	}
	if !supported || traitDefaults == nil {
		return serviceMonitor, err
	}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	secret, err := extract.FetchSourceCredentialsSecretIfRequired(trait, traitDefaults, workload)
	if err != nil {
		return serviceMonitor, err
	}
	//fetch if trait uses Istio
	useHTTPS, err := extract.UseHTTPSForScrapeTarget(trait)
	if err != nil {
		return serviceMonitor, err
	}
	//fetch if workload is WebLogic
	wlsWorkload, err := extract.IsWLSWorkload(workload)
	if err != nil {
		return serviceMonitor, err
	}

	vzPromLabels := !wlsWorkload

	//populate servicemonitor scrape info
	scrapeInfo := ScrapeInfo{
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
	PopulateServiceMonitor(scrapeInfo, &serviceMonitor)
	PrintServiceMonitor(&serviceMonitor)
	return serviceMonitor, nil

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
type ScrapeInfo struct {
	// The path by which Prometheus should scrape metrics
	Path *string
	// The number of ports located for the workload
	Ports int
	// The basic authentication secret required for the service monitor if applicable
	BasicAuthSecret *corev1.Secret
	// Determines whether to enable Istio for the generated service monitor
	IstioEnabled *bool
	// Verify if the scrape target uses the Verrazzano Prometheus Labels
	VZPrometheusLabels *bool
	// The map to generate keep labels
	// This matches the expected pod labels to the scrape config
	KeepLabels map[string]string
	// The name of the cluster for the selected workload
	ClusterName string
}

// PopulateServiceMonitor populates the Service Monitor to prepare for a create or update
// the Service Monitor reflects the specifications defined in the ScrapeInfo object
func PopulateServiceMonitor(info ScrapeInfo, serviceMonitor *promoperapi.ServiceMonitor) error {
	// Create the Service Monitor selector from the info label if it exists
	if serviceMonitor.ObjectMeta.Labels == nil {
		serviceMonitor.ObjectMeta.Labels = make(map[string]string)
	}
	serviceMonitor.Labels["release"] = "prometheus-operator"
	serviceMonitor.Spec.NamespaceSelector = promoperapi.NamespaceSelector{
		MatchNames: []string{serviceMonitor.Namespace},
	}

	// Clear the existing endpoints to avoid duplications
	serviceMonitor.Spec.Endpoints = nil

	// Loop through ports in the info and create scrape targets for each
	for i := 0; i < info.Ports; i++ {
		endpoint, err := createServiceMonitorEndpoint(info, i)
		if err != nil {
			print(err)
		}
		serviceMonitor.Spec.Endpoints = append(serviceMonitor.Spec.Endpoints, endpoint)
	}
	return nil
}

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

// createServiceMonitorEndpoint creates an endpoint for a given port increment and info
// this function effectively creates a scrape config for the workload target through the Service Monitor API
func createServiceMonitorEndpoint(info ScrapeInfo, portIncrement int) (promoperapi.Endpoint, error) {
	var endpoint promoperapi.Endpoint
	enabledHTTP2 := false
	// Add the secret username and password if basic auth is required for this endpoint
	// The secret has to exist in the workload and namespace
	if secret := info.BasicAuthSecret; secret != nil {
		endpoint.BasicAuth = &promoperapi.BasicAuth{
			Username: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secret.Name,
				},
				Key: "username",
			},
			Password: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secret.Name,
				},
				Key: "password",
			},
		}
	}
	endpoint.Scheme = "http"
	endpoint.Path = "/metrics"
	if info.Path != nil {
		endpoint.Path = *info.Path
	}

	if info.IstioEnabled != nil && *info.IstioEnabled {
		// The Prometheus Pod contains Istio certificates from the installation process
		// These certs are generated by Istio and are mounted as a volume on the Prometheus pod
		// ServiceMonitors are used to take advantage of these existing files because it allows us to reference the files in the volume
		certPath := "/etc/istio-certs"
		endpoint.EnableHttp2 = &enabledHTTP2
		endpoint.Scheme = "https"
		endpoint.TLSConfig = &promoperapi.TLSConfig{
			CAFile:   fmt.Sprintf("%s/root-cert.pem", certPath),
			CertFile: fmt.Sprintf("%s/cert-chain.pem", certPath),
			KeyFile:  fmt.Sprintf("%s/key.pem", certPath),
		}
		endpoint.TLSConfig.InsecureSkipVerify = true
	}

	// Change the expected labels based on the workload type
	enabledLabel := "__meta_kubernetes_pod_annotation_prometheus_io_scrape"
	portLabel := "__meta_kubernetes_pod_annotation_prometheus_io_port"
	pathLabel := "__meta_kubernetes_pod_annotation_prometheus_io_path"
	if info.VZPrometheusLabels != nil && *info.VZPrometheusLabels {
		var portString string
		if portIncrement > 0 {
			portString = strconv.Itoa(portIncrement)
		}
		enabledLabel = fmt.Sprintf("__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled%s", portString)
		portLabel = fmt.Sprintf("__meta_kubernetes_pod_annotation_verrazzano_io_metricsPort%s", portString)
		pathLabel = fmt.Sprintf("__meta_kubernetes_pod_annotation_verrazzano_io_metricsPath%s", portString)
	}

	// Add default cluster name if not populated
	if info.ClusterName == "" {
		info.ClusterName = constants.DefaultClusterName
	}

	// Relabel the cluster name
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Replacement: info.ClusterName,
		TargetLabel: prometheusClusterNameLabel,
	})

	// Relabel to match the expected labels
	regexString := "true"
	sourceLabels := []promoperapi.LabelName{promoperapi.LabelName(enabledLabel)}
	for key, val := range info.KeepLabels {
		sourceLabels = append(sourceLabels, promoperapi.LabelName(key))
		regexString = fmt.Sprintf("%s;%s", regexString, val)
	}
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:       "keep",
		Regex:        regexString,
		SourceLabels: sourceLabels,
	})

	// Replace the metrics path if specified
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action: "replace",
		Regex:  "(.+)",
		SourceLabels: []promoperapi.LabelName{
			promoperapi.LabelName(pathLabel),
		},
		TargetLabel: "__metrics_path__",
	})

	// Relabel the address of the metrics endpoint
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Regex:       `([^:]+)(?::\d+)?;(\d+)`,
		Replacement: "$1:$2",
		SourceLabels: []promoperapi.LabelName{
			"__address__",
			promoperapi.LabelName(portLabel),
		},
		TargetLabel: "__address__",
	})

	// Relabel the namespace label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Regex:       `(.*)`,
		Replacement: "$1",
		SourceLabels: []promoperapi.LabelName{
			"__meta_kubernetes_namespace",
		},
		TargetLabel: "namespace",
	})

	// Relabel the pod label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action: "labelmap",
		Regex:  `__meta_kubernetes_pod_label_(.+)`,
	})

	// Relabel the pod name label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action: "replace",
		SourceLabels: []promoperapi.LabelName{
			"__meta_kubernetes_pod_name",
		},
		TargetLabel: "pod_name",
	})

	// Drop the controller revision hash label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action: "labeldrop",
		Regex:  `(controller_revision_hash)`,
	})

	// Relabel the webapp label
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Regex:       `.*/(.*)$`,
		Replacement: "$1",
		SourceLabels: []promoperapi.LabelName{
			"name",
		},
		TargetLabel: "webapp",
	})

	// Add a relabel config that will copy the value of "app" to "application" if "application" is empty
	endpoint.RelabelConfigs = append(endpoint.RelabelConfigs, &promoperapi.RelabelConfig{
		Action:      "replace",
		Regex:       `;(.*)`,
		Replacement: "$1",
		Separator:   ";",
		SourceLabels: []promoperapi.LabelName{
			"application",
			"app",
		},
		TargetLabel: "application",
	})

	return endpoint, nil
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
