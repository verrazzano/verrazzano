// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"io/fs"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"text/template"
)

var (
	// For Unit test purposes
	writeFileFunc = ioutil.WriteFile
)

func resetWriteFileFunc() {
	writeFileFunc = ioutil.WriteFile
}

const (
	deploymentName       = "jaeger-operator"
	tmpFilePrefix        = "jaeger-operator-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix
)

// Define the Jaeger images using extraEnv key.
// We need to replace image using the real image in the bom
const extraEnvValueTemplate = `extraEnv:
  - name: "JAEGER-AGENT-IMAGE"
    value: "{{.AgentImage}}"
  - name: "JAEGER-QUERY-IMAGE"
    value: "{{.QueryImage}}"
  - name: "JAEGER-COLLECTOR-IMAGE"
    value: "{{.CollectorImage}}"
  - name: "JAEGER-INGESTER-IMAGE"
    value: "{{.IngesterImage}}"
`

// imageData needed for template rendering
type imageData struct {
	AgentImage     string
	QueryImage     string
	CollectorImage string
	IngesterImage  string
}

// isjaegerOperatorReady checks if the Jaeger Operator deployment is ready
func isJaegerOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      deploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// PreInstall implementation for the Jaeger Operator Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Jaeger Operator PreInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace if not already created
	return ensureVerrazzanoMonitoringNamespace(ctx)
}

// AppendOverrides appends Helm value overrides for the Jaeger Operator component's Helm chart
// A go template is used to specify the Jaeger images using extraEnv key.
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	// Get jaeger-agent image
	agentImages, err := bomFile.BuildImageOverrides("jaeger-agent")
	if err != nil {
		return nil, err
	}
	if len(agentImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Agent, found %v", len(agentImages))
	}

	// Get jaeger-collector image
	collectorImages, err := bomFile.BuildImageOverrides("jaeger-collector")
	if err != nil {
		return nil, err
	}
	if len(collectorImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Collector, found %v", len(collectorImages))
	}

	// Get jaeger-query image
	queryImages, err := bomFile.BuildImageOverrides("jaeger-query")
	if err != nil {
		return nil, err
	}
	if len(queryImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Query, found %v", len(queryImages))
	}

	// Get jaeger-ingester image
	ingesterImages, err := bomFile.BuildImageOverrides("jaeger-ingester")
	if err != nil {
		return nil, err
	}
	if len(ingesterImages) != 1 {
		return nil, fmt.Errorf("component Jaeger Operator failed, expected 1 image for Jaeger Ingester, found %v", len(ingesterImages))
	}

	// use template to populate Jaeger images
	var b bytes.Buffer
	t, err := template.New("images").Parse(extraEnvValueTemplate)
	if err != nil {
		return nil, err
	}

	// Render the template
	data := imageData{AgentImage: agentImages[0].Value, CollectorImage: collectorImages[0].Value,
		QueryImage: queryImages[0].Value, IngesterImage: ingesterImages[0].Value}
	err = t.Execute(&b, data)
	if err != nil {
		return nil, err
	}

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(compContext, b.Bytes())
	if err != nil {
		return kvs, fmt.Errorf("failed generating Jaeger Operator overrides file: %v", err)
	}

	// Append any installArgs overrides
	kvs = append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})
	return kvs, nil
}

// validateJaegerOperator checks scenarios in which the Verrazzano CR violates install verification
// due to Jaeger Operator specifications
func (c jaegerOperatorComponent) validateJaegerOperator(vz *vzapi.Verrazzano) error {
	// Validate install overrides
	if vz.Spec.Components.JaegerOperator != nil {
		if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.JaegerOperator.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}

// GetOverrides returns the list of install overrides for a component
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.JaegerOperator != nil {
		return effectiveCR.Spec.Components.JaegerOperator.ValueOverrides
	}
	return []vzapi.Overrides{}
}

func ensureVerrazzanoMonitoringNamespace(ctx spi.ComponentContext) error {
	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Jaeger Operator", ComponentNamespace)
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

func generateOverridesFile(ctx spi.ComponentContext, contents []byte) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), tmpFileCreatePattern)
	if err != nil {
		return "", err
	}

	overridesFileName := file.Name()
	if err := writeFileFunc(overridesFileName, contents, fs.ModeAppend); err != nil {
		return "", err
	}
	ctx.Log().Debugf("Verrazzano jaeger-operator install overrides file %s contents: %s", overridesFileName,
		string(contents))
	return overridesFileName, nil
}
