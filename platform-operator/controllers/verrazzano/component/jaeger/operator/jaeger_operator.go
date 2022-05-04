// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	deploymentName = "jaeger-operator"
	templateFile   = "/jaeger/jaeger-operator.yaml"
)

var subcomponentNames = []string{
	"jaeger-ingester",
	"jaeger-agent",
	"jaeger-query",
	"jaeger-collector",
	"jaeger-operator",
}

func componentInstall(ctx spi.ComponentContext) error {
	args, err := buildInstallArgs()
	if err != nil {
		return err
	}

	// Apply Jaeger Operator
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), "")
	if err := yamlApplier.ApplyFT(path.Join(config.GetThirdPartyManifestsDir(), templateFile), args); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to install Jaeger Operator: %v", err)
	}
	return nil
}

func buildInstallArgs() (map[string]interface{}, error) {
	args := map[string]interface{}{
		"namespace": constants.VerrazzanoMonitoringNamespace,
	}
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return args, err
	}
	for _, subcomponent := range subcomponentNames {
		if err := setImageOverride(args, bomFile, subcomponent); err != nil {
			return args, err
		}
	}
	return args, nil
}

func setImageOverride(args map[string]interface{}, bomFile bom.Bom, subcomponent string) error {
	images, err := bomFile.GetImageNameList(subcomponent)
	if err != nil {
		return err
	}
	if len(images) != 1 {
		return fmt.Errorf("expected 1 %s image, got %d", subcomponent, len(images))
	}

	args[strings.ReplaceAll(subcomponent, "-", "")] = images[0]
	return nil
}

// isJaegerOperatorReady checks if the Jaeger operator deployment is ready
func isJaegerOperatorReady(context spi.ComponentContext) bool {
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, componentPrefix)
}

func ensureVerrazzanoMonitoringNamespace(ctx spi.ComponentContext) error {
	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Jaeger Operator", ComponentNamespace)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}
