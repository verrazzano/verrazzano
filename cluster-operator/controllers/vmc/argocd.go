// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	clusterapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterSecretName = "cluster-secret"
	//argocdClusterTokenTTLEnvVarName = "ARGOCD_CLUSTER_TOKEN_TTL" //nolint:gosec
)

func (r *VerrazzanoManagedClusterReconciler) isArgoCDEnabled() bool {
	vz, _ := r.getVerrazzanoResource()
	return vzcr.IsArgoCDEnabled(vz)
}

func (r *VerrazzanoManagedClusterReconciler) isRancherEnabled() bool {
	vz, _ := r.getVerrazzanoResource()
	return vzcr.IsRancherEnabled(vz)
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
		msg := "Could not find Verrazzano resource"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to find Verrazzano resource on admin cluster: %v", err)
	}
	if vz.Status.VerrazzanoInstance == nil {
		msg := "No instance information found in Verrazzano resource status"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to find instance information in Verrazzano resource status")
	}

	var rancherURL = *(vz.Status.VerrazzanoInstance.RancherURL) + k8sClustersPath + clusterID

	if vmc.Status.ArgoCDRegistration.Status == clusterapi.RegistrationPendingRancher || vmc.Status.ArgoCDRegistration.Status == clusterapi.MCRegistrationFailed {
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

		err = r.updateArgoCDClusterRoleBindingTemplate(vmc)
		if err != nil {
			msg := "Failed to update Argo CD ClusterRoleBindingTemplate"
			return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr(msg, err)
		}
	}

	err = r.createClusterSecret(vmc, clusterID, rancherURL)
	if err != nil {
		msg := "Failed to create Argo CD cluster secret"
		return newArgoCDRegistration(clusterapi.MCRegistrationFailed, msg), r.log.ErrorfNewErr("Unable to call Argo CD clusters POST API on admin cluster: %v", err)
	}
	msg := "Successfully registered managed cluster in ArgoCD"
	return newArgoCDRegistration(clusterapi.MCRegistrationCompleted, msg), nil
}

// argocdClusterAdd registers cluster using the Rancher Proxy by creating a user in rancher, with api token and cluster roles set, and a secret containing Rancher proxy for the cluster
func (r *VerrazzanoManagedClusterReconciler) createClusterSecret(vmc *clusterapi.VerrazzanoManagedCluster, clusterID, rancherURL string) error {
	r.log.Debugf("Configuring Rancher user for cluster registration in ArgoCD")

	if vmc.Status.ArgoCDRegistration.Status == clusterapi.RegistrationPendingRancher || vmc.Status.ArgoCDRegistration.Status == clusterapi.MCRegistrationFailed {
		return nil
	}

	caCert, err := common.GetRootCA(r.Client)
	if err != nil {
		return err
	}
	secret, err := GetArgoCDClusterUserSecret(r.Client)
	if err != nil {
		return nil
	}
	rc, err := rancherutil.NewRancherConfigForUser(r.Client, vzconst.ArgoCDClusterRancherUsername, secret, r.log)
	if err != nil {
		return err
	}

	// create/update the cluster secret with the rancher config
	_, err = r.createOrUpdateSecret(rc, vmc, rancherURL, clusterID, caCert)
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

func (r *VerrazzanoManagedClusterReconciler) createOrUpdateSecret(rc *rancherutil.RancherConfig, vmc *clusterapi.VerrazzanoManagedCluster, rancherURL, clusterID string, caData []byte) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Name = vmc.Name + "-" + clusterSecretName
	secret.Namespace = constants.ArgoCDNamespace

	// Create or update on the local cluster
	return controllerruntime.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateClusterSecret(&secret, rc, vmc.Name, clusterID, rancherURL, caData)
		return nil
	})
}

func (r *VerrazzanoManagedClusterReconciler) mutateClusterSecret(secret *corev1.Secret, rc *rancherutil.RancherConfig, clusterID, cluserName string, rancherURL string, caData []byte) error {
	token := rc.APIAccessToken
	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	tokenCreated, okCreated := secret.Annotations["verrazzano.io/createTimestamp"]
	tokenExpiresAt, okExpires := secret.Annotations["verrazzano.io/expiresAtTimestamp"]
	createNewToken := true

	if okCreated && okExpires {
		timeToCheck := time.Now()
		timeCreated, err := time.Parse(time.RFC3339, tokenCreated)
		if err != nil {
			return err
		}
		timeExpired, err := time.Parse(time.RFC3339, tokenExpiresAt)
		if err != nil {
			return err
		}
		createNewToken = (timeToCheck.Unix()-timeCreated.Unix())/(timeExpired.Unix()-timeCreated.Unix())*100 > 75
	}
	if createNewToken {
		// Update the current token ttl using bearer token obtained
		//ttl := os.Getenv(argocdClusterTokenTTLEnvVarName)
		ttl := "30"
		newToken, err := rancherutil.SetTokenTTL(rc, r.log, ttl, clusterID)
		if err != nil {
			return err
		}
		attrs, err := rancherutil.GetToken(rc, r.log, ttl, clusterID)
		if err != nil {
			return err
		}
		secret.Annotations["verrazzano.io/createTimestamp"] = attrs.Created
		secret.Annotations["verrazzano.io/expiresAtTimestamp"] = attrs.ExpiredAt
		token = newToken
	}

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
// to grant the Verrazzano argocd cluster user correct permission on the managed cluster
func (r *VerrazzanoManagedClusterReconciler) updateArgoCDClusterRoleBindingTemplate(vmc *clusterapi.VerrazzanoManagedCluster) error {
	if vmc == nil {
		r.log.Debugf("Empty VMC, no ClusterRoleBindingTemplate created")
		return nil
	}

	clusterID := vmc.Status.RancherRegistration.ClusterID
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

func newArgoCDRegistration(status clusterapi.ArgoCDRegistrationStatus, message string) *clusterapi.ArgoCDRegistration {
	now := metav1.Now()
	return &clusterapi.ArgoCDRegistration{
		Status:    status,
		Timestamp: &now,
		Message:   message,
	}
}
