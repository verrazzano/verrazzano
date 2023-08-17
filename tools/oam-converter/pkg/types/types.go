// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

import (
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ConversionComponents struct {
	AppName           string
	ComponentName     string
	AppNamespace      string
	IngressTrait      *vzapi.IngressTrait
	Helidonworkload   *unstructured.Unstructured
	Coherenceworkload *unstructured.Unstructured
	Weblogicworkload  *unstructured.Unstructured

	Service *corev1.Service
}

type KubeResources struct {
	VirtualServices  []*vsapi.VirtualService
	Gateway          map[string]interface{}
	DestinationRules []*istioclient.DestinationRule
	AuthPolicies     []*clisecurity.AuthorizationPolicy
	ServiceMonitors  []*promoperapi.ServiceMonitor
}
