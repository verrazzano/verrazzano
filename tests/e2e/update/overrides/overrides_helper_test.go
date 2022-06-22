// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package overrides

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var oldCMData = `
prometheusOperator:
  podLabels:
    override: "true"
`

var oldSecretData = `
prometheusOperator:
  podAnnotations:
    override: "true"
`

var newCMData = `
prometheusOperator:
  podLabels:
    override: "false"
`

var newSecretData = `
prometheusOperator:
  podAnnotations:
    override: "false"
`

var oldInlineData = "{\"prometheusOperator\": {\"podAnnotations\": {\"inlineOverride\": \"true\"}}}"

var newInlineData = "{\"prometheusOperator\": {\"podAnnotations\": {\"inlineOverride\": \"false\"}}}"

var testConfigMap = corev1.ConfigMap{
	TypeMeta: metav1.TypeMeta{
		Kind: constants.ConfigMapKind,
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      overrideConfigMapSecretName,
		Namespace: constants.DefaultNamespace,
	},
	Immutable:  nil,
	Data:       map[string]string{},
	BinaryData: nil,
}

var testSecret = corev1.Secret{
	TypeMeta: metav1.TypeMeta{
		Kind: constants.SecretKind,
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      overrideConfigMapSecretName,
		Namespace: constants.DefaultNamespace,
	},
	Immutable:  nil,
	Data:       nil,
	StringData: map[string]string{},
	Type:       "",
}
