// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/common/auth"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	CredentialsLoader interface {
		GetCredentialsIfAllowed(ctx context.Context, cli clipkg.Client, identityRef vmcv1alpha1.NamespacedRef, namespace string) (*Credentials, error)
	}
	CredentialsLoaderImpl struct{}
	Credentials           struct {
		Region               string
		Tenancy              string
		User                 string
		PrivateKey           string
		Fingerprint          string
		Passphrase           string
		UseInstancePrincipal string
	}
	CAPIIdentity struct {
		Spec struct {
			Namespaces      *AllowedNamespaces `json:"allowedNamespaces"`
			PrincipalSecret struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"principalSecret"`
		} `json:"spec"`
	}
	AllowedNamespaces struct {
		List     []string              `json:"list"`
		Selector *metav1.LabelSelector `json:"selector"`
	}
)

var (
	_                     CredentialsLoader = CredentialsLoaderImpl{}
	gvkOCIClusterIdentity                   = schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta2",
		Kind:    "ociclusteridentity",
	}
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

// GetCredentialsIfAllowed fetches the OCI Credentials for an OCIClusterIdentity, if that OCIClusterIdentity exists, has a principal secret,
// and allows access from a given namespace.
func (c CredentialsLoaderImpl) GetCredentialsIfAllowed(ctx context.Context, cli clipkg.Client, identityRef vmcv1alpha1.NamespacedRef, namespace string) (*Credentials, error) {
	nsn := types.NamespacedName{
		Name:      identityRef.Name,
		Namespace: identityRef.Namespace,
	}
	identity, err := getIdentity(ctx, cli, nsn)
	if err != nil {
		return nil, err
	}
	if !IsAllowedNamespace(ctx, cli, identity, namespace) {
		return nil, fmt.Errorf("cannot access OCI identity %s/%s", identityRef.Namespace, identityRef.Name)
	}
	return getCredentialsFromIdentity(ctx, cli, identity)
}

func (c Credentials) AsConfigurationProvider() (common.ConfigurationProvider, error) {
	if c.UseInstancePrincipal == "true" {
		return auth.InstancePrincipalConfigurationProvider()
	}
	var passphrase *string
	if len(c.Passphrase) > 0 {
		passphrase = &c.Passphrase
	}
	return common.NewRawConfigurationProvider(c.Tenancy, c.User, c.Region, c.Fingerprint, c.PrivateKey, passphrase), nil
}

// IsAllowedNamespace checks if a given identity allows access from a given namespace.
func IsAllowedNamespace(ctx context.Context, cli clipkg.Client, identity *CAPIIdentity, namespace string) bool {
	// No allowed namespaces means nothing allowed
	if identity.Spec.Namespaces == nil {
		return false
	}
	// Empty allowed namespaces means all namespaces allowed
	if reflect.DeepEqual(*identity.Spec.Namespaces, AllowedNamespaces{}) {
		return true
	}
	// If namespace is in the allowed namespaces list, access is permitted
	for _, ns := range identity.Spec.Namespaces.List {
		if ns == namespace {
			return true
		}
	}
	// Deny access for invalid or empty selectors
	selector, err := metav1.LabelSelectorAsSelector(identity.Spec.Namespaces.Selector)
	if err != nil {
		return false
	}
	if selector.Empty() {
		return false
	}
	// Allow access if the namespace matches any namespace from the selectors
	namespaces := &v1.NamespaceList{}
	if err := cli.List(ctx, namespaces, clipkg.MatchingLabelsSelector{Selector: selector}); err != nil {
		return false
	}
	for _, ns := range namespaces.Items {
		if ns.Name == namespace {
			return true
		}
	}
	return false
}

func getIdentity(ctx context.Context, cli clipkg.Client, nsn types.NamespacedName) (*CAPIIdentity, error) {
	unstructuredIdentity := &unstructured.Unstructured{}
	unstructuredIdentity.SetGroupVersionKind(gvkOCIClusterIdentity)
	if err := cli.Get(ctx, nsn, unstructuredIdentity); err != nil {
		return nil, err
	}
	identityBytes, err := json.Marshal(unstructuredIdentity.Object)
	if err != nil {
		return nil, err
	}
	identity := &CAPIIdentity{}
	err = json.Unmarshal(identityBytes, identity)
	return identity, err
}

func getCredentialsFromIdentity(ctx context.Context, cli clipkg.Client, identity *CAPIIdentity) (*Credentials, error) {
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
