// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	testclient "k8s.io/client-go/rest/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var validStdOut = "W1122 18:11:20.905585\nNew password for default admin user (user-p958n):\npassword\n"

// TestCreateAdminSecretIfNotExists verifies creation of the rancher-admin-secret Secret
// GIVEN a cluster with Rancher running
//
//	WHEN createAdminSecretIfNotExists is called
//	THEN createAdminSecretIfNotExists should ensure the rancher-admin-secret exists and contains a valid Rancher admin password
func TestCreateAdminSecretIfNotExists(t *testing.T) {
	log := getTestLogger(t)
	clientConfigFunction := func() (*rest.Config, rest.Interface, error) {
		cfg, _ := rest.InClusterConfig()

		return cfg, &testclient.RESTClient{}, nil
	}
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config, k := k8sutilfake.NewClientsetConfig()
		return config, k, nil
	}

	podListAllRunning := createRancherPodListWithAllRunning()
	podListNoneRunning := createRancherPodListWithNoneRunning()
	podListLastRunning := createRancherPodListWithLastRunning()
	adminSecret := createAdminSecret()

	var tests = []struct {
		testName string
		stdout   string
		c        client.Client
		f        func() (*rest.Config, rest.Interface, error)
		isErr    bool
	}{
		{
			"should skip secret creation when secret is present",
			"",
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&adminSecret).Build(),
			clientConfigFunction,
			false,
		},
		{
			"should be able to reset the admin password",
			validStdOut,
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&podListAllRunning.Items[0]).Build(),
			clientConfigFunction,
			false,
		},
		{
			"should fail when resetting admin password fails",
			"",
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&podListAllRunning.Items[0]).Build(),
			clientConfigFunction,
			true,
		},
		{
			"should fail when no Rancher pods exist",
			validStdOut,
			fake.NewClientBuilder().WithScheme(getScheme()).Build(),
			clientConfigFunction,
			true,
		},
		// GIVEN a cluster with no Rancher pods running
		// WHEN an attempt is made to create the Rancher admin secret
		// THEN the request should fail before attempting to invoke commands
		{
			"should fail when no Rancher pod is available",
			validStdOut,
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&podListNoneRunning.Items[0]).Build(),
			clientConfigFunction,
			true,
		},
		// GIVEN a cluster with one of several Rancher pods running
		// WHEN an attempt is made to create the Rancher admin secret
		// THEN the request should succeed and correct commands invoked on the pod
		{
			"should pass when one Rancher pod is available",
			validStdOut,
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&podListLastRunning.Items[0], &podListLastRunning.Items[1]).Build(),
			clientConfigFunction,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			k8sutilfake.PodExecResult = func(url *url.URL) (string, string, error) { return tt.stdout, "", nil }
			err := createAdminSecretIfNotExists(log, tt.c)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestParsePasswordStdout verifies parsing the Rancher password from reset STDOUT
// GIVEN STDOUT from the reset-password command
//
//	WHEN parsePasswordStdout is called
//	THEN parsePasswordStdout should return the password from STDOUT
func TestParsePasswordStdout(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{validStdOut, "password"},
		{"foo\npassword\n", "password"},
		{"foo", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.out, parsePasswordStdout(tt.in))
		})
	}
}
