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
	"github.com/verrazzano/verrazzano/authproxy/internal/httputil"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/cert"
)

const (
	kubernetesAPIServerHostname = "kubernetes.default.svc.cluster.local"
	localClusterPrefix          = "/clusters/local"
	caCertKey                   = "cacrt"
)

// reformatLocalClusterRequest reformats a local cluster request
func (a *APIRequest) reformatLocalClusterRequest(req *http.Request) (*retryablehttp.Request, error) {
	req.Host = kubernetesAPIServerHostname
	err := a.reformatClusterPath(req, "local", "")
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
	err := a.processManagedClusterResources(clusterName)
	if err != nil {
		return nil, err
	}

	err = a.reformatClusterPath(req, clusterName, localClusterPrefix)
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
func (a *APIRequest) reformatClusterPath(req *http.Request, clusterName, newClusterPrefix string) error {
	req.RequestURI = ""

	path := strings.Replace(req.URL.Path, fmt.Sprintf("/clusters/%s", clusterName), newClusterPrefix, 1)
	newReq, err := url.JoinPath(a.APIServerURL, path)
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
	req.Host = formattedURL.Hostname()
	return nil
}

// processManagedClusterResources uses the Verrazzano Managed Cluster object to process and edit the request resources
func (a *APIRequest) processManagedClusterResources(clusterName string) error {
	var vmc v1alpha1.VerrazzanoManagedCluster
	err := a.K8sClient.Get(context.TODO(), types.NamespacedName{Name: clusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}, &vmc)
	if err != nil {
		a.Log.Errorf("Failed to get the Verrazzano Managed Cluster resource from the cluster: %v", err)
		return err
	}

	err = a.setManagedClusterAPIURL(vmc)
	if err != nil {
		return err
	}

	return a.rewriteClientCACerts(vmc)
}

// getManagedClusterAPIURL returns the API URL for the managed cluster given the cluster name
func (a *APIRequest) setManagedClusterAPIURL(vmc v1alpha1.VerrazzanoManagedCluster) error {
	if vmc.Status.APIUrl == "" {
		return fmt.Errorf("could not find API URL from the VMC status")
	}
	a.APIServerURL = vmc.Status.APIUrl
	return nil
}

// rewriteClientCACerts generates a new client with the managed CA certs
// The client CA certs will need to be updated to use the certificates for the managed cluster ingress
func (a *APIRequest) rewriteClientCACerts(vmc v1alpha1.VerrazzanoManagedCluster) error {
	if vmc.Spec.CASecret == "" {
		return fmt.Errorf("could not find CA secret name from the VMC spec")
	}

	var caSecret v1.Secret
	err := a.K8sClient.Get(context.TODO(), types.NamespacedName{Name: vmc.Spec.CASecret, Namespace: constants.VerrazzanoMultiClusterNamespace}, &caSecret)
	if err != nil {
		a.Log.Errorf("Failed to get the Verrazzano Managed Cluster resource from the cluster: %v", err)
		return err
	}

	caData, ok := caSecret.Data[caCertKey]
	if !ok {
		err = fmt.Errorf("CA data empty in the secret %s", caSecret.Name)
		a.Log.Errorf("Failed to CA data from the cluster: %v", err)
		return err
	}

	rootCA, err := cert.NewPoolFromBytes(caData)
	if err != nil {
		a.Log.Errorf("Failed to get in cluster Root Certificate for the Kubernetes API server")
		return err
	}

	a.Client = httputil.GetHTTPClientWithCABundle(rootCA)
	return nil
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
