// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"context"
	"fmt"

	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/verrazzano/verrazzano/application-operator/clients/oam/clientset/versioned/scheme"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	fmt.Println("Foo")
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
		yamlBytes, err := ioutil.ReadFile(path + "/" + file.Name())
		if err != nil {
			log.Error(err, "Unable to read file")
			return err
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode(yamlBytes, nil, nil)
		if err != nil {
			log.Error(err, "Unable to decode yaml")
			return err
		}

		if obj.GetObjectKind().GroupVersionKind().Kind == "CustomResourceDefinition" {

			_, err = controllerutil.CreateOrUpdate(context.TODO(), c, obj, func() error {
				return nil
			})
			if err != nil {
				log.Error(err, "Unable persist object to kubernetes")
				return err
			}
		}
	}
	for _, file := range files {
		yamlBytes, err := ioutil.ReadFile(path + "/" + file.Name())
		if err != nil {
			log.Error(err, "Unable to read file")
			return err
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode(yamlBytes, nil, nil)
		if err != nil {
			log.Error(err, "Unable to decode yaml")
			return err
		}

		if obj.GetObjectKind().GroupVersionKind().Kind != "CustomResourceDefinition" {

			_, err = controllerutil.CreateOrUpdate(context.TODO(), c, obj, func() error {
				return nil
			})
			if err != nil {
				log.Error(err, "Unable persist object to kubernetes")
				return err
			}
		}
	}

	return nil
}
