// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherbackup

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"path"
)

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	deploymentName         = "rancher-backup"
	imagePullSecretHelmKey = "image.imagePullSecrets[0]"
	crdRelativePath        = "rancher-backup-crd/templates"
)

// isRancherBackupOperatorReady checks if the Velero deployment is ready
func isRancherBackupOperatorReady(context spi.ComponentContext) bool {
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, componentPrefix)
}

// AppendOverrides appends Helm value overrides for the Velero component's Helm chart
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	arguments := []bom.KeyValue{}
	kvs = append(kvs, arguments...)
	return kvs, nil
}

// GetOverrides gets the install overrides
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.RancherBackup != nil {
		return effectiveCR.Spec.Components.RancherBackup.ValueOverrides
	}
	return []vzapi.Overrides{}
}

func ensureVeleroNamespace(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Creating namespace %s for Rancher Backup.", ComponentNamespace)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

func rancherBackupCrdInstall(ctx spi.ComponentContext) error {
	// Apply Rancher Backup Operator CRDS
	err := ensureVeleroNamespace(ctx)
	if err != nil {
		return err
	}
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), ComponentNamespace)
	crdPath := path.Join(config.GetThirdPartyDir(), crdRelativePath)
	if err := yamlApplier.ApplyFT(path.Join(config.GetThirdPartyDir(), crdPath), nil); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to deploy rancher-backup crds: %v", err)
	}
	return nil
}
