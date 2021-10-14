// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// ComponentName is the name of the component
const (
	ComponentName = "verrazzano-application-operator"
)

// AppendApplicationOperatorOverrides Honor the APP_OPERATOR_IMAGE env var if set; this allows an explicit override
// of the verrazzano-application-operator image when set.
func AppendApplicationOperatorOverrides(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	envImageOverride := os.Getenv(constants.VerrazzanoAppOperatorImageEnvVar)
	if len(envImageOverride) == 0 {
		return kvs, nil
	}
	kvs = append(kvs, bom.KeyValue{
		Key:   "image",
		Value: envImageOverride,
	})

	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	// Get fluentd and istio proxy images
	var fluentdImage string
	var istioProxyImage string
	images, err := bomFile.BuildImageOverrides("verrazzano")
	if err != nil {
		return nil, err
	}
	for _, image := range images {
		if image.Key == "logging.fluentdImage" {
			fluentdImage = image.Value
		}
		if image.Key == "monitoringOperator.istioProxyImage" {
			istioProxyImage = image.Value
		}
	}
	if len(fluentdImage) == 0 {
		return nil, fmt.Errorf("Can not find logging.fluentdImage in BOM")
	}
	if len(istioProxyImage) == 0 {
		return nil, fmt.Errorf("Can not find monitoringOperator.istioProxyImage in BOM")
	}

	// fluentdImage for ENV DEFAULT_FLUENTD_IMAGE
	kvs = append(kvs, bom.KeyValue{
		Key:   "fluentdImage",
		Value: fluentdImage,
	})

	// istioProxyImage for ENV ISTIO_PROXY_IMAGE
	kvs = append(kvs, bom.KeyValue{
		Key:   "istioProxyImage",
		Value: istioProxyImage,
	})

	return kvs, nil
}

// IsApplicationOperatorReady checks if the application operator deployment is ready
func IsApplicationOperatorReady(ctx spi.ComponentContext, name string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: "verrazzano-application-operator", Namespace: namespace},
	}
	return status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1)
}

func ApplyCRDYaml(log *zap.SugaredLogger, c client.Client, _ string, _ string, _ string) error {
	var err error
	path := filepath.Join(config.GetHelmAppOpChartsDir(), "/crds")

	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Error(err, "Unable to list files in directory")
		return err
	}
	for _, file := range files {
		u := &unstructured.Unstructured{Object: map[string]interface{}{}}
		yamlBytes, err := ioutil.ReadFile(path + "/" + file.Name())
		if err != nil {
			log.Error(err, "Unable to read file")
			return err
		}
		err = yaml.Unmarshal(yamlBytes, u)
		if err != nil {
			log.Error(err, "Unable to unmarshal yaml")
			return err
		}
		if u.GetKind() == "CustomResourceDefinition" {
			specCopy, _, err := unstructured.NestedFieldCopy(u.Object, "spec")
			if err != nil {
				log.Error(err, "Unable to make a copy of the spec")
				return err
			}

			_, err = controllerutil.CreateOrUpdate(context.TODO(), c, u, func() error {
				return unstructured.SetNestedField(u.Object, specCopy, "spec")
			})
			if err != nil {
				log.Error(err, "Unable persist object to kubernetes")
				return err
			}
		}
	}
	for _, file := range files {
		u := &unstructured.Unstructured{Object: map[string]interface{}{}}
		yamlBytes, err := ioutil.ReadFile(path + "/" + file.Name())
		if err != nil {
			log.Error(err, "Unable to read file")
			return err
		}
		err = yaml.Unmarshal(yamlBytes, u)
		if err != nil {
			log.Error(err, "Unable to unmarshal yaml")
			return err
		}
		if u.GetKind() != "CustomResourceDefinition" {
			specCopy, _, err := unstructured.NestedFieldCopy(u.Object, "spec")
			if err != nil {
				log.Error(err, "Unable to make a copy of the spec")
				return err
			}

			_, err = controllerutil.CreateOrUpdate(context.TODO(), c, u, func() error {
				return unstructured.SetNestedField(u.Object, specCopy, "spec")
			})
			if err != nil {
				log.Error(err, "Unable persist object to kubernetes")
				return err
			}

		}
	}

	return nil
}
