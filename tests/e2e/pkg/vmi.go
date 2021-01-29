// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	NUM_RETRIES    = 7
	RETRY_WAIT_MIN = 1 * time.Second
	RETRY_WAIT_MAX = 30 * time.Second
)

// GetSystemVMICredentials - Obtain VMI system credentials
func GetSystemVMICredentials() (*UsernamePassword, error) {
	vmi, err := GetVerrazzanoMonitoringInstance("verrazzano-system", "system")
	if err != nil {
		return nil, fmt.Errorf("error getting system VMI: %w", err)
	}

	secret, err := GetSecret("verrazzano-system", vmi.Spec.SecretsName)
	if err != nil {
		return nil, err
	}

	username := secret.Data["username"]
	password := secret.Data["password"]
	if username == nil || password == nil {
		return nil, fmt.Errorf("username and password fields required in secret %v", secret)
	}

	return &UsernamePassword{
		Username: string(username),
		Password: string(password),
	}, nil
}

func newRetryableHttpClient(client *http.Client) *retryablehttp.Client {
	retryableClient := retryablehttp.NewClient() //default of 4 retries is sufficient for us
	retryableClient.RetryMax = NUM_RETRIES
	retryableClient.RetryWaitMin = RETRY_WAIT_MIN
	retryableClient.RetryWaitMax = RETRY_WAIT_MAX
	retryableClient.HTTPClient = client
	return retryableClient
}


func GetBindingVmiHttpClient(bindingName string) *retryablehttp.Client {
	bindingVmiCaCert := getBindingVMICACert(bindingName)
	vmiRawClient := getHttpClientWIthCABundle(bindingVmiCaCert)
	return newRetryableHttpClient(vmiRawClient)
}

func getBindingVMICACert(bindingName string) []byte {
	return doGetCACertFromSecret(fmt.Sprintf("%v-tls", bindingName), "verrazzano-system")
}

func getHttpClientWIthCABundle(caData []byte) *http.Client {
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCertPool(caData)}}

	proxyURL := getProxyURL()
	if proxyURL != "" {
		tURL := url.URL{}
		tURLProxy, _ := tURL.Parse(proxyURL)
		tr.Proxy = http.ProxyURL(tURLProxy)
	}

	ipResolve := getManagementClusterNodeIP()
	if ipResolve != "" {
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			Log(Debug, fmt.Sprintf("original address %s", addr))
			if strings.Contains(addr, "127.0.0.1") && strings.Contains(addr, ":443") {
				addr = ipResolve + ":443"
				Log(Debug, fmt.Sprintf("modified address %s", addr))
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	return &http.Client{Transport: tr}
}

// If testing against KIND, returns the control-plane node ip ; "" otherwise
func getManagementClusterNodeIP() string {
	pods := ListPods("ingress-nginx")
	for i := range pods.Items {
		pod := pods.Items[i]
		if strings.HasPrefix(pod.Name, "ingress-controller-ingress-nginx-controller-") {
			return pod.Status.HostIP
		}
	}

	return ""
}
