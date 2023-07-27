// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

// GetRulesFromThanosRuler returns the rule data from the Thanos Ruler API given a kubeconfig
func GetRulesFromThanosRuler(kubeconfigPath string) (interface{}, error) {
	retryableClient, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed get an HTTP client: %v", err))
		return nil, err
	}

	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get the Verrazzano password from the cluster %v", err))
		return nil, err
	}

	url := GetSystemThanosRulerURL(kubeconfigPath)
	resp, err := doReq(url, "GET", "", "", "verrazzano", password, nil, retryableClient)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to make a request to the Thanos Ruler ingress %v", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Failed to get an OK response from %s, response: %d", url, resp.StatusCode))
		return nil, err
	}

	type ruleData struct {
		Data interface{} `json:"data"`
	}
	var data ruleData
	err = json.Unmarshal(resp.Body, &data)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to unmarshal the response data %s: %v", data, err))
		return nil, err
	}

	return data, err
}

// GetSystemThanosRulerURL gets the system Thanos Ingress host in the given cluster
func GetSystemThanosRulerURL(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingress, err := clientset.NetworkingV1().Ingresses(VerrazzanoNamespace).Get(context.TODO(), "thanos-ruler", metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to find Thanos Ruler Ingress %v", err))
		return ""
	}

	Log(Info, fmt.Sprintf("Found Thanos Ruler Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
}
