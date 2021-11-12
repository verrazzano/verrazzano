// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	namespace = "cert-manager"
)

func (c certManagerComponent) PreInstall(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Infof("cert-manager PreInstall dry run")
		return nil
	}
	// create cert-manager namespace
	compContext.Log().Info("Adding label needed by network policies to cert-manager namespace")
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		return nil
	}); err != nil {
		compContext.Log().Errorf("Failed to create the cert-manager namespace: %s", err)
		return err
	}

	// Apply the cert-manager manifest
	compContext.Log().Info("Applying cert-manager crds")
	err := c.ApplyManifest(compContext)
	if err != nil {
		compContext.Log().Errorf("Failed to apply the cert-manager manifest: %s", err)
		return err
	}
	return nil
}

func (c certManagerComponent) PostInstall(compContext spi.ComponentContext) error {
	// setup component issuer
	return nil
}

// PatchManifest uses the patch file to patch the cert manager manifest
func (c certManagerComponent) ApplyManifest(compContext spi.ComponentContext) error {
	script := filepath.Join(config.GetInstallDir(), "apply-cert-manager-manifest.sh")

	if compContext.EffectiveCR().Spec.Components.DNS != nil && compContext.EffectiveCR().Spec.Components.DNS.OCI != nil {
		compContext.Log().Info("Patch cert-manager crds to use OCI DNS")
		err := os.Setenv("DNS_TYPE", "oci")
		if err != nil {
			compContext.Log().Errorf("Could not set DNS_TYPE environment variable: %s", err)
			return err
		}
	}
	if _, stderr, err := vzos.RunBash(script); err != nil {
		compContext.Log().Errorf("Failed to apply the cert-manager manifest %s: %s", err, stderr)
		return err
	}
	return nil
}

// isCertManagerEnabled returns true if the cert-manager is enabled, which is the default
func isCertManagerEnabled(compContext spi.ComponentContext) bool {
	comp := compContext.EffectiveCR().Spec.Components.CertManager
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// AppendOverrides Build the set of cert-manager overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()
	if namespace := cr.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace; namespace != "" {
		kvs = append(kvs, bom.KeyValue{Key: "clusterResourceNamespace", Value: namespace})
	}
	return kvs, nil
}
