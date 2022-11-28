// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestShouldSyncRancherClusters tests the shouldSyncRancherClusters function
func TestShouldSyncRancherClusters(t *testing.T) {
	asserts := assert.New(t)

	// when this test is done, reset the cluster sync env var
	envValue, envValueExists := os.LookupEnv(syncClustersEnvVarName)
	defer func() {
		if envValueExists {
			os.Setenv(syncClustersEnvVarName, envValue)
		} else {
			os.Unsetenv(syncClustersEnvVarName)
		}
	}()

	var tests = []struct {
		testName              string
		enabled               bool
		clusterSelectorText   string
		expectedLabelSelector *metav1.LabelSelector
		expectedError         bool
	}{
		// GIVEN cluster sync is disabled
		// WHEN  shouldSyncRancherClusters is called
		// THEN  the call returns that cluster sync is disabled, a nil label selector, and no error
		{
			"Sync Rancher clusters is disabled",
			false,
			"",
			nil,
			false,
		},
		// GIVEN cluster sync is enabled and no label selector yaml is provided
		// WHEN  shouldSyncRancherClusters is called
		// THEN  the call returns that cluster sync is enabled, a nil label selector, and no error
		{
			"Sync Rancher clusters is enabled, no label selector specified",
			true,
			"",
			nil,
			false,
		},
		// GIVEN cluster sync is enabled and a label selector yaml is provided
		// WHEN  shouldSyncRancherClusters is called
		// THEN  the call returns that cluster sync is enabled, a populated label selector, and no error
		{
			"Sync Rancher clusters is enabled, simple label selector specified",
			true,
			"matchLabels:\n  foo: bar\n",
			&metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
			false,
		},
		// GIVEN cluster sync is enabled and a more complex label selector yaml is provided
		// WHEN  shouldSyncRancherClusters is called
		// THEN  the call returns that cluster sync is enabled, a populated label selector, and no error
		{
			"Sync Rancher clusters is enabled, complex label selector specified",
			true,
			"matchLabels:\n  foo: bar\nmatchExpressions:\n- key: clustertype\n  operator: In\n  values: [special, reallyspecial]",
			&metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "clustertype",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"special", "reallyspecial"},
					},
				},
			},
			false,
		},
		// GIVEN cluster sync is enabled and malformed label selector yaml is provided
		// WHEN  shouldSyncRancherClusters is called
		// THEN  the call returns that cluster sync is enabled, a nil label selector, and an error
		{
			"Sync Rancher clusters is enabled, invalid label selector",
			true,
			"matchLabels:\n  bogus\n",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			// if cluster selector text is specified, write it to a temp file
			var filename string
			if len(tt.clusterSelectorText) > 0 {
				var err error
				filename, err = writeTempFile(tt.clusterSelectorText)
				asserts.NoError(err)
				defer os.Remove(filename)
			}
			os.Setenv(syncClustersEnvVarName, strconv.FormatBool(tt.enabled))

			enabled, labelSelector, err := shouldSyncRancherClusters(filename)

			if tt.expectedError {
				asserts.Error(err, tt.testName)
			} else {
				asserts.NoError(err, tt.testName)
			}
			asserts.Equal(tt.enabled, enabled, tt.testName)
			asserts.Equal(tt.expectedLabelSelector, labelSelector, tt.testName)
		})
	}
}

// TestIsCattleClustersCRDInstalled tests the isCattleClustersCRDInstalled function
func TestIsCattleClustersCRDInstalled(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN a cluster that does not have the cattle clusters CRD installed
	// WHEN  a call is made to isCattleClustersCRDInstalled
	// THEN  the function returns false
	client := fake.NewSimpleClientset().ApiextensionsV1()
	isInstalled, err := isCattleClustersCRDInstalled(client)
	asserts.NoError(err)
	asserts.False(isInstalled)

	// GIVEN a cluster that does have the cattle clusters CRD installed
	// WHEN  a call is made to isCattleClustersCRDInstalled
	// THEN  the function returns true
	crd := &apiextv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: cattleClustersCRDName,
		},
	}
	client = fake.NewSimpleClientset(crd).ApiextensionsV1()
	isInstalled, err = isCattleClustersCRDInstalled(client)
	asserts.NoError(err)
	asserts.True(isInstalled)
}

// TestWatchCattleClustersCRD tests the watchCattleClustersCRD function
func TestWatchCattleClustersCRD(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN the cattle clusters CRD is installed after the cluster operator starts up
	// WHEN  the watchCattleClustersCRD function is called
	// THEN  the context Done channel has been closed due to the context cancel function being called
	crd := &apiextv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: cattleClustersCRDName,
		},
	}
	client := fake.NewSimpleClientset(crd).ApiextensionsV1()

	ctx, cancel := context.WithCancel(context.TODO())
	watchCattleClustersCRD(cancel, client, false, zap.L().Sugar())
	_, open := <-ctx.Done()
	asserts.False(open)

	// GIVEN the cattle clusters CRD is uninstalled after the cluster operator starts up
	// WHEN  the watchCattleClustersCRD function is called
	// THEN  the context Done channel has been closed due to the context cancel function being called
	client = fake.NewSimpleClientset().ApiextensionsV1()

	ctx, cancel = context.WithCancel(context.TODO())
	watchCattleClustersCRD(cancel, client, true, zap.L().Sugar())
	_, open = <-ctx.Done()
	asserts.False(open)
}

// writeTempFile creates a temp file with the specified string content. It returns the
// file name.
func writeTempFile(clusterSelectorText string) (string, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	f.Write([]byte(clusterSelectorText))
	f.Close()
	return f.Name(), nil
}
