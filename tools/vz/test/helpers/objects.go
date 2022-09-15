// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pinnedVersion = "v1.3.4"
)

func CreateVerrazzanoObject() client.Object {
	return &v1beta1.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
}

func CreateVerrazzanoObjectWithVersion() client.Object {
	vz := CreateVerrazzanoObject().(*v1beta1.Verrazzano)
	vz.Status = v1beta1.VerrazzanoStatus{
		Version: pinnedVersion,
	}
	return vz
}

func CreateTestVPOObjects() []client.Object {
	return []client.Object{
		CreateVPOPod(constants.VerrazzanoPlatformOperator),
		CreateVPOReplicset(constants.VerrazzanoPlatformOperator),
		&appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vzconstants.VerrazzanoInstallNamespace,
				Name:      constants.VerrazzanoPlatformOperator,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": constants.VerrazzanoPlatformOperator},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				UpdatedReplicas:   1,
			},
		},
	}
}

func CreateVPOPod(name string) client.Object {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      name,
			Labels: map[string]string{
				"app":               constants.VerrazzanoPlatformOperator,
				"pod-template-hash": "56f78ffcfd",
			},
		},
	}
}

func CreateVPOReplicset(name string) client.Object {
	return &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      fmt.Sprintf("%s-56f78ffcfd", name),
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "1",
			},
		},
	}
}
