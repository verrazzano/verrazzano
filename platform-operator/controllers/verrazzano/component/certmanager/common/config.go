// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type GetCoreV1ClientFuncType func(log ...vzlog.VerrazzanoLogger) (clientcorev1.CoreV1Interface, error)

var GetClientFunc GetCoreV1ClientFuncType = k8sutil.GetCoreV1Client

func ResetCoreV1ClientFunc() {
	GetClientFunc = k8sutil.GetCoreV1Client
}

func IsOCIDNS(vz *vzapi.Verrazzano) bool {
	return vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil
}

func GetSecret(namespace string, name string) (*corev1.Secret, error) {
	v1Client, err := GetClientFunc()
	if err != nil {
		return nil, err
	}
	return v1Client.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// IsCA - Check if cert-type is CA, if not it is assumed to be Acme
func IsCA(compContext spi.ComponentContext) (bool, error) {
	return vzcr.IsCAConfig(compContext.EffectiveCR())
}
