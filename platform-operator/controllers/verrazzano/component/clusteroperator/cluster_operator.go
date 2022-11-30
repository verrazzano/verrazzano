// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteroperator

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	rancherutil "github.com/verrazzano/verrazzano/pkg/rancher"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	usersByNamePath = "/v3/users?name="
	usersPath       = "/v3/users"

	dataField     = "data"
	passwordField = "password"
)

var userPayload = `
{
	"description": "Verrazzano Cluster",
	"enabled": true,
	"mustChangePassword": false,
	"name": %s,
	"password": %s,
	"username": %s
}`

// AppendOverrides appends any additional overrides needed by the Cluster Operator component
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	envImageOverride := os.Getenv(constants.VerrazzanoClusterOperatorImageEnvVar)
	if len(envImageOverride) > 0 {
		kvs = append(kvs, bom.KeyValue{
			Key:   "image",
			Value: envImageOverride,
		})
	}

	return kvs, nil
}

// isClusterOperatorReady checks if the cluster operator deployment is ready
func (c clusterOperatorComponent) isClusterOperatorReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	if c.AvailabilityObjects != nil {
		return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), c.AvailabilityObjects.DeploymentNames, 1, prefix)
	}
	return true
}

// GetOverrides gets the install overrides for the Cluster Operator component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.ClusterOperator != nil {
			return effectiveCR.Spec.Components.ClusterOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ClusterOperator != nil {
			return effectiveCR.Spec.Components.ClusterOperator.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}
	return []vzapi.Overrides{}
}

func (c clusterOperatorComponent) postInstallUpgrade(ctx spi.ComponentContext) error {
	if vzcr.IsRancherEnabled(ctx.EffectiveCR()) {
		if err := createVZClusterUser(ctx); err != nil {
			return err
		}

		return rancher.CreateOrUpdateRoleTemplate(ctx, vzconst.VerrazzanoClusterRancherName)
	}
	return nil
}

// createVZClusterUser creates the Verrazzano cluster user in Rancher using the Rancher API
func createVZClusterUser(ctx spi.ComponentContext) error {
	rc, err := rancherutil.NewRancherConfig(ctx.Client(), true, ctx.Log())
	if err != nil {
		return err
	}

	// Send a request to see if the user exists
	reqURL := rc.BaseURL + usersByNamePath + vzconst.VerrazzanoClusterRancherUsername
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken}
	response, body, err := rancherutil.SendRequest(http.MethodGet, reqURL, headers, "", rc, ctx.Log())
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
		return ctx.Log().ErrorfNewErr("Failed getting user %s in Rancher, got status code: %d",
			vzconst.VerrazzanoClusterRancherUsername, response.StatusCode)
	}

	if response.StatusCode == http.StatusOK {
		data, err := httputil.ExtractFieldFromResponseBodyOrReturnError(body, dataField, "failed to locate the data field of the response body")
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed to find user given the username: %v", err)
		}
		if data != "[]" {
			ctx.Log().Oncef("User %s was located, skipping the creation process", vzconst.VerrazzanoClusterRancherUsername)
			return nil
		}
	}

	// If the user has not been located in the response, or the status was not found, generate the user with a new password
	pass, err := vzpassword.GeneratePassword(15)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to generate a password for the Verrazzano cluster user: %v", err)
	}
	reqURL = rc.BaseURL + usersPath
	payload := fmt.Sprintf(userPayload, vzconst.VerrazzanoClusterRancherName, pass, vzconst.VerrazzanoClusterRancherName)
	response, _, err = rancherutil.SendRequest(http.MethodPost, reqURL, headers, payload, rc, ctx.Log())
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create the Verrazzano cluster user in Rancher: %v", err)
	}
	if response.StatusCode != http.StatusCreated {
		return ctx.Log().ErrorfNewErr("Failed creating user %s in Rancher, got status code: %d",
			vzconst.VerrazzanoClusterRancherUsername, response.StatusCode)
	}

	// Store the password in a secret, so we can later use it to provide the Verrazzano cluster user credentials
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoMultiClusterNamespace,
			Name:      vzconst.VerrazzanoClusterRancherName,
		},
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
		secret.Data = map[string][]byte{passwordField: []byte(pass)}
		return nil
	})
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed create or update the Secret for the Verrazzano Cluster User: %v", err)
	}
	return nil
}
