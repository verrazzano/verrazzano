// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
)

const (
	jaegerSpanIndexPrefix       = "verrazzano-jaeger-span"
	jaegerClusterNameLabel      = "verrazzano_cluster"
	adminClusterName            = "local"
	jaegerOperatorSampleMetric  = "jaeger_operator_instances_managed"
	jaegerAgentSampleMetric     = "jaeger_agent_collector_proxy_total"
	jaegerQuerySampleMetric     = "jaeger_query_requests_total"
	jaegerCollectorSampleMetric = "jaeger_collector_queue_capacity"
	jaegerESIndexCleanerJob     = "jaeger-operator-jaeger-es-index-cleaner"
	componentLabelKey           = "app.kubernetes.io/component"
	instanceLabelKey            = "app.kubernetes.io/instance"
)

const (
	jaegerListServicesErrFmt = "Error listing services in Jaeger: url=%s, error=%v"
	jaegerListTracesErrFmt   = "Error listing traces in Jaeger: url=%s, error=%v"
)

var (
	// common services running in both admin and managed cluster
	managedClusterSystemServiceNames = []string{
		"verrazzano-authproxy.verrazzano-system",
		"fluentd.verrazzano-system",
	}

	// services that are common plus the ones unique to admin cluster
	adminClusterSystemServiceNames = append(managedClusterSystemServiceNames,
		"jaeger-operator-jaeger.verrazzano-monitoring",
		"verrazzano-monitoring-operator.verrazzano-system",
		"ingress-controller-ingress-nginx-controller.ingress-nginx",
		"system-es-master.verrazzano-system")
)

type JaegerTraceData struct {
	TraceID string `json:"traceID"`
	Spans   []struct {
		TraceID       string `json:"traceID"`
		SpanID        string `json:"spanID"`
		Flags         int    `json:"flags"`
		OperationName string `json:"operationName"`
		References    []struct {
			RefType string `json:"refType"`
			TraceID string `json:"traceID"`
			SpanID  string `json:"spanID"`
		} `json:"references"`
		StartTime int64 `json:"startTime"`
		Duration  int   `json:"duration"`
		Tags      []struct {
			Key   string      `json:"key"`
			Type  string      `json:"type"`
			Value interface{} `json:"value"`
		} `json:"tags"`
		Logs []struct {
			Timestamp int64 `json:"timestamp"`
			Fields    []struct {
				Key   string `json:"key"`
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"fields"`
		} `json:"logs"`
		ProcessID string      `json:"processID"`
		Warnings  interface{} `json:"warnings"`
	} `json:"spans"`
	Processes struct {
		P1 struct {
			ServiceName string `json:"serviceName"`
			Tags        []struct {
				Key   string `json:"key"`
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"tags"`
		} `json:"p1"`
	} `json:"processes"`
	Warnings interface{} `json:"warnings"`
}

type JaegerTraceDataWrapper struct {
	Data   []JaegerTraceData `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Errors interface{}       `json:"errors"`
}

//IsJaegerInstanceCreated checks whether the default Jaeger CR is created
func IsJaegerInstanceCreated(kubeconfigPath string) (bool, error) {
	collectorDeployments, err := GetJaegerCollectorDeployments(kubeconfigPath, globalconst.JaegerInstanceName)
	if err != nil {
		return false, err
	}
	Log(Info, fmt.Sprintf("cluster has %d jaeger-collector deployments", len(collectorDeployments)))
	queryDeployments, err := GetJaegerQueryDeployments(kubeconfigPath, globalconst.JaegerInstanceName)
	if err != nil {
		return false, err
	}
	Log(Info, fmt.Sprintf("cluster has %d jaeger-query deployments", len(queryDeployments)))
	return len(collectorDeployments) > 0 && len(queryDeployments) > 0, nil
}

// GetJaegerCollectorDeployments returns the deployment object of the Jaeger collector corresponding to the given
//		Jaeger instance. If no instance name is provided, then it returns all Jaeger collector pods in the
////		verrazzano-monitoring namespace.
func GetJaegerCollectorDeployments(kubeconfigPath, jaegerCRName string) ([]appsv1.Deployment, error) {
	labels := map[string]string{
		componentLabelKey: globalconst.JaegerCollectorComponentName,
	}
	if jaegerCRName != "" {
		labels[instanceLabelKey] = jaegerCRName
	}
	Log(Info, fmt.Sprintf("Checking for collector deployments with labels %v", labels))
	deployments, err := ListDeploymentsMatchingLabelsInCluster(kubeconfigPath, constants.VerrazzanoMonitoringNamespace, labels)
	if err != nil {
		return nil, err
	}
	return deployments.Items, err
}

// GetJaegerQueryDeployments returns the deployment object of the Jaeger query corresponding to the given
//		Jaeger instance. If no Jaeger instance name is provided, then it returns all Jaeger query pods in the
//		verrazzano-monitoring namespace
func GetJaegerQueryDeployments(kubeconfigPath, jaegerCRName string) ([]appsv1.Deployment, error) {
	labels := map[string]string{
		componentLabelKey: globalconst.JaegerQueryComponentName,
	}
	if jaegerCRName != "" {
		labels[instanceLabelKey] = jaegerCRName
	}
	Log(Info, fmt.Sprintf("Checking for query deployments with labels %v", labels))
	deployments, err := ListDeploymentsMatchingLabelsInCluster(kubeconfigPath, constants.VerrazzanoMonitoringNamespace, labels)
	if err != nil {
		return nil, err
	}
	return deployments.Items, err
}

//JaegerSpanRecordFoundInOpenSearch checks if jaeger span records are found in OpenSearch storage
func JaegerSpanRecordFoundInOpenSearch(kubeconfigPath string, after time.Time, serviceName string) bool {
	indexName, err := GetJaegerSpanIndexName(kubeconfigPath)
	if err != nil {
		return false
	}
	fields := map[string]string{
		"process.serviceName": serviceName,
	}
	searchResult := querySystemElasticSearch(indexName, fields, kubeconfigPath)
	if len(searchResult) == 0 {
		Log(Info, fmt.Sprintf("Expected to find log record matching fields %v", fields))
		return false
	}
	found := findJaegerSpanHits(searchResult, &after)
	if !found {
		Log(Error, fmt.Sprintf("Failed to find recent jaeger span record for service %s", serviceName))
	}
	return found
}

//GetJaegerSpanIndexName returns the index name used in OpenSearch used for storage
func GetJaegerSpanIndexName(kubeconfigPath string) (string, error) {
	var jaegerIndices []string
	for _, indexName := range listSystemElasticSearchIndices(kubeconfigPath) {
		if strings.HasPrefix(indexName, jaegerSpanIndexPrefix) {
			jaegerIndices = append(jaegerIndices, indexName)
			break
		}
	}
	if len(jaegerIndices) > 0 {
		return jaegerIndices[0], nil
	}
	return "", fmt.Errorf("Jaeger Span index not found")
}

// IsJaegerMetricFound validates if the given jaeger metrics contain the labels with values specified as key-value pairs of the map
func IsJaegerMetricFound(kubeconfigPath, metricName, clusterName string, kv map[string]string) bool {
	compMetrics, err := QueryMetricWithLabel(metricName, kubeconfigPath, jaegerClusterNameLabel, clusterName)
	if err != nil {
		return false
	}
	metrics := JTq(compMetrics, "data", "result").([]interface{})
	for _, metric := range metrics {
		metricFound := true
		for key, value := range kv {
			if Jq(metric, "metric", key) != value {
				metricFound = false
				break
			}
		}
		return metricFound
	}
	return false
}

//ListJaegerTracesWithTags lists all trace ids for a given service with the given tags
func ListJaegerTracesWithTags(kubeconfigPath string, start time.Time, serviceName string, tags map[string]string) []string {
	var traces []string
	params := url.Values{}
	params.Add("service", serviceName)
	params.Add("start", strconv.FormatInt(start.UnixMicro(), 10))
	params.Add("end", strconv.FormatInt(time.Now().UnixMicro(), 10))
	jsonStr, err := json.Marshal(tags)
	if err != nil {
		Log(Error, fmt.Sprintf("Error parsing tags %v to JSON string", tags))
		return traces
	}
	params.Add("tags", string(jsonStr))
	url := fmt.Sprintf("%s/api/traces?%s", getJaegerURL(kubeconfigPath), params.Encode())
	username, password, err := getJaegerUsernamePassword(kubeconfigPath)
	if err != nil {
		return traces
	}
	resp, err := getJaegerWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(jaegerListTracesErrFmt, url, err))
		return traces
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf(jaegerListTracesErrFmt, url, resp.StatusCode))
		return traces
	}
	var jaegerTraceDataWrapper JaegerTraceDataWrapper
	json.Unmarshal(resp.Body, &jaegerTraceDataWrapper)
	for _, traceObj := range jaegerTraceDataWrapper.Data {
		traces = append(traces, traceObj.TraceID)
	}
	return traces
}

//ListServicesInJaeger lists the services whose traces are available in Jaeger
func ListServicesInJaeger(kubeconfigPath string) []string {
	var services []string
	url := fmt.Sprintf("%s/api/services", getJaegerURL(kubeconfigPath))
	username, password, err := getJaegerUsernamePassword(kubeconfigPath)
	if err != nil {
		return services
	}
	resp, err := getJaegerWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(jaegerListServicesErrFmt, url, err))
		return services
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf(jaegerListServicesErrFmt, url, resp.StatusCode))
		return services
	}
	var serviceMap map[string][]string
	json.Unmarshal(resp.Body, &serviceMap)
	services = append(services, serviceMap["data"]...)
	return services
}

// DoesCronJobExist returns whether a cronjob with the given name and namespace exists for the cluster
func DoesCronJobExist(kubeconfigPath, namespace string, name string) (bool, error) {
	cronjobs, err := ListCronJobNamesMatchingLabels(kubeconfigPath, namespace, nil)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed listing deployments in cluster for namespace %s: %v", namespace, err))
		return false, err
	}
	for _, cronJobName := range cronjobs {
		if strings.HasPrefix(cronJobName, name) {
			return true, nil
		}
	}
	return false, nil
}

// ListDeploymentsMatchingLabelsInCluster returns the list of deployments in a given namespace matching the given labels for the cluster
func ListDeploymentsMatchingLabelsInCluster(kubeconfigPath, namespace string, matchLabels map[string]string) (*appsv1.DeploymentList, error) {
	// Get the Kubernetes clientset
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	listOptions := metav1.ListOptions{}
	if matchLabels != nil {
		selector := labels.NewSelector()
		for k, v := range matchLabels {
			selectorLabel, _ := labels.NewRequirement(k, selection.Equals, []string{v})
			selector = selector.Add(*selectorLabel)
		}
		listOptions.LabelSelector = selector.String()
	}
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), listOptions)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list deployments in namespace %s: %v", namespace, err))
		return nil, err
	}
	return deployments, nil
}

// ListCronJobNamesMatchingLabels returns the list of cronjobs in a given namespace matching the given labels for the cluster
func ListCronJobNamesMatchingLabels(kubeconfigPath, namespace string, matchLabels map[string]string) ([]string, error) {
	var cronjobNames []string
	// Get the Kubernetes clientset
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	info, err := clientset.ServerVersion()
	if err != nil {
		return nil, err
	}
	majorVersion, err := strconv.Atoi(info.Major)
	if err != nil {
		return nil, err
	}
	if majorVersion > 1 {
		return nil, fmt.Errorf("Unknown major version %d", majorVersion)
	}
	minorVersion, err := strconv.Atoi(info.Minor)
	if err != nil {
		return nil, err
	}
	// For k8s version 1.20 and lesser, cronjobs are created under version batch/v1beta1
	// For k8s version greater than 1.20, cronjobs are created under version batch/v1
	if minorVersion <= 20 {
		cronJobs, err := listV1Beta1CronJobNames(clientset, namespace, fillLabelSelectors(matchLabels))
		if err != nil {
			return nil, err
		}
		for _, cronjob := range cronJobs {
			cronjobNames = append(cronjobNames, cronjob.Name)
		}
	} else {
		cronJobs, err := listV1CronJobNames(clientset, namespace, fillLabelSelectors(matchLabels))
		if err != nil {
			return nil, err
		}
		for _, cronjob := range cronJobs {
			cronjobNames = append(cronjobNames, cronjob.Name)
		}
	}
	return cronjobNames, nil
}

// GetJaegerSystemServicesInManagedCluster returns the system services that needs to be running in a managed cluster
func GetJaegerSystemServicesInManagedCluster() []string {
	return managedClusterSystemServiceNames
}

// GetJaegerSystemServicesInAdminCluster returns the system services that needs to be running in a admin cluster
func GetJaegerSystemServicesInAdminCluster() []string {
	return adminClusterSystemServiceNames
}

// ValidateJaegerOperatorMetricFunc returns a function that validates if metrics of Jaeger operator is scraped by prometheus.
func ValidateJaegerOperatorMetricFunc() func() bool {
	return func() bool {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return false
		}
		return IsJaegerMetricFound(kubeconfigPath, jaegerOperatorSampleMetric, adminClusterName, nil)
	}
}

// ValidateJaegerCollectorMetricFunc returns a function that validates if metrics of Jaeger collector is scraped by prometheus.
func ValidateJaegerCollectorMetricFunc() func() bool {
	return func() bool {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return false
		}
		return IsJaegerMetricFound(kubeconfigPath, jaegerCollectorSampleMetric, adminClusterName, nil)
	}
}

// ValidateJaegerQueryMetricFunc returns a function that validates if metrics of Jaeger query is scraped by prometheus.
func ValidateJaegerQueryMetricFunc() func() bool {
	return func() bool {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return false
		}
		return IsJaegerMetricFound(kubeconfigPath, jaegerQuerySampleMetric, adminClusterName, nil)
	}
}

// ValidateJaegerAgentMetricFunc returns a function that validates if metrics of Jaeger agent is scraped by prometheus.
func ValidateJaegerAgentMetricFunc() func() bool {
	return func() bool {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return false
		}
		return IsJaegerMetricFound(kubeconfigPath, jaegerAgentSampleMetric, adminClusterName, nil)
	}
}

// ValidateEsIndexCleanerCronJobFunc returns a function that validates if cron job for periodically cleaning the OS indices are created.
func ValidateEsIndexCleanerCronJobFunc() func() (bool, error) {
	return func() (bool, error) {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return false, err
		}
		create := IsOpensearchEnabled(kubeconfigPath)
		if create {
			return DoesCronJobExist(kubeconfigPath, constants.VerrazzanoMonitoringNamespace, jaegerESIndexCleanerJob)
		}
		return false, nil
	}
}

// ValidateSystemTracesFuncInCluster returns a function that validates if system traces for the given cluster can be successfully queried from Jaeger
func ValidateSystemTracesFuncInCluster(kubeconfigPath string, start time.Time, clusterName string) func() (bool, error) {
	return func() (bool, error) {
		// Check if the service name is registered in Jaeger and traces are present for that service
		systemServices := GetJaegerSystemServicesInManagedCluster()
		if clusterName == "admin" || clusterName == "local" {
			systemServices = GetJaegerSystemServicesInAdminCluster()
		}
		tracesFound := true
		for i := 0; i < len(systemServices); i++ {
			Log(Info, fmt.Sprintf("Inspecting traces for service: %s", systemServices[i]))
			if i == 0 {
				tracesFound =
					len(ListJaegerTracesWithTags(kubeconfigPath, start, systemServices[i],
						map[string]string{"verrazzano_cluster": clusterName})) > 0
			} else {
				tracesFound = tracesFound && len(ListJaegerTracesWithTags(kubeconfigPath, start, systemServices[i],
					map[string]string{"verrazzano_cluster": clusterName})) > 0
			}
			Log(Info, fmt.Sprintf("Trace found flag for service: %s is %v", systemServices[i], tracesFound))
			// return early and retry later
			if !tracesFound {
				return false, nil
			}
		}
		return tracesFound, nil
	}
}

// ValidateSystemTracesInOSFunc returns a function that validates if system traces are stored successfully in OS backend storage
func ValidateSystemTracesInOSFunc(start time.Time) func() bool {
	return func() bool {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return false
		}
		tracesFound := true
		systemServices := GetJaegerSystemServicesInAdminCluster()
		for i := 0; i < len(systemServices); i++ {
			Log(Info, fmt.Sprintf("Finding traces for service %s after %s", systemServices[i], start.String()))
			if i == 0 {
				tracesFound = JaegerSpanRecordFoundInOpenSearch(kubeconfigPath, start, systemServices[i])
			} else {
				tracesFound = tracesFound && JaegerSpanRecordFoundInOpenSearch(kubeconfigPath, start, systemServices[i])
			}
			// return early and retry later
			if !tracesFound {
				return false
			}
		}
		return tracesFound
	}
}

// ValidateApplicationTracesInCluster returns a function that validates if application traces can be successfully queried from Jaeger
func ValidateApplicationTracesInCluster(kubeconfigPath string, start time.Time, appServiceName, clusterName string) func() (bool, error) {
	return func() (bool, error) {
		tracesFound := false
		servicesWithJaegerTraces := ListServicesInJaeger(kubeconfigPath)
		for _, serviceName := range servicesWithJaegerTraces {
			Log(Info, fmt.Sprintf("Checking if service name %s matches the expected app service %s", serviceName, appServiceName))
			if strings.HasPrefix(serviceName, appServiceName) {
				Log(Info, fmt.Sprintf("Finding traces for service %s after %s", serviceName, start.String()))
				traceIds := ListJaegerTracesWithTags(kubeconfigPath, start, appServiceName,
					map[string]string{"verrazzano_cluster": clusterName})
				tracesFound = len(traceIds) > 0
				if !tracesFound {
					errMsg := fmt.Sprintf("traces not found for service: %s", serviceName)
					Log(Error, errMsg)
					return false, fmt.Errorf(errMsg)
				}
				break
			}
		}
		return tracesFound, nil
	}
}

// ValidateApplicationTracesInOS returns a function that validates if application traces are stored successfully in OS backend storage
func ValidateApplicationTracesInOS(start time.Time, appServiceName string) func() bool {
	return func() bool {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return false
		}
		return JaegerSpanRecordFoundInOpenSearch(kubeconfigPath, start, appServiceName)
	}
}

// fillLabelSelectors fills the match labels from map to be passed in list options
func fillLabelSelectors(matchLabels map[string]string) metav1.ListOptions {
	listOptions := metav1.ListOptions{}
	if matchLabels != nil {
		var selector labels.Selector
		for k, v := range matchLabels {
			selectorLabel, _ := labels.NewRequirement(k, selection.Equals, []string{v})
			selector = labels.NewSelector()
			selector = selector.Add(*selectorLabel)
		}
		listOptions.LabelSelector = selector.String()
	}
	return listOptions
}

// listV1CronJobNames lists the cronjob under batch/v1 api version for k8s version > 1.20
func listV1CronJobNames(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) ([]v1.CronJob, error) {
	var cronJobs []v1.CronJob
	cronJobList, err := clientset.BatchV1().CronJobs(namespace).List(context.TODO(), listOptions)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list v1/cronjobs in namespace %s: %v", namespace, err))
		return cronJobs, err
	}
	return cronJobList.Items, nil
}

// listV1Beta1CronJobNames lists the cronjob under batch/v1beta1 api version for k8s version <= 1.20
func listV1Beta1CronJobNames(clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) ([]v1beta1.CronJob, error) {
	var cronJobs []v1beta1.CronJob
	cronJobList, err := clientset.BatchV1beta1().CronJobs(namespace).List(context.TODO(), listOptions)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list v1beta1/cronjobs in namespace %s: %v", namespace, err))
		return cronJobs, err
	}
	return cronJobList.Items, nil
}

// getJaegerWithBasicAuth access Jaeger with GET using basic auth, using a given kubeconfig
func getJaegerWithBasicAuth(url string, hostHeader string, username string, password string, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := getJaegerClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "GET", "", hostHeader, username, password, nil, retryableClient)
}

// getJaegerClient returns the Jaeger client which can be used for GET/POST operations using a given kubeconfig
func getJaegerClient(kubeconfigPath string) (*retryablehttp.Client, error) {
	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return client, err
}

// getJaegerURL returns Jaeger URL from the corresponding ingress resource using the given kubeconfig
func getJaegerURL(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingressList, _ := clientset.NetworkingV1().Ingresses(VerrazzanoNamespace).List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "verrazzano-jaeger" {
			Log(Info, fmt.Sprintf("Found Jaeger Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
			return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
		}
	}
	return ""
}

// getJaegerUsernamePassword returns the username/password for connecting to Jaeger
func getJaegerUsernamePassword(kubeconfigPath string) (username, password string, err error) {
	password, err = GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return "", "", err
	}
	return "verrazzano", password, err
}

// findJaegerSpanHits returns the number of span hits that are older than the given time
func findJaegerSpanHits(searchResult map[string]interface{}, after *time.Time) bool {
	hits := Jq(searchResult, "hits", "hits")
	if hits == nil {
		Log(Info, "Expected to find hits in span record query results")
		return false
	}
	Log(Info, fmt.Sprintf("Found %d records", len(hits.([]interface{}))))
	if len(hits.([]interface{})) == 0 {
		Log(Info, "Expected span record query results to contain at least one hit")
		return false
	}
	if after == nil {
		return true
	}
	for _, hit := range hits.([]interface{}) {
		timestamp := Jq(hit, "_source", "startTimeMillis")
		t := time.UnixMilli(int64(timestamp.(float64)))
		if t.After(*after) {
			Log(Info, fmt.Sprintf("Found recent record: %f", timestamp))
			return true
		}
		Log(Info, fmt.Sprintf("Found old record: %f", timestamp))
	}
	return true
}
