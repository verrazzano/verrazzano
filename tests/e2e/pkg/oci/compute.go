// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"errors"
	"github.com/oracle/oci-go-sdk/v53/core"
	"strings"
)

// GetOL8ImageID returns the latest OL8 image id for a given compartment
func GetOL8ImageID(client core.ComputeClient, compartmentID string) (string, error) {
	images, err := client.ListImages(context.Background(), core.ListImagesRequest{
		CompartmentId: &compartmentID,
	})
	if err != nil {
		return "", err
	}
	var filteredImages []string
	for _, image := range images.Items {
		name := *image.DisplayName
		if !strings.Contains(name, "GPU") &&
			!strings.Contains(name, "Oracle-Linux-6") &&
			!strings.Contains(name, "aarch64") &&
			strings.HasPrefix(name, "Oracle-Linux-8") {
			filteredImages = append(filteredImages, *image.Id)
		}
	}
	if len(filteredImages) < 1 {
		return "", errors.New("no OL8 compatible images found")
	}
	return filteredImages[0], nil
}
