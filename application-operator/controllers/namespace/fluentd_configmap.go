// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/constants"
	k8serrors "github.com/verrazzano/verrazzano/pkg/k8s/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	fluentdConfigMapName = "fluentd-config"

	nsConfigKeyTemplate         = "oci-logging-ns-%s.conf"
	fluentdConfigKey            = "fluentd-standalone.conf"
	startNamespaceConfigsMarker = "# Start namespace logging configs"
)

const loggingTemplateBody = `|
<match kubernetes.**_{{ .namespace }}_**>
  @type oci_logging
  log_object_id {{ .logId }}
  <buffer>
    @type file
    path /fluentd/log/oci-logging-ns-{{ .namespace }}
    disable_chunk_backup true
    chunk_limit_size 5MB
    flush_interval 180s
    total_limit_size 1GB
    overflow_action throw_exception
    retry_type exponential_backoff
  </buffer>
</match>
`

var loggingTemplate *template.Template

// init creates a logging template.
func init() {
	loggingTemplate, _ = template.New("loggingConfig").Parse(loggingTemplateBody)
}

// addNamespaceLogging updates the system Fluentd config map to include a match section that directs all logs for the given
// namespace to the given OCI Log object id. It returns true if the config map was updated.
func addNamespaceLogging(ctx context.Context, cli client.Client, namespace string, ociLogID string) (bool, error) {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fluentdConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}}

	opResult, err := controllerutil.CreateOrUpdate(ctx, cli, cm, func() error {
		if cm.ObjectMeta.CreationTimestamp.IsZero() {
			return fmt.Errorf("configmap '%s' in namespace '%s' must exist", cm.ObjectMeta.Name, cm.ObjectMeta.Namespace)
		}
		return addNamespaceLoggingToConfigMap(cm, namespace, ociLogID)
	})

	if err != nil {
		return false, err
	}

	return opResult != controllerutil.OperationResultNone, nil
}

// removeNamespaceLogging updates the system Fluentd config map, removing the namespace-specific logging configuration.
// It returns true if the config map was updated.
func removeNamespaceLogging(ctx context.Context, cli client.Client, namespace string) (bool, error) {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fluentdConfigMapName, Namespace: constants.VerrazzanoSystemNamespace}}

	opResult, err := controllerutil.CreateOrUpdate(ctx, cli, cm, func() error {
		// if the config map exists, remove the namespace logging config
		if !cm.ObjectMeta.CreationTimestamp.IsZero() {
			removeNamespaceLoggingFromConfigMap(cm, namespace)
			return nil
		}
		// return an error here, otherwise the configmap will get created and we don't want that
		return k8serrors.NewNotFound(schema.ParseGroupResource("ConfigMap"), fluentdConfigMapName)
	})

	if err != nil {
		return false, client.IgnoreNotFound(err)
	}

	return opResult != controllerutil.OperationResultNone, nil
}

// addNamespaceLoggingToConfigMap adds a config map key for the namespace-specific logging configuration and
// adds an include directive in the main Fluentd config. This function is idempotent.
func addNamespaceLoggingToConfigMap(configMap *corev1.ConfigMap, namespace string, ociLogID string) error {
	// make sure the logging template parsed
	if loggingTemplate == nil {
		return fmt.Errorf("logging config template is empty")
	}

	// use the template to create the logging config for the namespace and add it to the config map
	pairs := map[string]string{"namespace": namespace, "logId": ociLogID}
	var buff bytes.Buffer
	if err := loggingTemplate.Execute(&buff, pairs); err != nil {
		return err
	}

	nsConfigKey := fmt.Sprintf(nsConfigKeyTemplate, namespace)
	configMap.Data[nsConfigKey] = buff.String()

	// if the logging config isn't already included in the main Fluentd config, include it
	if fluentdConfig, ok := configMap.Data[fluentdConfigKey]; ok {
		includeLine := fmt.Sprintf("@include %s", nsConfigKey)
		if !strings.Contains(fluentdConfig, includeLine) {
			replace := fmt.Sprintf("%s\n%s", startNamespaceConfigsMarker, includeLine)
			fluentdConfig = strings.Replace(fluentdConfig, startNamespaceConfigsMarker, replace, 1)
			configMap.Data[fluentdConfigKey] = fluentdConfig
		}
	}

	return nil
}

// removeNamespaceLoggingFromConfigMap removes the config map key for the namespace-specific logging configuration and
// removes the include directive in the main Fluentd config. This function is idempotent.
func removeNamespaceLoggingFromConfigMap(configMap *corev1.ConfigMap, namespace string) {
	// remove the map entry for the logging config
	nsConfigKey := fmt.Sprintf(nsConfigKeyTemplate, namespace)
	delete(configMap.Data, nsConfigKey)

	// if the logging config is included in the main Fluentd config, remove it
	if fluentdConfig, ok := configMap.Data[fluentdConfigKey]; ok {
		includeLine := fmt.Sprintf("@include %s", nsConfigKey)
		if strings.Contains(fluentdConfig, includeLine) {
			toRemove := fmt.Sprintf("%s\n", includeLine)
			fluentdConfig = strings.Replace(fluentdConfig, toRemove, "", 1)
			configMap.Data[fluentdConfigKey] = fluentdConfig
		}
	}
}
