// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ConversionComponents struct {
	AppName           string
	ComponentName     string
	AppNamespace      string
	IstioEnabled      bool
	IngressTrait      *vzapi.IngressTrait
	MetricsTrait      *vzapi.MetricsTrait
	Helidonworkload   *unstructured.Unstructured
	Coherenceworkload *unstructured.Unstructured
	Weblogicworkload  *unstructured.Unstructured
	Genericworkload   *unstructured.Unstructured
	Service           *corev1.Service
}
type ConversionInput struct {
	InputDirectory  string
	OutputDirectory string
	Namespace       string
	IstioEnabled    bool
}