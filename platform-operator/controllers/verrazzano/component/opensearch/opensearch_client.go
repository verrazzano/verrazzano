// Copyright (C) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

type (
	OSClient struct {
		httpClient *http.Client
		DoHTTP     func(request *http.Request) (*http.Response, error)
	}
	ClusterHealth struct {
		Status string `json:"status"`
	}
)

const (
	indexSettings     = `{"index_patterns": [".opendistro*"],"priority": 0,"template": {"settings": {"auto_expand_replicas": "0-1"}}}`
	applicationJSON   = "application/json"
	contentTypeHeader = "Content-Type"
	HealthGreen       = "green"
)

func NewOSClient(pas string) *OSClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec //#gosec G402
	}
	o := &OSClient{
		httpClient: &http.Client{Transport: tr},
	}
	o.DoHTTP = func(request *http.Request) (*http.Response, error) {
		request.SetBasicAuth("verrazzano", pas)
		return o.httpClient.Do(request)
	}
	return o
}

// ConfigureISM sets up the ISM Policies
func (o *OSClient) ConfigureISM(log vzlog.VerrazzanoLogger, client clipkg.Client, vz *vzv1alpha1.Verrazzano) error {
	if !*vz.Spec.Components.Elasticsearch.Enabled {
		return nil
	}
	if !o.IsOpenSearchReady(client) {
		return nil
	}
	opensearchEndpoint, err := GetOpenSearchHTTPEndpoint(client)
	if err != nil {
		return err
	}
	for _, policy := range vz.Spec.Components.Elasticsearch.Policies {
		if err := o.createISMPolicy(opensearchEndpoint, policy); err != nil {
			return err
		}
	}
	o.cleanupPolicies(opensearchEndpoint, vz.Spec.Components.Elasticsearch.Policies)

	return nil
}

// DeleteDefaultISMPolicy deletes the default ISM policy if they exists
func (o *OSClient) DeleteDefaultISMPolicy(log vzlog.VerrazzanoLogger, client clipkg.Client, vz *vzv1alpha1.Verrazzano) error {
	// if Elasticsearch.DisableDefaultPolicy is set to false, skip the deletion.
	if !*vz.Spec.Components.Elasticsearch.Enabled || !vz.Spec.Components.Elasticsearch.DisableDefaultPolicy {
		return nil
	}
	if !o.IsOpenSearchReady(client) {
		return nil
	}

	openSearchEndpoint, err := GetOpenSearchHTTPEndpoint(client)
	if err != nil {
		return err
	}
	for policyName := range defaultISMPoliciesMap {
		resp, err := o.deletePolicy(openSearchEndpoint, policyName)
		// If policy doesn't exist, ignore the error, otherwise return error.
		if (err != nil && resp == nil) || (err != nil && resp != nil && resp.StatusCode != http.StatusNotFound) {
			return err
		}
		// Remove the policy from the current write indices of system and application data streams
		var pattern string
		if policyName == "vz-system" {
			pattern = "verrazzano-system"
		} else {
			pattern = "verrazzano-application-*"
		}
		indices, err := o.getWriteIndexForDataStream(log, openSearchEndpoint, pattern)
		if err != nil {
			return err
		}
		for _, index := range indices {
			ok, err := o.shouldAddOrRemoveDefaultPolicy(openSearchEndpoint, index, policyName)
			if err != nil {
				return err
			}
			if ok {
				err = o.removePolicyForIndex(openSearchEndpoint, index)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// SyncDefaultISMPolicy set up the default ISM Policies.
func (o *OSClient) SyncDefaultISMPolicy(log vzlog.VerrazzanoLogger, client clipkg.Client, vz *vzv1alpha1.Verrazzano) error {
	if !*vz.Spec.Components.Elasticsearch.Enabled || vz.Spec.Components.Elasticsearch.DisableDefaultPolicy {
		return nil
	}
	if !o.IsOpenSearchReady(client) {
		return nil
	}
	openSearchEndpoint, err := GetOpenSearchHTTPEndpoint(client)
	if err != nil {
		return err
	}
	_, err = o.createOrUpdateDefaultISMPolicy(log, openSearchEndpoint)
	return err
}

// SetAutoExpandIndices updates the default index settings to auto expand replicas (max 1) when nodes are added to the cluster
func (o *OSClient) SetAutoExpandIndices(log vzlog.VerrazzanoLogger, client clipkg.Client, vz *vzv1alpha1.Verrazzano) error {
	openSearchEndpoint, err := GetOpenSearchHTTPEndpoint(client)
	if err != nil {
		return err
	}
	settingsURL := fmt.Sprintf("%s/_index_template/ism-plugin-template", openSearchEndpoint)
	req, err := http.NewRequest("PUT", settingsURL, bytes.NewReader([]byte(indexSettings)))
	if err != nil {
		return err
	}
	req.Header.Add(contentTypeHeader, applicationJSON)
	resp, err := o.DoHTTP(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status code %d when updating default settings of index, expected %d", resp.StatusCode, http.StatusOK)
	}
	var updatedIndexSettings map[string]bool
	err = json.NewDecoder(resp.Body).Decode(&updatedIndexSettings)
	if err != nil {
		return err
	}
	if !updatedIndexSettings["acknowledged"] {
		return fmt.Errorf("expected acknowledgement for index settings update but did not get. Actual response  %v", updatedIndexSettings)
	}
	return nil
}

// IsOpenSearchReady returns true when all OpenSearch pods are ready, false otherwise
func (o *OSClient) IsOpenSearchReady(client clipkg.Client) bool {
	statefulSets := appsv1.StatefulSetList{}
	listOptions := clipkg.ListOptions{Namespace: "verrazzano-logging"}
	if err := client.List(context.TODO(), &statefulSets, &listOptions); err != nil {
		if errors.IsNotFound(err) {
			// StatefulSet not found
			return false
		}
		return false
	}
	if len(statefulSets.Items) == 0 {
		zap.S().Warn("waiting for OpenSearch statefulset to be created.")
		return false
	}

	return statefulSets.Items[0].Status.ReadyReplicas == statefulSets.Items[0].Status.Replicas
}

// IsClusterHealthy checks whether Opensearch Cluster is healthy or not.
func (o *OSClient) IsClusterHealthy(client clipkg.Client) (bool, error) {
	openSearchEndpoint, err := GetOpenSearchHTTPEndpoint(client)
	if err != nil {
		return false, err
	}
	healthURL := fmt.Sprintf("%s/_cluster/health", openSearchEndpoint)
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Add(contentTypeHeader, applicationJSON)
	resp, err := o.DoHTTP(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("got status code %d when fetching cluster health, expected %d", resp.StatusCode, http.StatusOK)
	}
	clusterHealth := &ClusterHealth{}
	if err := json.NewDecoder(resp.Body).Decode(clusterHealth); err != nil {
		return false, err
	}
	return clusterHealth.Status == HealthGreen, nil
}

func GetOpenSearchHTTPEndpoint(client clipkg.Client) (string, error) {
	opensearchURL, err := k8sutil.GetURLForIngress(client, "opensearch", "verrazzano-system", "https")
	if err != nil {
		return "", err
	}
	return opensearchURL, nil
}

func GetOSDHTTPEndpoint(client clipkg.Client) (string, error) {
	osdURL, err := k8sutil.GetURLForIngress(client, "opensearch-dashboards", "verrazzano-system", "https")
	if err != nil {
		return "", err
	}
	return osdURL, nil
}

// GetVerrazzanoPassword returns the password credential for the Verrazzano secret
func GetVerrazzanoPassword(client clipkg.Client) (string, error) {
	var secret = &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: "verrazzano", Namespace: "verrazzano-system"}, secret)
	if err != nil {
		return "", fmt.Errorf("unable to fetch secret %s/%s, %v", "verrazzano", "verrazzano-system", err)
	}
	return string(secret.Data["password"]), nil
}
