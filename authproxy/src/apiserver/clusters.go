// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"k8s.io/apimachinery/pkg/types"
)

const (
	kubernetesAPIServerHostname = "kubernetes.default.svc.cluster.local"
	localClusterPrefix          = "/clusters/local"
)

// reformatLocalClusterRequest reformats a local cluster request
func (a *APIRequest) reformatLocalClusterRequest(req *http.Request) (*retryablehttp.Request, error) {
	req.Host = kubernetesAPIServerHostname
	err := a.reformatClusterPath(req, a.APIServerURL, "local", "")
	if err != nil {
		return nil, err
	}

	err = setImpersonationHeaders(req)
	if err != nil {
		a.Log.Errorf("Failed to set impersonation headers for request: %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.BearerToken)

	retryableReq, err := retryablehttp.FromRequest(req)
	if err != nil {
		a.Log.Errorf("Failed to convert reformatted request to a retryable request: %v", err)
		return retryableReq, err
	}
	return retryableReq, nil
}

// reformatManagedClusterRequest formats a request to a managed cluster
func (a *APIRequest) reformatManagedClusterRequest(req *http.Request, clusterName string) (*retryablehttp.Request, error) {
	apiURL, err := a.getManagedClusterAPIURL(clusterName)
	if err != nil {
		return nil, err
	}

	err = a.reformatClusterPath(req, apiURL, clusterName, localClusterPrefix)
	if err != nil {
		return nil, err
	}

	retryableReq, err := retryablehttp.FromRequest(req)
	if err != nil {
		a.Log.Errorf("Failed to convert reformatted request to a retryable request: %v", err)
		return retryableReq, err
	}
	return retryableReq, nil
}

// reformatClusterPath reformats the cluster path given request data
func (a *APIRequest) reformatClusterPath(req *http.Request, apiServerURL, clusterName, newClusterPrefix string) error {
	req.RequestURI = ""

	path := strings.Replace(req.URL.Path, fmt.Sprintf("/clusters/%s", clusterName), newClusterPrefix, 1)
	newReq, err := url.JoinPath(apiServerURL, path)
	if err != nil {
		a.Log.Errorf("Failed to format request path for path %s: %v", path, err)
		return err
	}

	formattedURL, err := url.Parse(newReq)
	if err != nil {
		a.Log.Errorf("Failed to format incoming url: %v", err)
		return err
	}
	formattedURL.RawQuery = req.URL.RawQuery
	req.URL = formattedURL
	return nil
}

// getManagedClusterAPIURL returns the API URL for the managed cluster given the cluster name
func (a *APIRequest) getManagedClusterAPIURL(clusterName string) (string, error) {
	var vmc v1alpha1.VerrazzanoManagedCluster
	err := a.K8sClient.Get(context.TODO(), types.NamespacedName{Name: clusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}, &vmc)
	if err != nil {
		a.Log.Errorf("Failed to get the Verrazzano Managed Cluster resource from the cluster: %v", err)
		return "", err
	}

	if vmc.Status.APIUrl == "" {
		return "", fmt.Errorf("could not find API URL from the VMC status")
	}

	return vmc.Status.APIUrl, nil
}

// getClusterName returns the cluster name given an API Server request
// this should have been pre-validated, so we can assume it has a prefixed cluster path and has the format
// /cluster/cluster-name/api-request-path
func getClusterName(req *http.Request) (string, error) {
	path := req.URL.Path

	splitPath := strings.Split(path, "/")
	if len(splitPath) < 3 {
		return "", fmt.Errorf("no cluster name was provided in the path")
	}

	return splitPath[2], nil
}
