// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"net/http"
	"net/url"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	clusterapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/argocd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	argocdAdminSecret = "argocd-initial-admin-secret" //nolint:gosec //#gosec G101

	clustersAPIPath   = "/api/v1/clusters"
	sessionPath       = "/api/v1/session"
	serviceURL        = "argocd-server.argocd.svc"
	argoUserName      = "service-argo"
	clusterSecretName = "cluster-secret"
)

type ArgoCDConfig struct {
	Host                     string
	BaseURL                  string
	APIAccessToken           string
	CertificateAuthorityData []byte
	AdditionalCA             []byte
}

var DefaultRetry = wait.Backoff{
	Steps:    10,
	Duration: 1 * time.Second,
	Factor:   2.0,
	Jitter:   0.1,
}

// requestSender is an interface for sending requests to Rancher that allows us to mock during unit testing
type requestSender interface {
	Do(httpClient *http.Client, req *http.Request) (*http.Response, error)
}

// HTTPRequestSender is an implementation of requestSender that uses http.Client to send requests
type HTTPRequestSender struct{}

var ArgoCDHTTPClient requestSender = &HTTPRequestSender{}

// Do is a function that simply delegates sending the request to the http.Client
func (*HTTPRequestSender) Do(httpClient *http.Client, req *http.Request) (*http.Response, error) {
	return httpClient.Do(req)
}

func (r *VerrazzanoManagedClusterReconciler) isArgoCDEnabled() bool {
	vz, _ := r.getVerrazzanoResource()
	return vzcr.IsArgoCDEnabled(vz)
}

func (r *VerrazzanoManagedClusterReconciler) isRancherEnabled() bool {
	vz, _ := r.getVerrazzanoResource()
	return vzcr.IsRancherEnabled(vz)
}

// registerManagedClusterWithArgoCD calls the ArgoCD api to register a managed cluster with ArgoCD
func (r *VerrazzanoManagedClusterReconciler) registerManagedClusterWithArgoCD(vmc *clusterapi.VerrazzanoManagedCluster) (*clusterapi.ArgoCDRegistration, error) {
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		msg := "Waiting for Rancher manifest to be applied on the managed cluster"
		return newArgoCDRegistration(clusterapi.RegistrationPendingRancher, msg), nil
	}

	//if vmc.Status.ArgoCDRegistration.Status == clusterapi.RegistrationMCResourceCreationCompleted || vmc.Status.ArgoCDRegistration.Status == clusterapi.MCRegistrationFailed || vmc.Status.ArgoCDRegistration.Status != clusterapi.MCRegistrationCompleted {
	if vmc.Status.ArgoCDRegistration.Status == clusterapi.RegistrationPendingRancher || vmc.Status.ArgoCDRegistration.Status == clusterapi.MCRegistrationFailed || vmc.Status.ArgoCDRegistration.Status != clusterapi.MCRegistrationCompleted {
		// If the managed cluster is not active, we should not attempt to register in Argo CD
		rc, err := rancherutil.NewAdminRancherConfig(r.Client, r.log)
		if err != nil || rc == nil {
			msg := "Could not create rancher config that authenticates with the admin user"
			return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr(msg, err)
		}
		isActive, err := isManagedClusterActiveInRancher(rc, clusterID, r.log)
		if err != nil || !isActive {
			msg := fmt.Sprintf("Waiting for managed cluster with id %s to become active before registering in Argo CD", clusterID)
			return newArgoCDRegistration(clusterapi.RegistrationPendingRancher, msg), nil
		}

		vz, err := r.getVerrazzanoResource()
		if err != nil {
			msg := "Could not find Verrazzano resource"
			return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to find Verrazzano resource on admin cluster: %v", err)
		}
		if vz.Status.VerrazzanoInstance == nil {
			msg := "No instance information found in Verrazzano resource status"
			return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to find instance information in Verrazzano resource status")
		}
		var rancherURL = *(vz.Status.VerrazzanoInstance.RancherURL) + k8sClustersPath + clusterID

		//TODO: Currently use Rancher root CA secret in cattle-system namespace. argocd won't work if rancher is disabled.
		//Use Verrazzano root ca so we support public CA?
		caCert, err := common.GetRootCA(r.Client)
		if err != nil {
			msg := "Failed to get Argo CD TLS CA"
			return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to get Argo CD TLS CA")
		}

		ac, err := newArgoCDConfig(r.Client, r.log)
		if err != nil {
			msg := "Failed to create ArgoCD API client"
			return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to connect to Argo CD API on admin cluster: %v", err)
		}

		isRegistered, err := isManagedClusterAlreadyExist(ac, vmc.Name, r.log)
		if err != nil {
			msg := "Failed to call Argo CD clusters GET API"
			return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to call Argo CD clusters GET API on admin cluster: %v", err)
		}
		if isRegistered {
			msg := "Cluster is already registered in Argo CD"
			return newArgoCDRegistration(clusterapi.MCRegistrationCompleted, msg), nil
		}

		err = r.argocdClusterAdd(vmc, caCert, rancherURL)
		if err != nil {
			msg := "Failed to create Argo CD cluster secret"
			return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to call Argo CD clusters POST API on admin cluster: %v", err)
		}
		msg := "Successfully registered managed cluster in ArgoCD"
		return newArgoCDRegistration(clusterapi.MCRegistrationCompleted, msg), nil
	}

	return nil, nil
}

type ClusterList struct {
	Items []struct {
		Server string `json:"server"`
		Name   string `json:"name"`
	} `json:"items"`
}

// isManagedClusterAlreadyExist returns true if the managed cluster does exist
func isManagedClusterAlreadyExist(ac *ArgoCDConfig, clusterName string, log vzlog.VerrazzanoLogger) (bool, error) {
	reqURL := "https://" + ac.Host + clustersAPIPath
	headers := map[string]string{"Authorization": "Bearer " + ac.APIAccessToken}

	response, responseBody, err := sendHTTPRequest(http.MethodGet, reqURL, headers, "", ac, log)

	if response != nil && response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("tried to get cluster from Rancher but failed, response code: %d", response.StatusCode)
	}

	if err != nil {
		return false, err
	}

	clusters := &ClusterList{}
	json.Unmarshal([]byte(responseBody), clusters)
	for _, item := range clusters.Items {
		if item.Name == clusterName {
			return true, nil
		}
	}

	return false, nil
}

// argocdClusterAdd registers cluster using the Rancher Proxy by creating a user in rancher, with api token and cluster roles set, and a secret containing Rancher proxy for the cluster
func (r *VerrazzanoManagedClusterReconciler) argocdClusterAdd(vmc *clusterapi.VerrazzanoManagedCluster, caCert []byte, rancherURL string) error {
	r.log.Debugf("Configuring Rancher user for cluster registration in ArgoCD")

	secret, err := GetArgoCDClusterUserSecret(r.Client)
	if err != nil {
		return nil
	}
	rc, err := rancherutil.NewRancherConfigForUser(r.Client, vzconst.ArgoCDClusterRancherUsername, secret, r.log)
	//rc, err := rancherutil.NewAdminRancherConfig(r.Client, r.log)
	if err != nil {
		return err
	}

	// create/update the cluster secret with the rancher config
	_, err = r.createClusterSecret(rc.APIAccessToken, vmc, rancherURL, caCert)
	if err != nil {
		return err
	}

	r.log.Oncef("Successfully registered managed cluster in ArgoCD with name: %s", vmc.Name)
	return nil
}

// GetArgoCDClusterUserSecret fetches the Rancher Verrazzano user secret
func GetArgoCDClusterUserSecret(rdr client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      vzconst.ArgoCDClusterRancherName}

	if err := rdr.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

type TLSClientConfig struct {
	CaData   string `json:"caData"`
	Insecure bool   `json:"insecure"`
}

type RancherConfig struct {
	BearerToken     string `json:"bearerToken"`
	TLSClientConfig `json:"tlsClientConfig"`
}

func (r *VerrazzanoManagedClusterReconciler) createClusterSecret(token string, vmc *clusterapi.VerrazzanoManagedCluster, rancherURL string, caData []byte) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Name = vmc.Name + "-" + clusterSecretName
	secret.Namespace = constants.ArgoCDNamespace

	// Create or update on the local cluster
	return controllerruntime.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateClusterSecret(&secret, token, vmc.Name, rancherURL, caData)
		return nil
	})
}

func (r *VerrazzanoManagedClusterReconciler) mutateClusterSecret(secret *corev1.Secret, token string, cluserName string, rancherURL string, caData []byte) error {
	if secret.StringData == nil {
		secret.StringData = make(map[string]string)
	}
	secret.Type = corev1.SecretTypeOpaque
	secret.ObjectMeta.Labels = map[string]string{"argocd.argoproj.io/secret-type": "cluster"}
	secret.StringData["name"] = cluserName
	secret.StringData["server"] = rancherURL

	rancherConfig := &RancherConfig{
		BearerToken: token,
		TLSClientConfig: TLSClientConfig{
			CaData:   base64.StdEncoding.EncodeToString(caData),
			Insecure: false},
	}
	data, err := json.Marshal(rancherConfig)
	if err != nil {
		return err
	}
	secret.StringData["config"] = string(data)

	return nil
}

// updateRancherClusterRoleBindingTemplate creates a new ClusterRoleBindingTemplate for the given VMC
// to grant the Verrazzano cluster user the correct permissions on the managed cluster
func (r *VerrazzanoManagedClusterReconciler) updateArgoCDClusterRoleBindingTemplate(vmc *clusterapi.VerrazzanoManagedCluster) error {
	if vmc == nil {
		r.log.Debugf("Empty VMC, no ClusterRoleBindingTemplate created")
		return nil
	}

	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		r.log.Progressf("Waiting to create ClusterRoleBindingTemplate for cluster %s, Rancher ClusterID not found in the VMC status", vmc.GetName())
		return nil
	}

	userID, err := r.getArgoCDClusterUserID()
	if err != nil {
		return err
	}

	name := fmt.Sprintf("crtb-argocd-%s", clusterID)
	nsn := types.NamespacedName{Name: name, Namespace: clusterID}
	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   APIGroupRancherManagement,
		Version: APIGroupVersionRancherManagement,
		Kind:    ClusterRoleTemplateBindingKind,
	})
	resource.SetName(nsn.Name)
	resource.SetNamespace(nsn.Namespace)
	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &resource, func() error {
		data := resource.UnstructuredContent()
		data[ClusterRoleTemplateBindingAttributeClusterName] = clusterID
		data[ClusterRoleTemplateBindingAttributeUserName] = userID
		data[ClusterRoleTemplateBindingAttributeRoleTemplateName] = "cluster-owner"
		return nil
	})
	if err != nil {
		return r.log.ErrorfThrottledNewErr("Failed configuring %s %s: %s", ClusterRoleTemplateBindingKind, nsn.Name, err.Error())
	}
	return nil
}

// getArgoCDClusterUserID returns the Rancher-generated user ID for the Verrazzano argocd cluster user
func (r *VerrazzanoManagedClusterReconciler) getArgoCDClusterUserID() (string, error) {
	usersList := unstructured.UnstructuredList{}
	usersList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   APIGroupRancherManagement,
		Version: APIGroupVersionRancherManagement,
		Kind:    UserListKind,
	})
	err := r.Client.List(context.TODO(), &usersList, &client.ListOptions{})
	if err != nil {
		return "", r.log.ErrorfNewErr("Failed to list Rancher Users: %v", err)
	}

	for _, user := range usersList.Items {
		userData := user.UnstructuredContent()
		if userData[UserUsernameAttribute] == vzconst.ArgoCDClusterRancherUsername {
			return user.GetName(), nil
		}
	}
	return "", r.log.ErrorfNewErr("Failed to find a Rancher user with username %s", vzconst.ArgoCDClusterRancherUsername)
}

func (r *VerrazzanoManagedClusterReconciler) unregisterClusterFromArgoCD(ctx context.Context, vmc *clusterapi.VerrazzanoManagedCluster) error {
	clusterSec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmc.Name + "-" + clusterSecretName,
			Namespace: constants.ArgoCDNamespace,
		},
	}
	if err := r.Delete(context.TODO(), &clusterSec); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return r.log.ErrorfNewErr("Failed to delete Argo CD cluster secret", err)
	}

	return nil
}

// getArgoCACert the initial build-in admin user admi password. If the secret does not exist, we
// return a nil slice.
func getArgoCDAdminSecret(rdr client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: constants.ArgoCDNamespace,
		Name:      argocdAdminSecret}

	if err := rdr.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// newArgoCDConfig returns a populated ArgoCDConfig struct that can be used to make calls to the clusters API
func newArgoCDConfig(rdr client.Reader, log vzlog.VerrazzanoLogger) (*ArgoCDConfig, error) {
	ac := &ArgoCDConfig{BaseURL: "https://" + serviceURL}
	log.Debug("Getting ArgoCD ingress host name")
	hostname, err := getArgoCDIngressHostname(rdr)
	if err != nil {
		log.Errorf("Failed to get ArgoCD ingress host name: %v", err)
		return nil, err
	}
	ac.Host = hostname
	//ac.BaseURL = "https://" + ac.Host

	caCert, err := common.GetRootCA(rdr)
	if err != nil {
		log.Errorf("Failed to get Rancher TLS root CA: %v", err)
		return nil, err
	}
	ac.CertificateAuthorityData = caCert

	log.Debugf("Checking for Rancher additional CA in secret %s", vzconst.AdditionalTLS)
	ac.AdditionalCA = common.GetAdditionalCA(rdr)

	log.Once("Getting admin token from ArgoCD")
	adminToken, err := getAdminTokenFromArgoCD(rdr, ac, log)
	if err != nil {
		log.ErrorfThrottled("Failed to get admin token from ArgoCD: %v", err)
		return nil, err
	}
	ac.APIAccessToken = adminToken

	return ac, nil
}

// getAdminTokenFromArgoCD does a login with ArgoCD and returns the token from the response
func getAdminTokenFromArgoCD(rdr client.Reader, ac *ArgoCDConfig, log vzlog.VerrazzanoLogger) (string, error) {
	secret, err := getArgoCDAdminSecret(rdr)
	if err != nil {
		return "", err
	}

	action := http.MethodPost
	payload := `{"Username": "admin", "Password": "` + secret + `"}`
	reqURL := ac.BaseURL + sessionPath
	headers := map[string]string{"Content-Type": "application/json"}

	response, responseBody, err := sendHTTPRequest(action, reqURL, headers, payload, ac, log)
	if err != nil {
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		return "", err
	}

	return httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "token", "unable to find token in Rancher response")
}

// sendRequest builds an HTTP request, sends it, and returns the response
func sendHTTPRequest(action string, reqURL string, headers map[string]string, payload string,
	ac *ArgoCDConfig, log vzlog.VerrazzanoLogger) (*http.Response, string, error) {

	req, err := http.NewRequest(action, reqURL, strings.NewReader(payload))
	if err != nil {
		return nil, "", err
	}

	req.Header.Add("Accept", "*/*")

	for k := range headers {
		req.Header.Add(k, headers[k])
	}
	req.Header.Add("Host", ac.Host)
	req.Host = ac.Host

	return doHTTPRequest(req, ac, log)
}

// doRequest configures an HTTP transport (including TLS), sends an HTTP request with retries, and returns the response
func doHTTPRequest(req *http.Request, ac *ArgoCDConfig, log vzlog.VerrazzanoLogger) (*http.Response, string, error) {
	log.Debugf("Attempting HTTP request: %v", req)

	proxyURL := getProxyURL()
	var tlsConfig *tls.Config
	if len(ac.CertificateAuthorityData) < 1 && len(ac.AdditionalCA) < 1 {
		tlsConfig = &tls.Config{
			ServerName: ac.Host,
			MinVersion: tls.VersionTLS12,
		}
	} else {
		tlsConfig = &tls.Config{
			RootCAs:    common.CertPool(ac.CertificateAuthorityData, ac.AdditionalCA),
			ServerName: ac.Host,
			MinVersion: tls.VersionTLS12,
		}
	}
	tr := &http.Transport{
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	// if we have a proxy, then set it in the transport
	if proxyURL != "" {
		u := url.URL{}
		proxy, err := u.Parse(proxyURL)
		if err != nil {
			return nil, "", err
		}
		tr.Proxy = http.ProxyURL(proxy)
	}

	client := &http.Client{Transport: tr, Timeout: 30 * time.Second}
	var resp *http.Response
	var err error

	// resp.Body is consumed by the first try, and then no longer available (empty)
	// so we need to read the body and save it so we can use it in each retry
	buffer, _ := io.ReadAll(req.Body)

	common.Retry(DefaultRetry, log, true, func() (bool, error) {
		// update the body with the saved data to prevent the "zero length body" error
		req.Body = io.NopCloser(bytes.NewBuffer(buffer))
		resp, err = ArgoCDHTTPClient.Do(client, req)

		// check for a network error and retry
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			log.Infof("Temporary error executing HTTP request %v : %v, retrying", req, nerr)
			return false, err
		}

		// if err is another kind of network error that is not considered "temporary", then retry
		if err, ok := err.(*url.Error); ok {
			if err, ok := err.Err.(*net.OpError); ok {
				if derr, ok := err.Err.(*net.DNSError); ok {
					log.Infof("DNS error: %v, retrying", derr)
					return false, err
				}
			}
		}

		// retry any HTTP 500 errors
		if resp != nil && resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			log.ErrorfThrottled("HTTP status %v executing HTTP request %v, retrying", resp.StatusCode, req)
			return false, err
		}

		// if err is some other kind of unexpected error, retry
		if err != nil {
			return false, err
		}
		return true, err
	})

	if err != nil {
		return resp, "", err
	}
	defer resp.Body.Close()

	// extract the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return resp, string(body), err
}

// getProxyURL returns an HTTP proxy from the environment if one is set, otherwise an empty string
func getProxyURL() string {
	if proxyURL := os.Getenv("https_proxy"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("HTTPS_PROXY"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("http_proxy"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("HTTP_PROXY"); proxyURL != "" {
		return proxyURL
	}
	return ""
}

// getArgoCDIngressHostname gets the ArgoCD ingress host name. This is used to set the host for TLS.
func getArgoCDIngressHostname(rdr client.Reader) (string, error) {
	ingress := &k8net.Ingress{}
	nsName := types.NamespacedName{
		Namespace: argocd.ComponentNamespace,
		Name:      constants.ArgoCDIngress}
	if err := rdr.Get(context.TODO(), nsName, ingress); err != nil {
		return "", fmt.Errorf("Failed to get ArgoCD ingress %v: %v", nsName, err)
	}

	if len(ingress.Spec.Rules) > 0 {
		// the first host will do
		return ingress.Spec.Rules[0].Host, nil
	}

	return "", fmt.Errorf("Failed, ArgoCD ingress %v is missing host names", nsName)
}

func newArgoCDRegistration(status clusterapi.ArgoCDRegistrationStatus, message string) *clusterapi.ArgoCDRegistration {
	now := metav1.Now()
	return &clusterapi.ArgoCDRegistration{
		Status:    status,
		Timestamp: &now,
		Message:   message,
	}
}
