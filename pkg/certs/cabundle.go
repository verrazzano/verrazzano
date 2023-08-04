// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certs

import (
	ctx "context"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var secretsList = []struct {
	types.NamespacedName
	caKey string
}{
	{
		NamespacedName: types.NamespacedName{Namespace: globalconst.VerrazzanoSystemNamespace, Name: globalconst.PrivateCABundle},
		caKey:          globalconst.CABundleKey,
	},
	{
		NamespacedName: types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: globalconst.VerrazzanoIngressTLSSecret},
		caKey:          mcconstants.CaCrtKey,
	},
}

// GetLocalClusterCABundleData gets the local cluster CA bundle data from one of the known/expected sources within Verrazzano
//
// Sources, in order of precedence
// - "cacerts.pem" data field in the verrazzano-system/verrazzano-tls-ca secret
// - "ca.crt" data field in the verrazzano-system/verrazzano-tls secret
func GetLocalClusterCABundleData(log *zap.SugaredLogger, cli client.Client, ctx ctx.Context) ([]byte, error) {
	for _, sourceSecretInfo := range secretsList {
		log.Debugf("checking secret %s", sourceSecretInfo.NamespacedName)
		bundleData, found, err := getBundleDataFromSecret(cli, ctx, sourceSecretInfo.NamespacedName, sourceSecretInfo.caKey)
		if err != nil {
			log.Errorf("Failed retrieving bundle data from secret %s", sourceSecretInfo.NamespacedName)
			return nil, err
		}
		if found {
			log.Debugf("Using bundle data from secret %s", sourceSecretInfo.NamespacedName)
			return bundleData, nil
		}
	}
	log.Debugf("No bundle data found")
	return nil, nil
}

// getBundleDataFromSecret Obtains bundle data from secret using provided key; returns the data and "true" if the data was found, or nil/false otherwise
func getBundleDataFromSecret(cli client.Client, ctx ctx.Context, name types.NamespacedName, caKey string) (bundleData []byte, found bool, err error) {
	sourceSecret := corev1.Secret{}
	err = cli.Get(ctx, client.ObjectKey{
		Namespace: name.Namespace,
		Name:      name.Name,
	}, &sourceSecret)
	if client.IgnoreNotFound(err) != nil {
		return nil, false, err
	}
	if err == nil {
		return sourceSecret.Data[caKey], true, nil
	}
	return nil, false, nil
}
