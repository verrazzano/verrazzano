// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	clusterapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterSecretName               = "argocd-cluster-secret"    //nolint:gosec
	argocdClusterTokenTTLEnvVarName = "ARGOCD_CLUSTER_TOKEN_TTL" //nolint:gosec
	createTimestamp                 = "verrazzano.io/create-timestamp"
	expiresAtTimestamp              = "verrazzano.io/expires-at-timestamp"
	clusterroletemplatebindingsPath = "/v3/clusterroletemplatebindings"
)

func (r *VerrazzanoManagedClusterReconciler) isArgoCDEnabled() (bool, error) {
	vz, err := r.getVerrazzanoResource()
	if err != nil {
		return false, err
	}
	return vzcr.IsArgoCDEnabled(vz), nil
}

func (r *VerrazzanoManagedClusterReconciler) isRancherEnabled() (bool, error) {
	vz, err := r.getVerrazzanoResource()
	if err != nil {
		return false, err
	}
	return vzcr.IsRancherEnabled(vz), nil
}

// registerManagedClusterWithArgoCD creates an argocd cluster secret to register a managed cluster in Argo CD
func (r *VerrazzanoManagedClusterReconciler) registerManagedClusterWithArgoCD(vmc *clusterapi.VerrazzanoManagedCluster) (*clusterapi.ArgoCDRegistration, error) {
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		msg := "Waiting for Rancher manifest to be applied on the managed cluster"
		return newArgoCDRegistration(clusterapi.RegistrationPendingRancher, msg), nil
	}

	vz, err := r.getVerrazzanoResource()
	if err != nil {
		msg := "Failed to find instance information in Verrazzano resource status"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), err
	}
	if vz.Status.VerrazzanoInstance == nil {
		msg := "Failed to find instance information in Verrazzano resource status"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to find instance information in Verrazzano resource status")
	}
	if vz.Status.VerrazzanoInstance.RancherURL == nil {
		msg := "Failed to find Rancher URL in Verrazzano resource status"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to find Rancher URL in Verrazzano resource status")
	}
	var rancherURL = *(vz.Status.VerrazzanoInstance.RancherURL) + k8sClustersPath + clusterID

	// If the managed cluster is not active, we should not attempt to register in Argo CD
	rc, err := rancherutil.NewAdminRancherConfig(r.Client, r.log)
	if err != nil {
		msg := "Could not create rancher config that authenticates with the admin user"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), err
	}
	isActive, err := isManagedClusterActiveInRancher(rc, clusterID, r.log)
	if err != nil || !isActive {
		msg := fmt.Sprintf("Waiting for managed cluster with id %s to become active before registering in Argo CD", clusterID)
		return newArgoCDRegistration(clusterapi.RegistrationPendingRancher, msg), err
	}

	err = r.updateArgoCDClusterRoleBindingTemplate(rc, vmc)
	if err != nil {
		msg := "Failed to update Argo CD ClusterRoleBindingTemplate"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), err
	}

	err = r.createArgoCDClusterSecret(vmc, clusterID, rancherURL)
	if err != nil {
		msg := "Failed to create Argo CD cluster secret"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), err
	}
	msg := "Successfully registered managed cluster in ArgoCD"
	return newArgoCDRegistration(clusterapi.MCRegistrationCompleted, msg), nil
}

// createArgoCDClusterSecret registers cluster with ArgoCD using the "vz-argoCD-reg" user and the Rancher proxy URL for the cluster
func (r *VerrazzanoManagedClusterReconciler) createArgoCDClusterSecret(vmc *clusterapi.VerrazzanoManagedCluster, clusterID, rancherURL string) error {
	r.log.Debugf("Configuring Rancher user for cluster registration in ArgoCD")

	caCert, err := common.GetRootCA(r.Client)
	if err != nil {
		return r.log.ErrorfNewErr("Fail to get the root CA certificate from the Rancher TLS secret: %v", err)
	}
	secret, err := r.getArgoCDClusterUserSecret()
	if err != nil {
		return err
	}
	rc, err := rancherutil.NewRancherConfigForUser(r.Client, vzconst.ArgoCDClusterRancherUsername, secret, r.log)
	if err != nil {
		return err
	}

	// create/update the cluster secret with the rancher config
	err = r.createOrUpdateArgoCDSecret(rc, vmc, rancherURL, clusterID, caCert)
	if err != nil {
		return err
	}

	r.log.Oncef("Successfully registered managed cluster in ArgoCD with name: %s", vmc.Name)
	return nil
}

// GetArgoCDClusterUserSecret fetches the Argo CD Verrazzano user secret
func (r *VerrazzanoManagedClusterReconciler) getArgoCDClusterUserSecret() (string, error) {
	var err error
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      vzconst.ArgoCDClusterRancherSecretName,
	}
	if err = r.Get(context.TODO(), nsName, secret); err != nil {
		return "", r.log.ErrorfNewErr("Failed to get the Argo CD secret: %v", err)
	}
	if pw, ok := secret.Data["password"]; ok {
		return string(pw), nil
	}
	return "", r.log.ErrorfNewErr("Failed to get password from Argo CD secret")
}

type TLSClientConfig struct {
	CaData   string `json:"caData"`
	Insecure bool   `json:"insecure"`
}

type ArgoCDRancherConfig struct {
	BearerToken     string `json:"bearerToken"`
	TLSClientConfig `json:"tlsClientConfig"`
}

// createOrUpdateArgoCDSecret create or update the Argo CD cluster secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateArgoCDSecret(rc *rancherutil.RancherConfig, vmc *clusterapi.VerrazzanoManagedCluster, rancherURL, clusterID string, caData []byte) error {
	var secret corev1.Secret
	secret.Name = vmc.Name + "-" + clusterSecretName
	secret.Namespace = constants.ArgoCDNamespace

	// Create or update on the local cluster
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		return r.mutateArgoCDClusterSecret(&secret, rc, vmc.Name, clusterID, rancherURL, caData)
	})
	return err
}

func (r *VerrazzanoManagedClusterReconciler) mutateArgoCDClusterSecret(secret *corev1.Secret, rc *rancherutil.RancherConfig, clusterID, cluserName string, rancherURL string, caData []byte) error {
	token := rc.APIAccessToken
	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	tokenCreated, okCreated := secret.Annotations[createTimestamp]
	tokenExpiresAt, okExpires := secret.Annotations[expiresAtTimestamp]
	createNewToken := true

	if okCreated && okExpires {
		now := time.Now()
		timeCreated, err := time.Parse(time.RFC3339, tokenCreated)
		if err != nil {
			return r.log.ErrorfNewErr("Failed to parse created timestamp: %v", err)
		}
		timeExpires, err := time.Parse(time.RFC3339, tokenExpiresAt)
		if err != nil {
			return r.log.ErrorfNewErr("Failed to parse expired timestamp: %v", err)
		}
		// Obtain new token if the time elapsed between time created and expired is greater than 3/4 of the lifespan of the token
		lifespan := timeExpires.Sub(timeCreated)
		createNewToken = now.After(timeCreated.Add(lifespan * 3 / 4))
	}
	if createNewToken {
		// Obtain a new token with ttl set using bearer token obtained
		ttl := os.Getenv(argocdClusterTokenTTLEnvVarName)
		newToken, tokenName, err := rancherutil.CreateTokenWithTTL(rc, r.log, ttl, clusterID)
		if err != nil {
			return err
		}
		attrs, err := rancherutil.GetTokenByName(rc, r.log, tokenName)
		if err != nil {
			return err
		}
		secret.Annotations[createTimestamp] = attrs.Created
		secret.Annotations[expiresAtTimestamp] = attrs.ExpiresAt
		token = newToken
	}

	if secret.StringData == nil {
		secret.StringData = make(map[string]string)
	}
	secret.Type = corev1.SecretTypeOpaque
	secret.ObjectMeta.Labels = map[string]string{"argocd.argoproj.io/secret-type": "cluster"}

	secret.StringData["name"] = cluserName
	secret.StringData["server"] = rancherURL

	rancherConfig := &ArgoCDRancherConfig{
		BearerToken: token,
		TLSClientConfig: TLSClientConfig{
			CaData:   base64.StdEncoding.EncodeToString(caData),
			Insecure: false,
		},
	}
	data, err := json.Marshal(rancherConfig)
	if err != nil {
		r.log.ErrorfNewErr("Failed to encode Argo CD rancher config object: %v", err)
		return err
	}
	secret.StringData["config"] = string(data)

	return nil
}

// updateArgoCDClusterRoleBindingTemplate invokes Rancher API creates a new ClusterRoleBindingTemplate for the given VMC
// to grant the Verrazzano argocd cluster user correct permission on the managed cluster
func (r *VerrazzanoManagedClusterReconciler) updateArgoCDClusterRoleBindingTemplate(rc *rancherutil.RancherConfig, vmc *clusterapi.VerrazzanoManagedCluster) error {
	if vmc == nil {
		r.log.Debugf("Empty VMC, no ClusterRoleBindingTemplate created")
		return nil
	}

	clusterID := vmc.Status.RancherRegistration.ClusterID
	userID, err := r.getArgoCDClusterUserID()
	if err != nil {
		return err
	}

	action := http.MethodPost
	payloadData := map[string]string{
		"userId":         userID,
		"roleTemplateId": "cluster-owner",
		"clusterId":      clusterID,
	}
	payload, err := json.Marshal(payloadData)
	if err != nil {
		return r.log.ErrorfNewErr("Failed to encode payload object: %v", err)
	}
	reqURL := rc.BaseURL + clusterroletemplatebindingsPath
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken, "Content-Type": "application/json"}

	response, _, err := rancherutil.SendRequest(action, reqURL, headers, string(payload), rc, r.log)
	if err != nil {
		return err
	}
	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		return r.log.ErrorfThrottledNewErr("Failed configuring Argo CD user cluster role template bindings: %v", err)
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
	err := r.List(context.TODO(), &usersList, &client.ListOptions{})
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
	if err := r.Delete(context.TODO(), &clusterSec); client.IgnoreNotFound(err) != nil {
		return r.log.ErrorfNewErr("Failed to delete Argo CD cluster secret: %v", err)
	}

	return nil
}

func newArgoCDRegistration(status clusterapi.ArgoCDRegistrationStatus, message string) *clusterapi.ArgoCDRegistration {
	now := metav1.Now()
	return &clusterapi.ArgoCDRegistration{
		Status:    status,
		Timestamp: &now,
		Message:   message,
	}
}
