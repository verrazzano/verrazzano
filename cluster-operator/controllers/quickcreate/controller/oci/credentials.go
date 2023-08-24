// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"fmt"
	capociv1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	"github.com/oracle/cluster-api-provider-oci/cloud/util"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ociTenancyField              = "tenancy"
	ociUserField                 = "user"
	ociFingerprintField          = "fingerprint"
	ociRegionField               = "region"
	ociPassphraseField           = "passphrase"
	ociKeyField                  = "key"
	ociUseInstancePrincipalField = "useInstancePrincipal"
)

type (
	Credentials struct {
		Region               string
		Tenancy              string
		User                 string
		PrivateKey           string
		Fingerprint          string
		Passphrase           string
		UseInstancePrincipal string
	}
)

func LoadCredentials(ctx context.Context, cli clipkg.Client, identityRef vmcv1alpha1.NamespacedRef, clusterNamespace string) (*Credentials, error) {
	nsn := types.NamespacedName{
		Name:      identityRef.Name,
		Namespace: identityRef.Namespace,
	}
	identity, ok := IsAllowedNamespace(ctx, cli, nsn, clusterNamespace)
	if !ok {
		return nil, fmt.Errorf("cannot access OCI identity %s/%s", identityRef.Namespace, identityRef.Name)
	}
	return getCredentialsFromIdentity(ctx, cli, identity)
}

func IsAllowedNamespace(ctx context.Context, cli clipkg.Client, nsn types.NamespacedName, namespace string) (*capociv1beta2.OCIClusterIdentity, bool) {
	identity := &capociv1beta2.OCIClusterIdentity{}
	if err := cli.Get(ctx, nsn, identity); err != nil {
		return nil, false
	}
	return identity, util.IsClusterNamespaceAllowed(ctx, cli, identity.Spec.AllowedNamespaces, namespace)
}

func getCredentialsFromIdentity(ctx context.Context, cli clipkg.Client, identity *capociv1beta2.OCIClusterIdentity) (*Credentials, error) {
	nsn := types.NamespacedName{
		Namespace: identity.Spec.PrincipalSecret.Namespace,
		Name:      identity.Spec.PrincipalSecret.Name,
	}
	s := &v1.Secret{}
	if err := cli.Get(ctx, nsn, s); err != nil {
		return nil, err
	}
	return &Credentials{
		Region:               string(s.Data[ociRegionField]),
		Tenancy:              string(s.Data[ociTenancyField]),
		User:                 string(s.Data[ociUserField]),
		PrivateKey:           string(s.Data[ociKeyField]),
		Fingerprint:          string(s.Data[ociFingerprintField]),
		Passphrase:           string(s.Data[ociPassphraseField]),
		UseInstancePrincipal: string(s.Data[ociUseInstancePrincipalField]),
	}, nil
}
