// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/onsi/gomega"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"html/template"
	"net/http"
	url2 "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ISO8601Layout defines the timestamp format
	ISO8601Layout = "2006-01-02T15:04:05.999999999-07:00"
)

// GetOpenSearchSystemIndex in Verrazzano 1.3.0, indices in the verrazzano-system namespace have been migrated
// to the verrazzano-system data stream
func GetOpenSearchSystemIndex(name string) (string, error) {
	return GetOpenSearchSystemIndexWithKC(name, "")
}

// GetOpenSearchSystemIndexWithKC is the same as GetOpenSearchSystemIndex but the kubeconfig may be specified for MC tests
func GetOpenSearchSystemIndexWithKC(name, kubeconfigPath string) (string, error) {
	usingDataStreams, err := isUsingDataStreams(kubeconfigPath)
	if err != nil {
		return "", err
	}
	if usingDataStreams {
		return "verrazzano-system", nil
	}
	if name == "systemd-journal" {
		return "verrazzano-systemd-journal", nil
	}
	return "verrazzano-namespace-" + name, nil
}

// Retention/Rollover policy names in ISM plugin
const (
	SystemLogIsmPolicyName      = "verrazzano-system"
	ApplicationLogIsmPolicyName = "verrazzano-application"
)

// Error logging formats
const (
	queryErrorFormat      = "Error retrieving Elasticsearch query results: url=%s, error=%s"
	queryStatusFormat     = "Error retrieving Elasticsearch query results: url=%s, status=%d"
	kubeconfigErrorFormat = "Error getting kubeconfig: %v"
)

// URL formats
const (
	getDataStreamURLFormat    = "%s/_data_stream/"
	listDataStreamURLFormat   = "%s/_data_stream"
	deleteDataStreamURLFormat = "%s/_data_stream/%s"
)

// Default values for Retention period and Rollover period
var (
	DefaultRetentionPeriod = "7d"
	DefaultRolloverPeriod  = "1d"
)

// ISMPolicy definition
type ISMPolicy struct {
	PolicyID        string      `json:"policy_id"`
	Description     string      `json:"description"`
	LastUpdatedTime int64       `json:"last_updated_time"`
	SchemaVersion   int         `json:"schema_version"`
	DefaultState    string      `json:"default_state"`
	States          []State     `json:"states"`
	IsmTemplate     IsmTemplate `json:"ism_template"`
}

// State defined in ISM policy
type State struct {
	Name        string       `json:"name"`
	Actions     []Action     `json:"actions"`
	Transitions []Transition `json:"transitions"`
}

// Rollover or Delete action defined in ISM policy
type Action struct {
	Rollover struct {
		MinIndexAge string `json:"min_index_age"`
	} `json:"rollover,omitempty"`
	Delete struct {
		MinIndexAge string `json:"min_index_age"`
	} `json:"delete,omitempty"`
}

// Transition defined in ISM policy
type Transition struct {
	StateName  string            `json:"state_name"`
	Conditions map[string]string `json:"conditions"`
}

// IsmTemplate defined in ISM policy
type IsmTemplate []struct {
	IndexPatterns   []string `json:"index_patterns"`
	Priority        int      `json:"priority"`
	LastUpdatedTime int64    `json:"last_updated_time"`
}

// IndexMetadata contains information about a particular
type IndexMetadata struct {
	Mapping struct {
		TotalFields struct {
			Limit string `json:"limit"`
		} `json:"total_fields"`
	} `json:"mapping"`
	RefreshInterval    string `json:"refresh_interval"`
	Hidden             string `json:"hidden"`
	NumberOfShards     string `json:"number_of_shards"`
	AutoExpandReplicas string `json:"auto_expand_replicas"`
	ProvidedName       string `json:"provided_name"`
	CreationDate       string `json:"creation_date"`
	NumberOfReplicas   string `json:"number_of_replicas"`
	UUID               string `json:"uuid"`
	Version            struct {
		Created string `json:"created"`
	} `json:"version"`
}

// IndexSettings  parent object containing the index metadata
type IndexSettings struct {
	Settings struct {
		Index IndexMetadata `json:"index"`
	} `json:"settings"`
}

// DataStream details
type DataStream struct {
	Name           string `json:"name"`
	TimestampField struct {
		Name string `json:"name"`
	} `json:"timestamp_field"`
	Indices []struct {
		IndexName string `json:"index_name"`
		IndexUUID string `json:"index_uuid"`
	} `json:"indices"`
	Generation int    `json:"generation"`
	Status     string `json:"status"`
	Template   string `json:"template"`
}

// SearchResult represents the result of an Opensearch search query
type SearchResult struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore interface{}   `json:"max_score"`
		Hits     []interface{} `json:"hits"`
	} `json:"hits"`
}

// IndexListData represents the row of /_cat/indices?format=json output
type IndexListData struct {
	Health       string `json:"health"`
	Status       string `json:"status"`
	Index        string `json:"index"`
	UUID         string `json:"uuid"`
	Pri          string `json:"pri"`
	Rep          string `json:"rep"`
	DocsCount    string `json:"docsCount"`
	DocsDeleted  string `json:"docsDeleted"`
	StoreSize    string `json:"storeSize"`
	PriStoreSize string `json:"priStoreSize"`
}

type ElasticSearchISMPolicyAddModifier struct{}

type ElasticSearchISMPolicyRemoveModifier struct{}

func (u ElasticSearchISMPolicyAddModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
	if cr.Spec.Components.Elasticsearch.Policies == nil {
		cr.Spec.Components.Elasticsearch.Policies = []vmov1.IndexManagementPolicy{
			{
				PolicyName:   "verrazzano-system",
				IndexPattern: "verrazzano-system*",
				MinIndexAge:  &DefaultRetentionPeriod,
				Rollover: vmov1.RolloverPolicy{
					MinIndexAge: &DefaultRolloverPeriod,
				},
			},
			{
				PolicyName:   "verrazzano-application",
				IndexPattern: "verrazzano-application*",
				MinIndexAge:  &DefaultRetentionPeriod,
				Rollover: vmov1.RolloverPolicy{
					MinIndexAge: &DefaultRolloverPeriod,
				},
			},
		}
	}
}

func (u ElasticSearchISMPolicyRemoveModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{}
}

// GetOpenSearchAppIndex in Verrazzano 1.3.0, application indices have been migrated to data streams
// following the pattern 'verrazzano-application-<application name>'
func GetOpenSearchAppIndex(namespace string) (string, error) {
	return GetOpenSearchAppIndexWithKC(namespace, "")
}

// GetOpenSearchAppIndexWithKC is the same as GetOpenSearchAppIndex but kubeconfig may be specified for MC tests
func GetOpenSearchAppIndexWithKC(namespace, kubeconfigPath string) (string, error) {
	usingDataStreams, err := isUsingDataStreams(kubeconfigPath)
	if err != nil {
		return "", err
	}
	if usingDataStreams {
		return "verrazzano-application-" + namespace, nil
	}
	return "verrazzano-namespace-" + namespace, nil
}

func isUsingDataStreams(kubeconfigPath string) (bool, error) {
	kubeConfig, err := getKubeConfigPath(kubeconfigPath)
	if err != nil {
		return false, err
	}
	return IsVerrazzanoMinVersion("1.3.0", kubeConfig)
}

func UseExternalElasticsearch() bool {
	return os.Getenv("EXTERNAL_ELASTICSEARCH") == "true"
}

// GetExternalOpenSearchURL gets the external Elasticsearch URL
func GetExternalOpenSearchURL(kubeconfigPath string) string {
	opensearchSvc := "opensearch-cluster-master"
	// the equivalent of kubectl get svc opensearchSvc -o=jsonpath='{.status.loadBalancer.ingress[0].ip}'
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	svc, err := clientset.CoreV1().Services("default").Get(context.TODO(), opensearchSvc, metav1.GetOptions{})
	if err != nil {
		Log(Info, fmt.Sprintf("Could not get services quickstart-es-http in sockshop: %v\n", err.Error()))
		return ""
	}
	if svc.Status.LoadBalancer.Ingress != nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
		return fmt.Sprintf("https://%s:9200", svc.Status.LoadBalancer.Ingress[0].IP)
	}
	return ""
}

// GetSystemOpenSearchIngressURL gets the system Elasticsearch Ingress host in the given cluster
func GetSystemOpenSearchIngressURL(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingressList, _ := clientset.NetworkingV1().Ingresses(VerrazzanoNamespace).List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-os-ingest" {
			Log(Info, fmt.Sprintf("Found Elasticsearch Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
			return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
		}
	}
	return ""
}

// getElasticSearchURL returns VMI or external ES URL depending on env var EXTERNAL_ELASTICSEARCH
func getElasticSearchURL(kubeconfigPath string) string {
	if UseExternalElasticsearch() {
		return GetExternalOpenSearchURL(kubeconfigPath)
	}
	return GetSystemOpenSearchIngressURL(kubeconfigPath)
}

// getElasticSearchUsernamePassword returns the username/password for connecting to opensearch
func getElasticSearchUsernamePassword(kubeconfigPath string) (username, password string, err error) {
	if UseExternalElasticsearch() {
		esSecret, err := GetSecretInCluster(VerrazzanoNamespace, "external-es-secret", kubeconfigPath)
		if err != nil {
			Log(Error, fmt.Sprintf("Failed to get external-es-secret secret: %v", err))
			return "", "", err
		}
		return string(esSecret.Data["username"]), string(esSecret.Data["password"]), err
	}
	password, err = GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return "", "", err
	}
	return "verrazzano", password, err
}

// getElasticSearchWithBasicAuth access ES with GET using basic auth, using a given kubeconfig
func getElasticSearchWithBasicAuth(url string, hostHeader string, username string, password string, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := getElasticSearchClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "GET", "", hostHeader, username, password, nil, retryableClient)
}

// postElasticSearchWithBasicAuth retries POST using basic auth
func postElasticSearchWithBasicAuth(url, body, username, password, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := getElasticSearchClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "POST", "application/json", "", username, password, strings.NewReader(body), retryableClient)
}

// deleteElasticSearchWithBasicAuth retries DELETE using basic auth
func deleteElasticSearchWithBasicAuth(url, body, username, password, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := getElasticSearchClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "DELETE", "application/json", "", username, password, strings.NewReader(body), retryableClient)
}

// getElasticSearchClient returns ES client to perform http operations
func getElasticSearchClient(kubeconfigPath string) (*retryablehttp.Client, error) {
	var retryableClient *retryablehttp.Client
	var err error
	if UseExternalElasticsearch() {
		caCert, err := getExternalESCACert(kubeconfigPath)
		if err != nil {
			return nil, err
		}
		client, err := getHTTPClientWithCABundle(caCert, kubeconfigPath)
		if err != nil {
			return nil, err
		}
		retryableClient = newRetryableHTTPClient(client)
		if err != nil {
			return nil, err
		}
	} else {
		retryableClient, err = GetVerrazzanoHTTPClient(kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}
	return retryableClient, nil
}

// getExternalESCACert returns the CA cert from external-es-secret in the specified cluster
func getExternalESCACert(kubeconfigPath string) ([]byte, error) {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	certSecret, err := clientset.CoreV1().Secrets(VerrazzanoNamespace).Get(context.TODO(), "external-es-secret", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return certSecret.Data["ca-bundle"], nil
}

// listSystemElasticSearchIndices lists the system Elasticsearch indices in the given cluster
func listSystemElasticSearchIndices(kubeconfigPath string) []string {
	list := []string{}
	url := fmt.Sprintf("%s/_all", getElasticSearchURL(kubeconfigPath))
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return list
	}
	resp, err := getElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting Elasticsearch indices: url=%s, error=%v", url, err))
		return list
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error retrieving Elasticsearch indices: url=%s, status=%d", url, resp.StatusCode))
		return list
	}
	Log(Debug, fmt.Sprintf("indices: %s", resp.Body))
	var indexMap map[string]interface{}
	json.Unmarshal(resp.Body, &indexMap)
	for name := range indexMap {
		list = append(list, name)
	}
	return list
}

// querySystemElasticSearch searches the Elasticsearch index with the fields in the given cluster
func querySystemElasticSearch(index string, fields map[string]string, kubeconfigPath string) map[string]interface{} {
	query := ""
	for name, value := range fields {
		fieldQuery := fmt.Sprintf("%s:%s", name, value)
		if query == "" {
			query = fieldQuery
		} else {
			query = fmt.Sprintf("%s+AND+%s", query, fieldQuery)
		}
	}
	var result map[string]interface{}
	url := fmt.Sprintf("%s/%s/_doc/_search?q=%s", getElasticSearchURL(kubeconfigPath), index, query)
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return result
	}
	resp, err := getElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(queryErrorFormat, url, err))
		return result
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf(queryStatusFormat, url, resp.StatusCode))
		return result
	}
	Log(Debug, fmt.Sprintf("records: %s", resp.Body))
	json.Unmarshal(resp.Body, &result)
	return result
}

// queryDocumentsOlderThan searches the Elasticsearch index with the fields in the given cluster
func queryDocumentsOlderThan(index string, retentionPeriod string, kubeconfigPath string) (SearchResult, error) {
	var result SearchResult

	// validate Retention period
	_, err := CalculateSeconds(retentionPeriod)
	if err != nil {
		return result, err
	}

	query := "@timestamp:<now-" + retentionPeriod
	url := fmt.Sprintf("%s/%s/_search?q=%s", getElasticSearchURL(kubeconfigPath), index, url2.QueryEscape(query))
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return result, nil
	}
	resp, err := getElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(queryErrorFormat, url, err))
		return result, nil
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf(queryStatusFormat, url, resp.StatusCode))
		return result, nil
	}
	Log(Debug, fmt.Sprintf("records: %s", resp.Body))
	json.Unmarshal(resp.Body, &result)
	return result, nil
}

// LogIndexFound confirms a named index can be found in Elasticsearch in the cluster specified in the environment
func LogIndexFound(indexName string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return false
	}

	return LogIndexFoundInCluster(indexName, kubeconfigPath)
}

// LogIndexFoundInCluster confirms a named index can be found in Elasticsearch on the given cluster
func LogIndexFoundInCluster(indexName, kubeconfigPath string) bool {
	Log(Info, fmt.Sprintf("Looking for log index %s in cluster with kubeconfig %s", indexName, kubeconfigPath))
	for _, name := range listSystemElasticSearchIndices(kubeconfigPath) {
		// If old index or data stream backend index, return true
		backendIndexRe := regexp.MustCompile(`^\.ds-` + indexName + `-\d+$`)
		if name == indexName || backendIndexRe.MatchString(name) {
			return true
		}
	}
	Log(Error, fmt.Sprintf("Expected to find log index %s", indexName))
	return false
}

// GetSystemIndices returns metadata of indices of all system indices
func GetSystemIndices() ([]IndexMetadata, error) {
	systemIndices, err := GetIndexMetadataList(ListSystemIndices())
	if err != nil {
		return []IndexMetadata{}, err
	}
	return systemIndices, nil
}

// GetApplicationIndices returns the metadata of indices used by application indices
func GetApplicationIndices() ([]IndexMetadata, error) {
	applicationIndices, err := GetIndexMetadataList(ListApplicationIndices())
	if err != nil {
		return []IndexMetadata{}, err
	}
	return applicationIndices, nil
}

// GetBackingIndicesForDataStream returns metadata of all backing indices for a given data stream
func GetBackingIndicesForDataStream(dataStreamName string) ([]IndexMetadata, error) {
	dataStream, err := GetDataStream(dataStreamName)
	if err != nil {
		return []IndexMetadata{}, err
	}
	var indexMetadataList []IndexMetadata
	for _, index := range dataStream.Indices {
		indexMetadata, err := GetIndexMetadata(index.IndexName)
		if err != nil {
			return indexMetadataList, err
		}
		indexMetadataList = append(indexMetadataList, indexMetadata)
	}
	return indexMetadataList, nil
}

// ContainsIndicesOlderThanRetentionPeriod returns true if there are any old (backing) indices present for
// the given data stream that is older than the retention period. Returns false otherwise.
func ContainsIndicesOlderThanRetentionPeriod(indexMetadataList []IndexMetadata, oldestTimestamp int64) (bool, error) {
	for _, indexMetadata := range indexMetadataList {
		Log(Info, fmt.Sprintf("Checking if creation time of index %s is older than %d", indexMetadata.ProvidedName, oldestTimestamp))
		indexCreationTime, _ := strconv.ParseInt(indexMetadata.CreationDate, 10, 64)
		Log(Info, fmt.Sprintf("Creation time of index '%s' is '%d'", indexMetadata.ProvidedName, indexCreationTime))
		if indexCreationTime < oldestTimestamp {
			return true, nil
		}
	}
	return false, nil
}

// GetDataStream return the data stream object with the given
func GetDataStream(dataStreamName string) (DataStream, error) {
	var dataStream DataStream
	resp, err := doGetElasticSearchURL(getDataStreamURLFormat + dataStreamName)
	if err != nil {
		return dataStream, err
	}
	if resp.StatusCode == http.StatusOK {
		var dataStreamMap map[string][]DataStream
		json.Unmarshal(resp.Body, &dataStreamMap)
		dataStreams := dataStreamMap["data_streams"]
		if len(dataStreams) > 0 {
			// since the data stream object is queried using its name which is unique,
			// atmost one element would be present in this splice
			dataStream = dataStreams[0]
		}
	}
	return dataStream, nil
}

// IsDataStreamSupported returns true if data stream is supported false otherwise
func IsDataStreamSupported() bool {
	resp, err := doGetElasticSearchURL(listDataStreamURLFormat)
	if err != nil {
		Log(Error, err.Error())
		return false
	}
	if resp.StatusCode == http.StatusOK {
		var dataStreamMap map[string][]DataStream
		json.Unmarshal(resp.Body, &dataStreamMap)
		dataStreams := dataStreamMap["data_streams"]
		if len(dataStreams) > 0 {
			return true
		}
	}
	Log(Error, "No data streams created")
	return false
}

// WaitForISMPolicyUpdate waits for the VMO reconcile to complete and the ISM policies are created
func WaitForISMPolicyUpdate(pollingInterval time.Duration, timeout time.Duration) {
	gomega.Eventually(func() bool {
		ismPolicyExists, err := ISMPolicyExists(ApplicationLogIsmPolicyName)
		if err != nil {
			Log(Error, err.Error())
			return false
		}
		return ismPolicyExists
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

func ListSystemIndices() []string {
	return []string{
		"verrazzano-namespace-cert-manager",
		"verrazzano-namespace-verrazzano-system",
		"verrazzano-namespace-local-path-storage",
		"verrazzano-namespace-kube-system",
		"verrazzano-namespace-cattle-fleet-local-system",
		"verrazzano-namespace-ingress-nginx",
		"verrazzano-systemd-journal",
		"verrazzano-namespace-cattle-fleet-system",
		"verrazzano-namespace-istio-system",
		"verrazzano-namespace-monitoring",
		"verrazzano-namespace-cattle-system",
		"verrazzano-namespace-verrazzano-install",
	}
}

func ListApplicationIndices() []string {
	var indexList []string
	resp, err := doGetElasticSearchURL("%s/_cat/indices?format=json")
	if err != nil {
		return indexList
	}
	if resp.StatusCode == http.StatusOK {
		var indexListData []IndexListData
		json.Unmarshal(resp.Body, &indexListData)
		for _, indexData := range indexListData {
			if !isSystemIndex(indexData.Index) {
				indexList = append(indexList, indexData.Index)
			}
		}
	}
	return indexList
}

func GetIndexMetadataList(indexNames []string) ([]IndexMetadata, error) {
	var indexMetadataList []IndexMetadata
	for _, systemIndex := range indexNames {
		systemIndexMetadata, err := GetIndexMetadata(systemIndex)
		if err != nil {
			return indexMetadataList, err
		}
		indexMetadataList = append(indexMetadataList, systemIndexMetadata)
	}
	return indexMetadataList, nil
}

// isSystemIndex returns true if the given index is a verrazzano system index false otherwise
func isSystemIndex(indexName string) bool {
	if strings.HasPrefix(indexName, ".") {
		return true
	}
	for _, systemIndex := range ListSystemIndices() {
		if systemIndex == indexName {
			return true
		}
	}
	return false
}

// doGetElasticSearchURL helper method to execute a GET request to elastic search url and return the response
func doGetElasticSearchURL(urlFormat string) (*HTTPResponse, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf(urlFormat, getElasticSearchURL(kubeconfigPath))
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return getElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
}

// GetApplicationDataStreamNames returns the data stream names of all application logs having
// prefix 'verrazzano-application-'
func GetApplicationDataStreamNames() ([]string, error) {
	result := []string{}
	resp, err := doGetElasticSearchURL("%s/_data_stream")
	if err != nil {
		return result, err
	}
	if resp.StatusCode == http.StatusOK {
		var dataStreams map[string][]DataStream
		json.Unmarshal(resp.Body, &dataStreams)
		for _, dataStream := range dataStreams["data_streams"] {
			if strings.HasPrefix(dataStream.Name, "verrazzano-application-") {
				result = append(result, dataStream.Name)
			}
		}
	}
	return result, nil
}

// DeleteApplicationDataStream deletes the given applicatoin data stream
func DeleteApplicationDataStream(datastreamName string) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return err
	}
	url := fmt.Sprintf(deleteDataStreamURLFormat, getElasticSearchURL(kubeconfigPath), datastreamName)
	resp, err := deleteElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return nil
}

// GetIndexMetadata returns the metadata of the index
func GetIndexMetadata(indexName string) (IndexMetadata, error) {
	result := IndexMetadata{}
	resp, err := doGetElasticSearchURL("%s/" + indexName + "/_settings")
	if err != nil {
		return result, err
	}
	if resp.StatusCode == http.StatusOK {
		var settings map[string]IndexSettings
		json.Unmarshal(resp.Body, &settings)
		return settings[indexName].Settings.Index, nil
	}
	return result, nil
}

// GetIndexMetadataForDataStream returns the metadata of all backing indices of a given
// datastream
func GetIndexMetadataForDataStream(dataStreamName string) ([]IndexMetadata, error) {
	result := []IndexMetadata{}
	resp, err := doGetElasticSearchURL("%s/" + dataStreamName + "/_settings")
	if err != nil {
		return result, err
	}
	if resp.StatusCode == http.StatusOK {
		var settings map[string]IndexSettings
		json.Unmarshal(resp.Body, &settings)
		for _, indexSettings := range settings {
			result = append(result, indexSettings.Settings.Index)
		}
	}
	return result, nil
}

// LogRecordFound confirms a recent log record for the index with matching fields can be found
// in the cluster specified in the environment
func LogRecordFound(indexName string, after time.Time, fields map[string]string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return false
	}

	return LogRecordFoundInCluster(indexName, after, fields, kubeconfigPath)
}

// LogRecordFoundInCluster confirms a recent log record for the index with matching fields can be found
// in the given cluster
func LogRecordFoundInCluster(indexName string, after time.Time, fields map[string]string, kubeconfigPath string) bool {
	searchResult := querySystemElasticSearch(indexName, fields, kubeconfigPath)
	if len(searchResult) == 0 {
		Log(Info, fmt.Sprintf("Expected to find log record matching fields %v", fields))
		return false
	}
	found := findHits(searchResult, &after)
	if !found {
		Log(Error, fmt.Sprintf("Failed to find recent log record for index %s", indexName))
	}
	return found
}

// ContainsDocsOlderThanRetentionPeriod returns true if the given index contains any doc that
// is older than the retention period, returns false otherwise.
func ContainsDocsOlderThanRetentionPeriod(indexName string, retentionPeriod string) (bool, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return false, err
	}
	oldRecordsSearchResult, err := queryDocumentsOlderThan(indexName, retentionPeriod, kubeconfigPath)
	if err != nil {
		return false, err
	}
	return oldRecordsSearchResult.Hits.Total.Value > 0, nil
}

// findHits returns the number of hits that match a given search query
func findHits(searchResult map[string]interface{}, after *time.Time) bool {
	hits := Jq(searchResult, "hits", "hits")
	if hits == nil {
		Log(Info, "Expected to find hits in log record query results")
		return false
	}
	Log(Info, fmt.Sprintf("Found %d records", len(hits.([]interface{}))))
	if len(hits.([]interface{})) == 0 {
		Log(Info, "Expected log record query results to contain at least one hit")
		return false
	}
	if after == nil {
		return true
	}
	for _, hit := range hits.([]interface{}) {
		timestamp := Jq(hit, "_source", "@timestamp")
		t, err := time.Parse(ISO8601Layout, timestamp.(string))
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, timestamp.(string))
			if err != nil {
				Log(Error, fmt.Sprintf("Failed to parse timestamp: %s", timestamp))
				return false
			}
		}
		if t.After(*after) {
			Log(Info, fmt.Sprintf("Found recent record: %s", timestamp))
			return true
		}
		Log(Info, fmt.Sprintf("Found old record: %s", timestamp))
	}
	return false
}

// ElasticsearchHit is the type used for a Elasticsearch hit returned in a search query result
type ElasticsearchHit map[string]interface{}

// ElasticsearchHitValidator is a function that validates a hit returned in a search query result
type ElasticsearchHitValidator func(hit ElasticsearchHit) bool

// ValidateElasticsearchHits invokes the HitValidator on every hit in the searchResults.
// The first invalid hit found will return in false being returned.
// Otherwise true will be returned.
func ValidateElasticsearchHits(searchResults map[string]interface{}, hitValidator ElasticsearchHitValidator, exceptions []*regexp.Regexp) bool {
	hits := Jq(searchResults, "hits", "hits")
	if hits == nil {
		Log(Info, "Expected to find hits in log record query results")
		return false
	}
	Log(Info, fmt.Sprintf("Found %d records", len(hits.([]interface{}))))
	if len(hits.([]interface{})) == 0 {
		Log(Info, "Expected log record query results to contain at least one hit")
		return false
	}
	valid := true
	for _, h := range hits.([]interface{}) {
		hit := h.(map[string]interface{})
		src := hit["_source"].(map[string]interface{})
		log := src["log"].(string)
		if isException(log, exceptions) {
			Log(Debug, fmt.Sprintf("Exception: %s", log))
		} else {
			if !hitValidator(src) {
				valid = false
			}
		}
	}
	return valid
}

func isException(log string, exceptions []*regexp.Regexp) bool {
	for _, re := range exceptions {
		if re.MatchString(log) {
			return true
		}
	}
	return false
}

// FindLog returns true if a recent log record can be found in the index with matching filters.
func FindLog(index string, match []Match, mustNot []Match) bool {
	after := time.Now().Add(-24 * time.Hour)
	query := ElasticQuery{
		Filters: match,
		MustNot: mustNot,
	}
	result := SearchLog(index, query)
	found := findHits(result, &after)
	if !found {
		Log(Error, fmt.Sprintf("Failed to find recent log record for index %s", index))
	}
	return found
}

// FindAnyLog returns true if a log record of any time can be found in the index with matching filters.
func FindAnyLog(index string, match []Match, mustNot []Match) bool {
	query := ElasticQuery{
		Filters: match,
		MustNot: mustNot,
	}
	result := SearchLog(index, query)
	found := findHits(result, nil)
	if !found {
		Log(Error, fmt.Sprintf("Failed to find recent log record for index %s", index))
	}
	return found
}

const numberOfErrorsToLog = 5

// NoLog returns true if no matched log record can be found in the index.
func NoLog(index string, match []Match, mustNot []Match) bool {
	query := ElasticQuery{
		Filters: match,
		MustNot: mustNot,
	}
	result := SearchLog(index, query)
	if len(result) == 0 {
		return true
	}
	hits := Jq(result, "hits", "hits")
	if hits == nil || len(hits.([]interface{})) == 0 {
		return true
	}
	Log(Error, fmt.Sprintf("Found unexpected %d records", len(hits.([]interface{}))))
	for i, hit := range hits.([]interface{}) {
		if i < numberOfErrorsToLog {
			Log(Error, fmt.Sprintf("Found unexpected log record: %v", hit))
		} else {
			break
		}
	}
	return false
}

var elasticQueryTemplate *template.Template

// SearchLog search recent log records for the index with matching filters.
func SearchLog(index string, query ElasticQuery) map[string]interface{} {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil
	}
	if elasticQueryTemplate == nil {
		temp, err := template.New("esQueryTemplate").Parse(queryTemplate)
		if err != nil {
			Log(Error, fmt.Sprintf("Error: %v", err))
		}
		elasticQueryTemplate = temp
	}
	var buffer bytes.Buffer
	err = elasticQueryTemplate.Execute(&buffer, query)
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
	}
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil
	}
	var result map[string]interface{}
	url := fmt.Sprintf("%s/%s/_search", getElasticSearchURL(kubeconfigPath), index)
	username, password, err := getElasticSearchUsernamePassword(configPath)
	if err != nil {
		return result
	}
	Log(Debug, fmt.Sprintf("Search: %v \nQuery: \n%v", url, buffer.String()))
	resp, err := postElasticSearchWithBasicAuth(url, buffer.String(), username, password, configPath)
	if err != nil {
		Log(Error, fmt.Sprintf(queryErrorFormat, url, err))
		return result
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf(queryStatusFormat, url, resp.StatusCode))
		return result
	}
	json.Unmarshal(resp.Body, &result)
	return result
}

// PostElasticsearch POST the request entity body to Elasticsearch API path
// The provided path is appended to the Elasticsearch base URL
func PostElasticsearch(path string, body string) (*HTTPResponse, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil, err
	}
	url := fmt.Sprintf("%s/%s", getElasticSearchURL(kubeconfigPath), path)
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil, err
	}
	username, password, err := getElasticSearchUsernamePassword(configPath)
	if err != nil {
		return nil, err
	}
	Log(Debug, fmt.Sprintf("REST API path: %v \nQuery: \n%v", url, body))
	resp, err := postElasticSearchWithBasicAuth(url, body, username, password, configPath)
	return resp, err
}

func IndicesNotExists(patterns []string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return false
	}
	Log(Debug, fmt.Sprintf("Looking for indices in cluster using kubeconfig %s", kubeconfigPath))
	for _, name := range listSystemElasticSearchIndices(kubeconfigPath) {
		for _, pattern := range patterns {
			matched, _ := regexp.MatchString(pattern, name)
			if matched {
				Log(Error, fmt.Sprintf("Index %s matching the pattern %s still exists", name, pattern))
				return false
			}
		}
	}
	return true
}

func ISMPolicyExists(policyName string) (bool, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return false, err
	}
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return false, err
	}
	url := fmt.Sprintf("%s/_plugins/_ism/policies/%s", getElasticSearchURL(kubeconfigPath), policyName)
	resp, err := getElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		return false, err
	}
	return resp.StatusCode == http.StatusOK, nil
}

func GetISMPolicy(policyName string) (ISMPolicy, error) {
	result := ISMPolicy{}
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return result, err
	}
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return result, err
	}
	url := fmt.Sprintf("%s/_plugins/_ism/policies/%s", getElasticSearchURL(kubeconfigPath), policyName)
	resp, err := getElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		return result, err
	}
	if resp.StatusCode == http.StatusOK {
		var ismPolicyJSON map[string]ISMPolicy
		json.Unmarshal(resp.Body, &ismPolicyJSON)
		return ismPolicyJSON["policy"], nil
	}
	return result, nil
}

func GetRetentionPeriod(policyName string) (string, error) {
	ismPolicy, err := GetISMPolicy(policyName)
	if err != nil {
		return "", err
	}
	for _, state := range ismPolicy.States {
		if state.Name == "ingest" {
			for _, transition := range state.Transitions {
				if transition.StateName == "delete" {
					minIndexAge := transition.Conditions["min_index_age"]
					return minIndexAge, nil
				}
			}
		}
	}
	return "", nil
}

func GetISMRolloverPeriod(policyName string) (string, error) {
	ismPolicy, err := GetISMPolicy(policyName)
	if err != nil {
		return "", err
	}
	for _, state := range ismPolicy.States {
		if state.Name == "ingest" {
			for _, action := range state.Actions {
				rolloverPeriod := action.Rollover.MinIndexAge
				return rolloverPeriod, nil
			}
		}
	}
	return "", nil
}

func CheckForDataStream(name string) bool {
	url := getDataStreamURLFormat + name
	resp, err := doGetElasticSearchURL(url)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting Elasticsearch data streams: url=%s, error=%v", url, err))
		return false
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error retrieving Elasticsearch data streams: url=%s, status=%d", url, resp.StatusCode))
		return false
	}
	return true
}

// ElasticQuery describes an Elasticsearch Query
type ElasticQuery struct {
	Filters []Match
	MustNot []Match
}

// Match describes a match_phrase in Elasticsearch Query
type Match struct {
	Key   string
	Value string
}

const queryTemplate = `{
  "size": 100,
  "sort": [
    {
      "@timestamp": {
        "order": "desc",
        "unmapped_type": "boolean"
      }
    }
  ],
  "query": {
    "bool": {
      "filter": [
        {
          "match_all": {}
        }
{{range $filter := .Filters}} 
		,
        {
          "match_phrase": {
            "{{$filter.Key}}": "{{$filter.Value}}"
          }
        }
{{end}}
      ],
      "must_not": [
{{range $index, $mustNot := .MustNot}} 
        {{if $index}},{{end}}
        {
          "match_phrase": {
            "{{$mustNot.Key}}": "{{$mustNot.Value}}"
          }
        }
{{end}}
      ]
    }
  }
}
`
