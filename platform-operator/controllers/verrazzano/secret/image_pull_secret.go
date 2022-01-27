// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package secret

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzlog "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultImagePullSecretKeyName = "imagePullSecrets[0].name"

// CheckImagePullSecret Checks if the global image pull secret exists and copies it into the specified namespace; returns
// true if the image pull secret exists and was copied successfully.
func CheckImagePullSecret(client client.Client, targetNamespace string) (bool, error) {
	var targetSecret v1.Secret
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: targetNamespace, Name: constants.GlobalImagePullSecName}, &targetSecret); err == nil {
		return true, nil
	} else if !errors.IsNotFound(err) {
		// Unexpected error
		return false, err
	}
	// Did not find the secret in the target ns, check for global image pull secret in default ns to copy
	var secret v1.Secret
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: constants.GlobalImagePullSecName}, &secret); err != nil {
		if errors.IsNotFound(err) {
			// Global secret not found
			return false, nil
		}
		// we had an unexpected error
		return false, err
	}
	// Copy the global secret to the target namespace
	targetSecret = v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      constants.GlobalImagePullSecName,
			Namespace: targetNamespace,
		},
		Data: secret.Data,
		Type: secret.Type,
	}
	if err := client.Create(context.TODO(), &targetSecret); err != nil && !errors.IsAlreadyExists(err) {
		// An unexpected error occurred copying the secret
		return false, err
	}
	return true, nil
}

// AddGlobalImagePullSecretHelmOverride Adds a helm override Key if the global image pull secret exists and was copied successfully to the target namespace
func AddGlobalImagePullSecretHelmOverride(log vzlog.VerrazzanoLogger, client client.Client, ns string, kvs []bom.KeyValue, keyName string) ([]bom.KeyValue, error) {
	secretExists, err := CheckImagePullSecret(client, ns)
	if err != nil {
		log.Errorf("Error copying global image pull secret %s to %s namespace", constants.GlobalImagePullSecName, ns)
		return kvs, err
	}
	if secretExists {
		kvs = append(kvs, bom.KeyValue{
			Key:   keyName,
			Value: constants.GlobalImagePullSecName,
		})
	}
	return kvs, nil
}
