// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	certManagerDeploymentName = "cert-manager"
	cainjectorDeploymentName  = "cert-manager-cainjector"
	webhookDeploymentName     = "cert-manager-webhook"

	clusterResourceNamespaceKey = "clusterResourceNamespace"

	crdDirectory  = "/cert-manager/"
	crdInputFile  = "cert-manager.crds.yaml"
	crdOutputFile = "output.crd.yaml"

	extraArgsKey  = "extraArgs[0]"
	acmeSolverArg = "--acme-http01-solver-image="

	// Uninstall resources
	controllerConfigMap          = "cert-manager-controller"
	caInjectorConfigMap          = "cert-manager-cainjector-leader-election"
	caInjectorLeaderElectionRole = "cert-manager-cainjector:leaderelection"
	controllerLeaderElectionRole = "cert-manager:leaderelection"
)

const snippetSubstring = "rfc2136:\n"

var ociDNSSnippet = strings.Split(`ocidns:
  description:
    ACMEIssuerDNS01ProviderOCIDNS is a structure containing
    the DNS configuration for OCIDNS DNS—Zone Record
    Management API
  properties:
    compartmentid:
      type: string
    ocizonename:
      type: string
    serviceAccountSecretRef:
      properties:
        key:
          description:
            The key of the secret to select from. Must be a
            valid secret key.
          type: string
        name:
          description: Name of the referent.
          type: string
      required:
        - name
      type: object
    useInstancePrincipals:
      type: boolean
  required:
    - ocizonename
  type: object`, "\n")

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
	// Output file format
	outputFile := filepath.Join(crdManifestDir, crdOutputFile)

	// Write out CRD Manifests for CertManager
	err := writeCRD(inputFile, outputFile, cmcommon.IsOCIDNS(compContext.EffectiveCR()))
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed writing CRD Manifests for CertManager: %v", err)
	}

	// Apply the CRD Manifest for CertManager
	if err = k8sutil.NewYAMLApplier(compContext.Client(), "").ApplyF(outputFile); err != nil {
		return compContext.Log().ErrorfNewErr("Failed applying CRD Manifests for CertManager: %v", err)

	}

	// Clean up the files written out. This may be different than the files applied
	return cleanTempFiles(outputFile)
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

// writeCRD writes out CertManager CRD manifests with OCI DNS specifications added
// reads the input CRD file line by line, adding OCI DNS snippets
func writeCRD(inFilePath, outFilePath string, useOCIDNS bool) error {
	infile, err := os.Open(inFilePath)
	if err != nil {
		return err
	}
	defer infile.Close()
	buffer := bytes.Buffer{}
	reader := bufio.NewReader(infile)

	// Flush the current buffer to the filesystem, creating a new manifest file
	flushBuffer := func() error {
		if buffer.Len() < 1 {
			return nil
		}
		outfile, err := os.Create(outFilePath)
		if err != nil {
			return err
		}
		if _, err := outfile.Write(buffer.Bytes()); err != nil {
			return err
		}
		if err := outfile.Close(); err != nil {
			return err
		}
		buffer.Reset()
		return nil
	}

	for {
		// Read the input file line by line
		line, err := reader.ReadBytes('\n')
		if err != nil {
			// If at the end of the file, flush any buffered data
			if err == io.EOF {
				flushErr := flushBuffer()
				return flushErr
			}
			return err
		}
		lineStr := string(line)
		// If the line specifies that the OCI DNS snippet should be written, write it
		if useOCIDNS && strings.HasSuffix(lineStr, snippetSubstring) {
			padding := strings.Repeat(" ", len(strings.TrimSuffix(lineStr, snippetSubstring)))
			snippet := createSnippetWithPadding(padding)
			if _, err := buffer.Write(snippet); err != nil {
				return err
			}
		}
		if _, err := buffer.Write(line); err != nil {
			return err
		}
	}
}

// createSnippetWithPadding left pads the OCI DNS snippet with a fixed amount of padding
func createSnippetWithPadding(padding string) []byte {
	builder := strings.Builder{}
	for _, line := range ociDNSSnippet {
		builder.WriteString(padding)
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	return []byte(builder.String())
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
