// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// Envoy filter yaml
const filterYaml = `
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: server-header-filter
  namespace: istio-system
spec:
  configPatches:
    - applyTo: NETWORK_FILTER
      match:
        listener:
          filterChain:
            filter:
              name: envoy.filters.network.http_connection_manager
      patch:
        operation: MERGE
        value:
          typed_config:
            '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
            server_header_transformation: PASS_THROUGH
`

// specField is the name of the spec field with unstructured
const specField = "spec"

// Create the Envoy network filter
func createEnvoyFilter(log vzlog.VerrazzanoLogger, client clipkg.Client) error {
	// Unmarshal the YAML into an object
	u := &unstructured.Unstructured{Object: map[string]interface{}{}}
	err := yaml.Unmarshal([]byte(filterYaml), u)
	if err != nil {
		return log.ErrorfNewErr("Failed to unmarshal the Envoy filter yaml: %v", err)
	}
	// Make a copy of the spec field
	filterSpec, _, err := unstructured.NestedFieldCopy(u.Object, specField)
	if err != nil {
		return log.ErrorfNewErr("Failed to make a copy of the Envoy filter spec: %v", err)
	}

	// Create or update the filter.  Always replace the entire spec.
	var filter unstructured.Unstructured
	filter.SetAPIVersion("networking.istio.io/v1alpha3")
	filter.SetKind("EnvoyFilter")
	filter.SetName("server-header-filter")
	filter.SetNamespace(constants.IstioSystemNamespace)
	_, err = controllerutil.CreateOrUpdate(context.TODO(), client, &filter, func() error {
		if err := unstructured.SetNestedField(filter.Object, filterSpec, specField); err != nil {
			log.ErrorfNewErr("Unable to set the Envoy filter spec: %v", err)
			return err
		}
		return nil
	})
	return err
}
