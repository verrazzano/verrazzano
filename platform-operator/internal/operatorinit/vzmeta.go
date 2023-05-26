// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	_ "embed"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	configmapName         = "verrazzano-meta"
	verrazzanoVersionsKey = "verrazzano-versions"
)

//go:embed meta/verrazzano-versions.json
var verrazzanoVersions string

func CreateVZMeta(ctx context.Context) error {
	client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return err
	}

	cm, err := client.ConfigMaps(vpoconst.VerrazzanoInstallNamespace).Get(ctx, configmapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return createMetaConfigMap(ctx, client)
	}
	if err != nil {
		return err
	}
	return updateMetaConfigMap(ctx, client, cm)
}

func createMetaConfigMap(ctx context.Context, client corev1.CoreV1Interface) error {
	_, err := client.ConfigMaps(vpoconst.VerrazzanoInstallNamespace).Create(ctx, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: vpoconst.VerrazzanoInstallNamespace,
		},
		Data: map[string]string{
			verrazzanoVersionsKey: verrazzanoVersions,
		},
	}, metav1.CreateOptions{})
	return err
}

func updateMetaConfigMap(ctx context.Context, client corev1.CoreV1Interface, cm *v1.ConfigMap) error {
	cm.Data = map[string]string{
		verrazzanoVersionsKey: verrazzanoVersions,
	}
	_, err := client.ConfigMaps(vpoconst.VerrazzanoInstallNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}
