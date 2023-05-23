// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	return IsCAConfig(compContext.EffectiveCR())
}

// IsCAConfig - Check if cert-type is CA, if not it is assumed to be Acme
func IsCAConfig(cr runtime.Object) (bool, error) {
	if cr == nil {
		return false, fmt.Errorf("Nil CR passed in")
	}
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		componentSpec := vzv1alpha1.Spec.Components
		if componentSpec.ClusterIssuer == nil {
			return true, nil
		}
		return componentSpec.ClusterIssuer.IsCAIssuer()
	} else if vzv1beta1, ok := cr.(*v1beta1.Verrazzano); ok {
		componentSpec := vzv1beta1.Spec.Components
		if componentSpec.ClusterIssuer == nil {
			return true, nil
		}
		return componentSpec.ClusterIssuer.IsCAIssuer()
	}
	return false, fmt.Errorf("Illegal object: %v", cr)
}

// IsACMEConfig - Check if cert-type is LetsEncrypt
func IsACMEConfig(cr runtime.Object) (bool, error) {
	isCAConfig, err := IsCAConfig(cr)
	return !isCAConfig, err
}
