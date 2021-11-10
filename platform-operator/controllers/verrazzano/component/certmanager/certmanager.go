// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return err
	}

	// get dns type from config
	// patch crd if necessary
	return nil
}

func (c certManagerComponent) Install(compContext spi.ComponentContext) error {
	// kubectl install crd
	// helm install and intermediate steps
	return nil
}

func (c certManagerComponent) PostInstall(compContext spi.ComponentContext) error {
	// setup component issuer
	return nil
}

// isCertManagerEnabled returns true if the WebLogic is enabled, which is the default
func isCertManagerEnabled(compContext spi.ComponentContext) bool {
	comp := compContext.EffectiveCR().Spec.Components.CertManager
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// AppendOverrides Build the set of cert-manager overrides for the helm install
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	return []bom.KeyValue{}, nil
}
