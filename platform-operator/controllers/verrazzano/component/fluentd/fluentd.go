// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// loggingPreInstall copies logging secrets from the verrazzano-install namespace to the verrazzano-system namespace
func loggingPreInstall(ctx spi.ComponentContext) error {
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) {
		// If fluentd is enabled, copy any custom secrets
		fluentdConfig := ctx.EffectiveCR().Spec.Components.Fluentd
		if fluentdConfig != nil {
			// Copy the internal Elasticsearch secret
			if len(fluentdConfig.ElasticsearchURL) > 0 && fluentdConfig.ElasticsearchSecret != globalconst.VerrazzanoESInternal {
				if err := copySecret(ctx, fluentdConfig.ElasticsearchSecret, "custom Elasticsearch"); err != nil {
					return err
				}
			}
			// Copy the OCI API secret
			if fluentdConfig.OCI != nil && len(fluentdConfig.OCI.APISecret) > 0 {
				if err := copySecret(ctx, fluentdConfig.OCI.APISecret, "OCI API"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// copySecret copies a secret from the verrazzano-install namespace to the verrazzano-system namespace. If
// the target secret already exists, then it will be updated if necessary.
func copySecret(ctx spi.ComponentContext, secretName string, logMsg string) error {
	vzLog := ctx.Log()
	vzLog.Debugf("Copying %s secret %s to %s namespace", logMsg, secretName, globalconst.VerrazzanoSystemNamespace)

	targetSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
	}
	opResult, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &targetSecret, func() error {
		sourceSecret := corev1.Secret{}
		nsn := types.NamespacedName{Name: secretName, Namespace: constants.VerrazzanoInstallNamespace}
		if err := ctx.Client().Get(context.TODO(), nsn, &sourceSecret); err != nil {
			return err
		}
		targetSecret.Type = sourceSecret.Type
		targetSecret.Immutable = sourceSecret.Immutable
		targetSecret.StringData = sourceSecret.StringData
		targetSecret.Data = sourceSecret.Data
		return nil
	})

	vzLog.Debugf("Copy %s secret result: %s", logMsg, opResult)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctx.Log().ErrorfNewErr("Failed in create/update for copysecret: %v", err)
		}
		return vzLog.ErrorfNewErr("Failed, the %s secret %s not found in namespace %s", logMsg, secretName, constants.VerrazzanoInstallNamespace)
	}

	return nil
}
