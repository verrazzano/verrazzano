// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/containerengine"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"strings"
)

// GetLatestOKEVersion returns the latest Kubernetes Version from the OKE service
func GetLatestOKEVersion(client containerengine.ContainerEngineClient, compartmentID string) (string, error) {
	options, err := client.GetClusterOptions(context.Background(), containerengine.GetClusterOptionsRequest{
		ClusterOptionId: common.String("all"),
		CompartmentId:   &compartmentID,
	})
	if err != nil {
		return "", err
	}
	var version *semver.SemVersion
	var idx int
	for i, opt := range options.KubernetesVersions {
		v2, err := semver.NewSemVersion(opt)
		if err != nil {
			return "", err
		}
		if version == nil || v2.IsGreaterThanOrEqualTo(version) {
			version = v2
			idx = i
		}
	}
	return options.KubernetesVersions[idx], nil
}

// GetOKENodeImageForVersion returns the latest OKE node image for a given OKE version and compartment
func GetOKENodeImageForVersion(client containerengine.ContainerEngineClient, compartmentID string, version string) (string, error) {
	options, err := client.GetNodePoolOptions(context.Background(), containerengine.GetNodePoolOptionsRequest{
		NodePoolOptionId: common.String("all"),
		CompartmentId:    &compartmentID,
	})
	if err != nil {
		return "", err
	}
	if version[0] == 'v' {
		version = version[1:]
	}
	var filteredImages []string
	for _, item := range options.Sources {
		name := *item.GetSourceName()
		if strings.Contains(name, version) &&
			!strings.Contains(name, "GPU") &&
			!strings.Contains(name, "aarch64") {
			id := *item.(containerengine.NodeSourceViaImageOption).ImageId
			filteredImages = append(filteredImages, id)
		}
	}
	if len(filteredImages) < 1 {
		return "", fmt.Errorf("no node images found for OKE version %s", version)
	}
	return filteredImages[0], nil
}
