// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

const (
	HealthGreen       = "green"
	applicationJSON   = "application/json"
	contentTypeHeader = "Content-Type"
	vzSystemNamespace = "verrazzano-system"
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

// AreOpensearchStsReady Check that the OS statefulsets have the minimum number of specified replicas ready and available. It ignores the updated replicas check if updated replicas are zero or cluster is not healthy.
func AreOpensearchStsReady(log vzlog.VerrazzanoLogger, client client.Client, namespacedNames []types.NamespacedName, expectedReplicas int32, prefix string) bool {
	for _, namespacedName := range namespacedNames {
		statefulset := appsv1.StatefulSet{}
		if err := client.Get(context.TODO(), namespacedName, &statefulset); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for statefulset %v to exist", prefix, namespacedName)
				// StatefulSet not found
				return false
			}
			log.Errorf("Failed getting statefulset %v: %v", namespacedName, err)
			return false
		}
		if !areOSReplicasUpdated(log, statefulset, expectedReplicas, client, prefix, namespacedName) {
			return false
		}
		if statefulset.Status.ReadyReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current ready replicas is %v", prefix, namespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
		log.Oncef("%s has enough ready replicas for statefulsets %v", prefix, namespacedName)
	}
	return true
}

// GetVerrazzanoPassword returns the password credential for the Verrazzano secret
func GetVerrazzanoPassword(client client.Client) (string, error) {
	var secret = &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: "verrazzano", Namespace: vzSystemNamespace}, secret)
	if err != nil {
		return "", fmt.Errorf("unable to fetch secret %s/%s, %v", "verrazzano", vzSystemNamespace, err)
	}
	return string(secret.Data["password"]), nil
}

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

// IsClusterHealthy checks whether Opensearch Cluster is healthy or not.
func (o *OSClient) IsClusterHealthy(client client.Client) (bool, error) {
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

func GetOpenSearchHTTPEndpoint(client client.Client) (string, error) {
	opensearchURL, err := k8sutil.GetURLForIngress(client, "opensearch", vzSystemNamespace, "https")
	if err != nil {
		return "", err
	}
	return opensearchURL, nil
}

// areOSReplicasUpdated check whether all replicas of opensearch are updated or not. In case of yellow cluster status, we skip this check and consider replicas are updated.
func areOSReplicasUpdated(log vzlog.VerrazzanoLogger, statefulset appsv1.StatefulSet, expectedReplicas int32, client client.Client, prefix string, namespacedName types.NamespacedName) bool {
	if statefulset.Status.UpdatedReplicas > 0 && statefulset.Status.UpdateRevision != statefulset.Status.CurrentRevision && statefulset.Status.UpdatedReplicas < expectedReplicas {
		pas, err := GetVerrazzanoPassword(client)
		if err != nil {
			log.Errorf("Failed getting OS secret to check OS cluster health: %v", err)
			return false
		}
		osClient := NewOSClient(pas)
		healthy, err := osClient.IsClusterHealthy(client)
		if err != nil {
			log.Errorf("Failed getting Opensearch cluster health: %v", err)
			return false
		}
		if !healthy {
			log.Errorf("Opensearch Cluster is not healthy. Please check Opensearch operator for more information")
			return true
		}
		log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current updated replicas is %v", prefix, namespacedName,
			expectedReplicas, statefulset.Status.UpdatedReplicas)
		return false
	}
	return true
}
