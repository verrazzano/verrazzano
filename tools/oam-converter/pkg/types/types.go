// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ConversionComponents struct {
	AppName           string
	ComponentName     string
	AppNamespace      string
	IngressTrait      *vzapi.IngressTrait
	MetricsTrait      *vzapi.MetricsTrait
	Helidonworkload   *unstructured.Unstructured
	Coherenceworkload *unstructured.Unstructured
	Weblogicworkload  *unstructured.Unstructured
}
