// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

// ImageInfo describes a single image used by one of the helm charts.  This structure
// is needed to render the helm chart manifest, so the image fields are correctly populated
// in the resulting YAML.  There is no helm standard which specifies the image information
// used by a template, each product usually has a custom way to do this. The helm keys fields
// in this structure specify those custom keys.
type ImageInfo struct {
	// ImageName specifies the name of the image tag, such as `nginx-ingress-controller`
	ImageName string

	// ImageName specifies the name of the image tag, such as `0.46.0-20210510134749-abc2d2088`
	ImageTag string

	// HelmImageNameKey is the helm template key which identifies the image name.  There are a variety
	// of keys used by the different helm charts, such as `api.imageName`.  The default is `image`
	HelmImageNameKey string

	// HelmTagNameKey is the helm template key which identifies the image tag.  There are a variety
	// of keys used by the different helm charts, such as `api.imageVersion`.  The default is `tag`
	HelmTagNameKey  string

	// HelmRegistryNameKey is the helm template key which identifies the image repository.  This is not
	// normally specified.  An example is `image.registry` in external-dns.  The default is empty string
	HelmRegistryNameKey  string
}

func getImageOverrides(component string, subComponent string) {

}


