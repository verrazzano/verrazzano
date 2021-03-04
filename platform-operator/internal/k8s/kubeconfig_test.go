// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestKubeConfig tests the building of a kubeconfig
// GIVEN a valid KubeconfigBuilder
// WHEN the Build function is called
// THEN the correct kubeconfig should be created
func TestKubeConfig(t *testing.T) {
	const (
		b64Cert     = "cert"
		clusterName = "cluster"
		contextName = "context"
		serverURL   = "url"
		token       = "token"
		userName    = "user"
	)
	asserts := assert.New(t)

	kb := KubeconfigBuilder{
		ClusterName: clusterName,
		Server:      serverURL,
		CertAuth:    b64Cert,
		UserName:    userName,
		UserToken:   string(token),
		ContextName: contextName,
	}
	kc := kb.Build()

	asserts.Equal(serverURL, kc.Clusters[0].Cluster.Server, "Incorrect server")
	asserts.Equal(b64Cert, kc.Clusters[0].Cluster.CertAuth, "Incorrect certAuth")
	asserts.Equal(userName, kc.Users[0].Name, "Incorrect username")
	asserts.Equal(token, kc.Users[0].User.Token, "Incorrect token")
	asserts.Equal(contextName, kc.Contexts[0].Name, "Incorrect context name")
	asserts.Equal(userName, kc.Contexts[0].Context.User, "Incorrect context username")
	asserts.Equal(clusterName, kc.Contexts[0].Context.Cluster, "Incorrect context clustername")
	asserts.Equal(contextName, kc.CurrentContext, "Incorrect context")
}
