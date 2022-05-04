// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace = "unit-test-ns"
	testLogID     = "ocid1.log.oc1.test"
)

const fluentdConfig = `|
# Common config
@include general.conf

# Input sources
@include systemd-input.conf
@include kubernetes-input.conf

# Parsing/Filtering
@include systemd-filter.conf
@include kubernetes-filter.conf

# Send to storage
@include output.conf
# Start namespace logging configs
# End namespace logging configs
@include oci-logging-system.conf
@include oci-logging-default-app.conf
`

// TestAddAndRemoveNamespaceLogging tests adding and removing namespace logging config to the Fluentd config map.
func TestAddAndRemoveNamespaceLogging(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN a system Fluentd config map
	// WHEN I add namespace-specific logging configuration
	// THEN the config map gets updated in the cluster and contains the new logging config
	cm := newConfigMap()
	cm.Data = make(map[string]string)
	cm.Data[fluentdConfigKey] = fluentdConfig

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(cm).Build()

	updated, err := addNamespaceLogging(context.TODO(), client, testNamespace, testLogID)
	asserts.NoError(err)
	asserts.True(updated)

	// fetch the config map and make sure it was updated correctly
	cm = &corev1.ConfigMap{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: fluentdConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}, cm)
	asserts.NoError(err)

	nsConfigKey := fmt.Sprintf(nsConfigKeyTemplate, testNamespace)
	includeSection := fmt.Sprintf("%s\n@include oci-logging-ns-unit-test-ns.conf\n", startNamespaceConfigsMarker)
	asserts.Contains(cm.Data[fluentdConfigKey], includeSection)
	asserts.Contains(cm.Data, nsConfigKey)
	asserts.Contains(cm.Data[nsConfigKey], testLogID)

	// GIVEN a system Fluentd config map with namespace-specific logging config
	// WHEN I remove the namespace-specific logging configuration
	// THEN the config map gets updated in the cluster and the config map matches the state prior to adding the config
	updated, err = removeNamespaceLogging(context.TODO(), client, testNamespace)
	asserts.NoError(err)
	asserts.True(updated)

	// fetch the config map and make sure the namespace logging config was removed
	cm = &corev1.ConfigMap{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: fluentdConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}, cm)
	asserts.NoError(err)
	asserts.Equal(fluentdConfig, cm.Data[fluentdConfigKey])
	asserts.NotContains(cm.Data, nsConfigKey)
}

// TestMissingFluentdConfigMap tests the cases where the system Fluentd config map is not found.
func TestMissingFluentdConfigMap(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN there is no system Fluentd config map
	// WHEN I add namespace-specific logging configuration
	// THEN an error is returned
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	_, err := addNamespaceLogging(context.TODO(), client, testNamespace, testLogID)
	asserts.Error(err)
	asserts.Contains(err.Error(), "must exist")

	// GIVEN there is no system Fluentd config map
	// WHEN I attempt to remove namespace-specific logging configuration
	// THEN no error is returned and the config map has not been created
	updated, err := removeNamespaceLogging(context.TODO(), client, testNamespace)
	asserts.NoError(err)
	asserts.False(updated)

	cm := &corev1.ConfigMap{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: fluentdConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}, cm)
	asserts.True(errors.IsNotFound(err))
}

// TestAddNamespaceLoggingAlreadyExists tests the case where we add logging config and it already exists in the config map.
// Since this operation is idempotent the config map should not be updated.
func TestAddNamespaceLoggingAlreadyExists(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN a system Fluentd config map
	// WHEN I add namespace-specific logging configuration and I attempt to add the same logging configuration
	// THEN the config map is not updated a second time
	cm := newConfigMap()
	cm.Data = make(map[string]string)
	cm.Data[fluentdConfigKey] = fluentdConfig

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(cm).Build()

	updated, err := addNamespaceLogging(context.TODO(), client, testNamespace, testLogID)
	asserts.NoError(err)
	asserts.True(updated)

	updated, err = addNamespaceLogging(context.TODO(), client, testNamespace, testLogID)
	asserts.NoError(err)
	asserts.False(updated)
}

// TestRemoveNamespaceLoggingDoesNotExist tests the case where we remove logging config and it does not exist in the config map.
// Since this operation is idempotent the config map should not be updated.
func TestRemoveNamespaceLoggingDoesNotExist(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN a system Fluentd config map that does not contain namespace-specific logging config
	// WHEN I attempt to remove the logging config
	// THEN the config map is not updated
	cm := newConfigMap()
	cm.Data = make(map[string]string)
	cm.Data[fluentdConfigKey] = fluentdConfig

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(cm).Build()

	updated, err := removeNamespaceLogging(context.TODO(), client, testNamespace)
	asserts.NoError(err)
	asserts.False(updated)
}

// TestUpdateExistingLoggingConfig tests the case where the logging config already exists but the log id is updated.
func TestUpdateExistingLoggingConfig(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN a system Fluentd config map that contains namespace-specific logging config
	// WHEN I attempt to add logging config for the same namespace and the log id has changed
	// THEN the config map is updated and it contains the updated log id
	cm := newConfigMap()
	cm.Data = make(map[string]string)
	cm.Data[fluentdConfigKey] = fluentdConfig

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(cm).Build()

	updated, err := addNamespaceLogging(context.TODO(), client, testNamespace, testLogID)
	asserts.NoError(err)
	asserts.True(updated)

	// add logging config with an updated log id
	updatedLogID := "ocid1.log.oc1.updated"
	updated, err = addNamespaceLogging(context.TODO(), client, testNamespace, updatedLogID)
	asserts.NoError(err)
	asserts.True(updated)

	// fetch the config map and confirm that the config exists and it references the updated log id
	cm = &corev1.ConfigMap{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: fluentdConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}, cm)
	asserts.NoError(err)

	nsConfigKey := fmt.Sprintf(nsConfigKeyTemplate, testNamespace)
	includeSection := fmt.Sprintf("%s\n@include oci-logging-ns-unit-test-ns.conf\n", startNamespaceConfigsMarker)
	asserts.Contains(cm.Data[fluentdConfigKey], includeSection)
	asserts.Contains(cm.Data, nsConfigKey)
	asserts.Contains(cm.Data[nsConfigKey], updatedLogID)
}

// newConfigMap returns a ConfigMap populated with the system Fluentd config map name and namespace, and
// a creation timestamp.
func newConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fluentdConfigMapName,
			Namespace:         constants.VerrazzanoSystemNamespace,
			CreationTimestamp: metav1.Now(),
		},
	}
}
