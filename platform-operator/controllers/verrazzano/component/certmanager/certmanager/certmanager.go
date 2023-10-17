// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	certManagerDeploymentName = "cert-manager"
	cainjectorDeploymentName  = "cert-manager-cainjector"
	webhookDeploymentName     = "cert-manager-webhook"

	clusterResourceNamespaceKey = "clusterResourceNamespace"

	crdDirectory = "/cert-manager/"
	crdInputFile = "cert-manager.crds.yaml"

	extraArgsKey  = "extraArgs[0]"
	acmeSolverArg = "--acme-http01-solver-image="

	// Uninstall resources
	controllerConfigMap          = "cert-manager-controller"
	caInjectorConfigMap          = "cert-manager-cainjector-leader-election"
	caInjectorLeaderElectionRole = "cert-manager-cainjector:leaderelection"
	controllerLeaderElectionRole = "cert-manager:leaderelection"
)

var leaderElectionSystemResources = []client.Object{
	&v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: controllerConfigMap, Namespace: constants.KubeSystem}},
	&v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: caInjectorConfigMap, Namespace: constants.KubeSystem}},
	&rbac.Role{ObjectMeta: metav1.ObjectMeta{Name: controllerLeaderElectionRole, Namespace: constants.KubeSystem}},
	&rbac.Role{ObjectMeta: metav1.ObjectMeta{Name: caInjectorLeaderElectionRole, Namespace: constants.KubeSystem}},
	&rbac.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: controllerLeaderElectionRole, Namespace: constants.KubeSystem}},
	&rbac.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: caInjectorLeaderElectionRole, Namespace: constants.KubeSystem}},
}

// CertIssuerType identifies the certificate issuer type
type CertIssuerType string

// applyManifest uses the patch file to patch the cert manager manifest and apply it to the cluster
func (c certManagerComponent) applyManifest(compContext spi.ComponentContext) error {
	crdManifestDir := filepath.Join(config.GetThirdPartyManifestsDir(), crdDirectory)
	// Exclude the input file, since it will be parsed into individual documents
	// Input file containing CertManager CRDs
	inputFile := filepath.Join(crdManifestDir, crdInputFile)

	// Apply the CRD Manifest for CertManager
	if err := k8sutil.NewYAMLApplier(compContext.Client(), "").ApplyF(inputFile); err != nil {
		return compContext.Log().ErrorfNewErr("Failed applying CRD Manifests for CertManager: %v", err)

	}
	return nil
}

func cleanTempFiles(tempFiles ...string) error {
	for _, file := range tempFiles {
		if err := os.Remove(file); err != nil {
			return err
		}
	}
	return nil
}

// AppendOverrides Build the set of cert-manager overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// use image value for arg override
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, compContext.Log().ErrorNewErr("Failed to get the BOM file for the cert-manager image overrides: ", err)
	}

	images, err := bomFile.BuildImageOverrides("cert-manager")
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		if image.Key == extraArgsKey {
			kvs = append(kvs, bom.KeyValue{Key: extraArgsKey, Value: acmeSolverArg + image.Value})
		}
	}

	// Verify that we are using CA certs before appending override
	effectiveCR := compContext.EffectiveCR()
	clusterIssuerComponent := effectiveCR.Spec.Components.ClusterIssuer

	kvs = append(kvs, bom.KeyValue{Key: clusterResourceNamespaceKey, Value: clusterIssuerComponent.ClusterResourceNamespace})
	return kvs, nil
}

// isCertManagerReady checks the state of the expected cert-manager deployments and returns true if they are in a ready state
func (c certManagerComponent) isCertManagerReady(context spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return ready.DeploymentsAreReady(context.Log(), context.Client(), c.AvailabilityObjects.DeploymentNames, 1, prefix)
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.CertManager != nil {
			return effectiveCR.Spec.Components.CertManager.ValueOverrides
		}
		return []vzapi.Overrides{}
	}
	effectiveCR := object.(*v1beta1.Verrazzano)
	if effectiveCR.Spec.Components.CertManager != nil {
		return effectiveCR.Spec.Components.CertManager.ValueOverrides
	}
	return []v1beta1.Overrides{}
}
