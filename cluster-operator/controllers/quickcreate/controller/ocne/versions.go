// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocne

import (
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type (
	VersionDefaults struct {
		Release         string `json:"Release"`
		ContainerImages struct {
			Calico         string `json:"calico"`
			CoreDNS        string `json:"coredns"`
			ETCD           string `json:"etcd"`
			TigeraOperator string `json:"tigera-operator" yaml:"tigera-operator"`
		} `json:"container-images" yaml:"container-images"`
		KubernetesVersion string `json:"-"`
	}
)

const (
	ocneConfigMapName      = "ocne-metadata"
	ocneConfigMapNamespace = "verrazzano-capi"
)

func GetVersionDefaults(ctx context.Context, cli clipkg.Client, ocneVersion string) (*VersionDefaults, error) {
	versions, err := getVersionMapping(ctx, cli)
	if err != nil {
		return nil, err
	}
	for k, v := range versions {
		if ocneVersion == v.Release {
			v.KubernetesVersion = k
			return v, nil
		}
	}
	return nil, fmt.Errorf("no verion mapping found for OCNE version %s", ocneVersion)
}

func getVersionMapping(ctx context.Context, cli clipkg.Client) (map[string]*VersionDefaults, error) {
	cm := &corev1.ConfigMap{}
	err := cli.Get(ctx, types.NamespacedName{
		Namespace: ocneConfigMapNamespace,
		Name:      ocneConfigMapName,
	}, cm)
	if err != nil {
		return nil, err
	}
	if cm.Data == nil {
		return nil, nil
	}
	mapping := cm.Data["mapping"]
	if len(mapping) < 1 {
		return nil, errors.New("no OCNE version mapping found")
	}
	versions := map[string]*VersionDefaults{}
	if err := yaml.Unmarshal([]byte(mapping), &versions); err != nil {
		return nil, err
	}
	if len(versions) < 1 {
		return nil, errors.New("no OCNE kubernetes versions found")
	}
	return versions, nil
}
