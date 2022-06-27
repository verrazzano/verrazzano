// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package adapter

import (
	"context"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/modules"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type componentAdapter struct {
	IsEnabled bool
	Name      string
	Namespace string

	ChartNamespace   string
	ChartPath        string
	InstallOverrides vzapi.InstallOverrides
}

func NewAdapter(enabled bool) *componentAdapter {
	return &componentAdapter{
		IsEnabled: enabled,
	}
}

func (c *componentAdapter) spec() modulesv1alpha1.ModuleSpec {
	return modulesv1alpha1.ModuleSpec{
		Installer: modulesv1alpha1.ModuleInstaller{
			HelmChart: &modulesv1alpha1.HelmChart{
				Name:      c.Name,
				Namespace: c.ChartNamespace,
				Repository: modulesv1alpha1.HelmRepository{
					Path: c.ChartPath,
				},
				InstallOverrides: c.InstallOverrides,
			},
		},
	}
}

func (c *componentAdapter) createOrUpdate(client clipkg.Client) error {
	if !c.IsEnabled {
		return nil
	}

	module := &modulesv1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), client, module, func() error {
		if module.Labels == nil {
			module.Labels = map[string]string{}
		}
		module.Labels[modules.ControllerLabel] = c.Name
		module.Spec = c.spec()
		return nil
	}); err != nil {
		return err
	}
	return nil
}
