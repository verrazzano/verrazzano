// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestCreateAdminSecretIfNotExists(t *testing.T) {
	log := getTestLogger(t)
	ff := func(a ...string) (string, string, error) {
		return "", "", nil
	}

	podList := createRancherPodList()
	adminSecret := createAdminSecret()

	var tests = []struct {
		testName string
		c        client.Client
		f        func(a ...string) (string, string, error)
		isErr    bool
	}{
		{
			"should skip secret creation when secret is present",
			fake.NewFakeClientWithScheme(getScheme(), &adminSecret),
			ff,
			false,
		},
		{
			"should be able to reset the admin password",
			fake.NewFakeClientWithScheme(getScheme(), &podList),
			func(a ...string) (string, string, error) {
				return "password", "", nil
			},
			false,
		},
		{
			"should fail when resetting admin password fails",
			fake.NewFakeClientWithScheme(getScheme(), &podList),
			func(a ...string) (string, string, error) {
				return "", "", errors.New("something bad happened!")
			},
			true,
		},
		{
			"should fail when no rancher pod is available",
			fake.NewFakeClientWithScheme(getScheme()),
			ff,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			setBashFunc(tt.f)
			err := createAdminSecretIfNotExists(log, tt.c)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestPatchAgents(t *testing.T) {
	log := getTestLogger(t)
	ip, host := "ip", "host"

	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      clusterAgentDeployName,
		},
	}
	daemonset := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      nodeAgentDaemonsetName,
		},
	}

	var tests = []struct {
		testName string
		c        client.Client
	}{
		{
			"should patch agents when present",
			fake.NewFakeClientWithScheme(getScheme(), &deploy, &daemonset),
		},
		{
			"should not fail when agents are not present",
			fake.NewFakeClientWithScheme(getScheme()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Nil(t, patchAgents(log, tt.c, host, ip))
		})
	}
}

func TestGetRancherIP(t *testing.T) {
	log := getTestLogger(t)

	in := createRancherIngress()
	inNoIp := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}

	var tests = []struct {
		testName string
		c        client.Client
		isErr    bool
	}{
		{
			"should be able to get an ingress ip",
			fake.NewFakeClientWithScheme(getScheme(), &in),
			false,
		},
		{
			"ingress should not be found",
			fake.NewFakeClientWithScheme(getScheme()),
			true,
		},
		{
			"ingress ip should not be found",
			fake.NewFakeClientWithScheme(getScheme(), &inNoIp),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			ip, err := getRancherIngressIP(log, tt.c)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, "ip", ip)
			}
		})
	}
}
