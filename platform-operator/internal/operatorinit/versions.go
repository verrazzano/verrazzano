// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configMapName    = "verrazzano-meta"
	configMapDataKey = "verrazzano-versions"
)

func CreateVerrazzanoVersionsConfigMap(ctx context.Context) error {
	client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return err
	}

	data, err := getVersionConfigMapData()
	if err != nil {
		return err
	}
	cmData := map[string]string{
		configMapDataKey: data,
	}
	cm, err := client.ConfigMaps(vpoconst.VerrazzanoInstallNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		if apierrors.IsNotFound(err) {
			_, err := client.ConfigMaps(vpoconst.VerrazzanoInstallNamespace).Create(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
				},
				Data: cmData,
			}, metav1.CreateOptions{})
			return err
		}
		return err
	}
	cm.Data = cmData
	_, err = client.ConfigMaps(vpoconst.VerrazzanoInstallNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

func getVersionConfigMapData() (string, error) {
	b, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return "", err
	}

	comp, err := b.GetComponent(vpoconst.VerrazzanoPlatformOperatorHelmName)
	if err != nil {
		return "", err
	}
	var vpoTag = ""
	for _, sc := range comp.SubComponents {
		if sc.Name == vpoconst.VerrazzanoPlatformOperatorHelmName {
			vpoTag = sc.Images[0].ImageTag
		}
	}
	if vpoTag == "" {
		return "", errors.New("failed to find image tag for Platform Operator")
	}

	versions := map[string]string{
		b.GetVersion(): vpoTag,
	}

	data, err := json.Marshal(versions)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
