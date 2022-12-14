// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"strings"
)

const (
	istioSubcomponent      = "istiod"
	verrazzanoSubcomponent = "verrazzano"
	proxyv2ImageName       = "proxyv2"
	fluentdImageName       = "fluentd-kubernetes-daemonset"
	wkoSubcomponent        = "weblogic-operator"
	wkoExporterImageName   = "weblogic-monitoring-exporter"
)

type bomTool struct {
	vzbom bom.Bom
}

func newBomTool() (*bomTool, error) {
	vzbom, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}
	return &bomTool{
		vzbom: vzbom,
	}, nil
}

func (b *bomTool) getImage(subComponent, imageName string) (string, error) {
	images, err := b.vzbom.GetImageNameList(subComponent)
	if err != nil {
		return "", errors.New("Failed to get the images for Istiod")
	}
	for i, image := range images {
		if strings.Contains(image, imageName) {
			return images[i], nil
		}
	}
	return "", fmt.Errorf("failed to find %s/%s image in the BOM", subComponent, imageName)
}

func getImages(kvs ...string) (map[string]string, error) {
	bt, err := newBomTool()
	if err != nil {
		return nil, err
	}
	if len(kvs)%2 != 0 {
		return nil, errors.New("must have even key/value pairs")
	}
	images := map[string]string{}
	for i := 0; i < len(kvs); i += 2 {
		subComponent := kvs[i]
		imageName := kvs[i+1]
		image, err := bt.getImage(subComponent, imageName)
		if err != nil {
			return nil, err
		}
		images[imageName] = image
	}
	return images, nil
}
