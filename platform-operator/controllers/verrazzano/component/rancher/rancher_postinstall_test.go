// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"k8s.io/client-go/rest"
	testclient "k8s.io/client-go/rest/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var validStdOut = "W1122 18:11:20.905585\nNew password for default admin user (user-p958n):\npassword\n"

func TestCreateAdminSecretIfNotExists(t *testing.T) {
	log := getTestLogger(t)
	clientConfigFunction := func() (*rest.Config, rest.Interface, error) {
		cfg, _ := rest.InClusterConfig()

		return cfg, &testclient.RESTClient{}, nil
	}
	common.NewSPDYExecutor = common.FakeNewSPDYExecutor
	podList := createRancherPodList()
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
			fake.NewFakeClientWithScheme(getScheme(), &adminSecret),
			clientConfigFunction,
			false,
		},
		{
			"should be able to reset the admin password",
			validStdOut,
			fake.NewFakeClientWithScheme(getScheme(), &podList),
			clientConfigFunction,
			false,
		},
		{
			"should fail when resetting admin password fails",
			"",
			fake.NewFakeClientWithScheme(getScheme(), &podList),
			clientConfigFunction,
			true,
		},
		{
			"should fail when no rancher pod is available",
			validStdOut,
			fake.NewFakeClientWithScheme(getScheme()),
			clientConfigFunction,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			common.FakeStdOut = tt.stdout
			setRestClientConfig(tt.f)
			err := createAdminSecretIfNotExists(log, tt.c)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

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
