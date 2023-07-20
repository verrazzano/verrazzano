// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

import (
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ConversionComponents struct {
	AppName             string
	ComponentName       string
	AppNamespace        string
	IngressTrait        *vzapi.IngressTrait
	MetricsTrait        *vzapi.MetricsTrait
	Helidonworkload     *unstructured.Unstructured
	Coherenceworkload   *unstructured.Unstructured
	WeblogicworkloadMap map[string]*unstructured.Unstructured
}

type KubeRecources struct {
	VirtualServices  []*vsapi.VirtualService
	Gateways         []*vsapi.Gateway
	DestinationRules []*vsapi.DestinationRule
	AuthPolicies     []*clisecurity.AuthorizationPolicy
	ServiceMonitors  []*promoperapi.ServiceMonitor
}
