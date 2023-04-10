// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/verrazzano/verrazzano/pkg/constants"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MetricSource implements an interface to interact with metrics sources
type MetricSource interface {
	GetHost() string
	GetTargets() ([]interface{}, error)
	getKubeConfigPath() string
}

// metricSourceBase implements shared functions between metric sources
type metricSourceBase struct {
	client         *kubernetes.Clientset
	kubeconfigPath string
}

// ThanosSource provides a struct to interact with the Thanos metric source
type ThanosSource struct {
	metricSourceBase
}

// PrometheusSource provides a struct to interact with Prometheus metric source
type PrometheusSource struct {
	metricSourceBase
}

var _ MetricSource = ThanosSource{}
var _ MetricSource = PrometheusSource{}

func newMetricsSourceBase(kubeconfigPath string) (metricSourceBase, error) {
	cli, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	return metricSourceBase{
		kubeconfigPath: kubeconfigPath,
		client:         cli,
	}, err
}

func NewThanosSource(kubeconfigPath string) (ThanosSource, error) {
	msb, err := newMetricsSourceBase(kubeconfigPath)
	return ThanosSource{metricSourceBase: msb}, err
}

func NewPrometheusSource(kubeconfigPath string) (PrometheusSource, error) {
	msb, err := newMetricsSourceBase(kubeconfigPath)
	return PrometheusSource{metricSourceBase: msb}, err
}

func (m metricSourceBase) getKubeConfigPath() string {
	return m.kubeconfigPath
}

// getHostByName returns the hostname given the ingress name
func (m metricSourceBase) getHostByIngressName(name string) string {
	ingress, err := m.client.NetworkingV1().Ingresses(constants.VerrazzanoSystemNamespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed get Thanos Frontend Ingress %s from the cluster: %v", constants.ThanosQueryIngress, err))
		return ""
	}
	return ingress.Spec.Rules[0].Host
}

// GetHost returns the host for the Thanos ingress
func (t ThanosSource) GetHost() string {
	return t.metricSourceBase.getHostByIngressName(constants.ThanosQueryIngress)
}

// GetHost returns the host for the Prometheus ingress
func (p PrometheusSource) GetHost() string {
	return p.metricSourceBase.getHostByIngressName(vzconst.PrometheusIngress)
}

// getTargetsByHostAndPath returns an unstructred target interface given the host and the path
func (m metricSourceBase) getTargetsByHostAndPath(host, path string, jsonPath []string) ([]interface{}, error) {
	metricsURL := fmt.Sprintf("https://%s%s", host, path)
	password, err := GetVerrazzanoPasswordInCluster(m.kubeconfigPath)
	if err != nil {
		return nil, err
	}
	resp, err := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", password, m.kubeconfigPath)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error retrieving targets %d", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body, &result)
	if err != nil {
		return nil, err
	}
	queryStores, ok := Jq(result, jsonPath...).([]interface{})
	if !ok {
		return nil, fmt.Errorf("error finding query store in the Thanos store list")
	}
	return queryStores, nil
}

// GetTargets returns the Thanos store targets
func (t ThanosSource) GetTargets() ([]interface{}, error) {
	return t.metricSourceBase.getTargetsByHostAndPath(t.GetHost(), "/api/v1/stores", []string{"data", "query"})
}

// GetTargets returns the Prometheus targets
func (p PrometheusSource) GetTargets() ([]interface{}, error) {
	return p.metricSourceBase.getTargetsByHostAndPath(p.GetHost(), "/api/v1/targets", []string{"data", "activeTargets"})
}
