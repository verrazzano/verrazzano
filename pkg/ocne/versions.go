// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocne

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/yaml"
)

type OCNEMetadata struct {
	OCNEImages `json:"container-images"`
}

type OCNEImages struct {
	ETCD           string `json:"etcd"`
	CoreDNS        string `json:"coredns"`
	TigeraOperator string `json:"tigera-operator"`
	Calico         string `json:"calico"`
}

const (
	defaultTigeraOperatorTag = "v1.29.0"
	defaultCalicoTag         = "v3.25.0"
	minOCNEVersion           = "v1.24.8"
	configMapName            = "ocne-metadata"
	cmDataKey                = "mapping"
)

func CreateOCNEMetadataConfigMap(ctx context.Context, metadataFile string) error {
	// Only create OCNE Metadata if file is present
	if _, err := os.Stat(metadataFile); err != nil {
		return nil
	}

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return err
	}

	rawMapping := map[string]OCNEMetadata{}
	if err := yaml.Unmarshal(data, &rawMapping); err != nil {
		return err
	}

	mapping, err := buildMapping(rawMapping)
	if err != nil {
		return nil
	}

	mappingBytes, err := yaml.Marshal(mapping)
	if err != nil {
		return err
	}

	cmData := map[string]string{
		cmDataKey: string(mappingBytes),
	}
	client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return err
	}
	cm, getErr := client.ConfigMaps(vzconst.VerrazzanoInstallNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if getErr != nil {
		// create new configmap if not found
		if apierrors.IsNotFound(getErr) {
			_, createErr := client.ConfigMaps(vzconst.VerrazzanoInstallNamespace).Create(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: vzconst.VerrazzanoInstallNamespace,
				},
				Data: cmData,
			}, metav1.CreateOptions{})
			return createErr
		}
		return getErr
	}
	// update configmap if it already exists
	cm.Data = cmData
	if _, updateErr := client.ConfigMaps(vzconst.VerrazzanoInstallNamespace).Update(ctx, cm, metav1.UpdateOptions{}); updateErr != nil {
		return updateErr
	}
	return nil
}

func DeleteOCNEMetadataConfigMap(ctx context.Context) error {
	client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return err
	}

	err = client.ConfigMaps(vzconst.VerrazzanoInstallNamespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func buildMapping(rawMapping map[string]OCNEMetadata) (map[string]OCNEImages, error) {
	minSupportedVersion, err := semver.NewSemVersion(minOCNEVersion)
	if err != nil {
		return nil, err
	}
	result := map[string]OCNEImages{}
	for version, meta := range rawMapping {
		if ok, _ := isSupported(version, minSupportedVersion); ok {
			if version[0] != 'v' {
				version = fmt.Sprintf("v%s", version)
			}

			// Add OCNE Defaults if metadata not present in yum
			images := meta.OCNEImages
			if images.TigeraOperator == "" {
				images.TigeraOperator = defaultTigeraOperatorTag
			}
			if images.Calico == "" {
				images.Calico = defaultCalicoTag
			}
			result[version] = images
		}
	}
	return result, nil
}

func isSupported(version string, minVersion *semver.SemVersion) (bool, error) {
	v, err := semver.NewSemVersion(version)
	if err != nil {
		return false, err
	}

	return v.IsGreaterThanOrEqualTo(minVersion), nil
}
