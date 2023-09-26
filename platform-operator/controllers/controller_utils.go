// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8serrors "github.com/verrazzano/verrazzano/pkg/k8s/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// Name of Effective Configmap Data Key
const effConfigKey = "effective-config.yaml"

// Suffix of the Name of the Configmap containing effective CR
const effConfigSuffix = "-effective-config"

// VzContainsResource checks to see if the resource is listed in the Verrazzano
func VzContainsResource(ctx spi.ComponentContext, objectName string, objectKind string) (string, bool) {
	for _, component := range registry.GetComponents() {
		if component.MonitorOverrides(ctx) {
			if found := componentContainsResource(component.GetOverrides(ctx.EffectiveCR()).([]installv1alpha1.Overrides), objectName, objectKind); found {
				return component.Name(), found
			}
		}
	}
	return "", false
}

// componentContainsResource looks through the component override list see if the resource is listed
func componentContainsResource(Overrides []installv1alpha1.Overrides, objectName string, objectKind string) bool {
	for _, override := range Overrides {
		if objectKind == constants.ConfigMapKind && override.ConfigMapRef != nil {
			if objectName == override.ConfigMapRef.Name {
				return true
			}
		}
		if objectKind == constants.SecretKind && override.SecretRef != nil {
			if objectName == override.SecretRef.Name {
				return true
			}
		}
	}
	return false
}

// UpdateVerrazzanoForInstallOverrides mutates the status subresource of Verrazzano Custom Resource specific
// to a component to cause a reconcile
func UpdateVerrazzanoForInstallOverrides(statusUpdater vzstatus.Updater, componentCtx spi.ComponentContext, componentName string) error {
	cr := componentCtx.ActualCR()
	// Return an error to requeue if Verrazzano Component Status hasn't been initialized
	if cr.Status.Components == nil {
		return fmt.Errorf("Components not initialized")
	}
	// Set ReconcilingGeneration to 1 to re-enter install flow
	details := cr.Status.Components[componentName].DeepCopy()
	details.ReconcilingGeneration = 1
	componentsToUpdate := map[string]*installv1alpha1.ComponentStatusDetails{
		componentName: details,
	}
	statusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: cr,
		Components: componentsToUpdate,
	})
	return nil
}

// ProcDeletedOverride checks Verrazzano CR for an override resource that has now been deleted,
// and updates the CR if the resource is found listed as an override
func ProcDeletedOverride(statusUpdater vzstatus.Updater, c client.Client, vz *installv1alpha1.Verrazzano, objectName string, objectKind string) error {
	// DefaultLogger is used since we only need to create a component context and any actual logging isn't being performed
	log := vzlog.DefaultLogger()
	ctx, err := spi.NewContext(log, c, vz, nil, false)
	if err != nil {
		return err
	}

	compName, ok := VzContainsResource(ctx, objectName, objectKind)
	if !ok {
		return nil
	}

	if err := UpdateVerrazzanoForInstallOverrides(statusUpdater, ctx, compName); err != nil {
		return err
	}
	return nil
}

// CreateOrUpdateEffectiveConfigCM takes in the Actual CR, retrieves the Effective CR,
// converts it into YAML and stores it in a configmap If no configmap exists,
// it will create one, otherwise it updates the configmap with the effective CR
func CreateOrUpdateEffectiveConfigCM(ctx context.Context, c client.Client, vz *installv1alpha1.Verrazzano) error {

	//In the case of verrazzano uninstall,the reconciler re-creates the config map
	//when the vz status is either uninstalling or uninstall completely then do not create anything
	var currentCondition installv1alpha1.ConditionType
	if len(vz.Status.Conditions) > 0 {
		currentCondition = vz.Status.Conditions[len(vz.Status.Conditions)-1].Type
	}
	if currentCondition == installv1alpha1.CondUninstallComplete || currentCondition == installv1alpha1.CondUninstallStarted {
		return nil
	}
	// Get the Effective CR from the Verrazzano CR supplied and convert it into v1beta1
	v1beta1ActualCR := &v1beta1.Verrazzano{}
	err := vz.ConvertTo(v1beta1ActualCR)
	if err != nil {
		return fmt.Errorf("failed Converting v1alpha1 Verrazzano to v1beta1: %v", err)
	}

	effCR, err := transform.GetEffectiveV1beta1CR(v1beta1ActualCR)
	if err != nil {
		return fmt.Errorf("failed retrieving the Effective CR: %v", err)
	}

	// Marshal Indent it to format the Effective CR Specs into YAML
	effCRSpecs, err := yaml.Marshal(effCR.Spec)
	if err != nil {
		return fmt.Errorf("failed to convert effective CR into YAML: %v", err)
	}

	// Create a Configmap Object that stores the Effective CR Specs
	effCRConfigmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      (vz.ObjectMeta.Name + effConfigSuffix),
			Namespace: (vz.ObjectMeta.Namespace),
		},
	}

	// Update the configMap if a ConfigMap already exists
	// In case, there's no ConfigMap, the IsNotFound() func will return true and then it will create one.
	_, err = controllerutil.CreateOrUpdate(ctx, c, effCRConfigmap, func() error {

		effCRConfigmap.Data = map[string]string{effConfigKey: string(effCRSpecs)}
		return nil
	})
	if k8serrors.IsAlreadyExists(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to Create or Update the configmap: %v", err)
	}

	return nil
}
