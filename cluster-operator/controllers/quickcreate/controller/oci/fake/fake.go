// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fake

import (
	"context"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	CredentialsLoaderImpl struct {
		Credentials *oci.Credentials
	}
)

func (c *CredentialsLoaderImpl) GetCredentialsIfAllowed(_ context.Context, _ clipkg.Client, _ vmcv1alpha1.NamespacedRef, _ string) (*oci.Credentials, error) {
	return c.Credentials, nil
}
