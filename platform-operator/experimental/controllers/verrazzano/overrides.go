// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulehelm "github.com/verrazzano/verrazzano-modules/pkg/helm"
	modulelog "github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapibeta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type overrideType string

const (
	secretType    overrideType = "secret"
	configMapType overrideType = "configmap"
)

// setModuleValues sets the Module values and valuesFrom fields.
// All VZ CR config override secrets or configmaps need to be copied to the module namespace
func (r Reconciler) setModuleValues(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module, comp spi.Component) error {
	module.Spec.Values = nil
	module.Spec.ValuesFrom = nil

	// Get component override list (either v1alpha1 or v1beta1)
	compOverrideList := comp.GetOverrides(effectiveCR)
	switch castType := compOverrideList.(type) {
	case []vzapi.Overrides:
		overrideList := castType
		for _, o := range overrideList {
			var b vzapibeta1.Overrides
			b.Values = o.Values
			b.SecretRef = o.SecretRef
			b.ConfigMapRef = o.ConfigMapRef
			if err := r.setModuleValuesForOneOverride(log, b, effectiveCR, module); err != nil {
				return err
			}
		}

	case []vzapibeta1.Overrides:
		overrideList := castType
		for _, o := range overrideList {
			if err := r.setModuleValuesForOneOverride(log, o, effectiveCR, module); err != nil {
				return err
			}
		}
	default:
		err := fmt.Errorf("Failed, component %s Overrides is not a known type", comp.Name())
		log.Error(err)
		return err
	}
	return nil
}

// Set the module values or valuesFrom for a single override struct
func (r Reconciler) setModuleValuesForOneOverride(log vzlog.VerrazzanoLogger, overrides vzapibeta1.Overrides, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module) error {
	if overrides.Values != nil {
		// TODO - need to combine with existing values
		module.Spec.Values = overrides.Values
	}

	// Copy Secret overrides to new secret and add info to the module ValuesFrom
	if overrides.SecretRef != nil {
		secretName := getConfigResourceName(module.Spec.ModuleName, overrides.SecretRef.Name)
		if err := r.copySecret(overrides.SecretRef, secretName, module, effectiveCR.Namespace); err != nil {
			log.ErrorfThrottled("Failed to create values secret for module %s: %v", module.Name, err)
			return err
		}
		module.Spec.ValuesFrom = append(module.Spec.ValuesFrom, moduleapi.ValuesFromSource{
			SecretRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key:      overrides.SecretRef.Key,
				Optional: overrides.SecretRef.Optional,
			},
		})
	}

	// Copy ConfigMap overrides to new CM and add info to the module ValuesFrom
	if overrides.ConfigMapRef != nil {
		cmName := getConfigResourceName(module.Spec.ModuleName, overrides.ConfigMapRef.Name)

		if err := r.copyConfigMap(overrides.ConfigMapRef, cmName, module, effectiveCR.Namespace); err != nil {
			log.ErrorfThrottled("Failed to create values configmap for module %s: %v", module.Name, err)
			return err
		}
		module.Spec.ValuesFrom = append(module.Spec.ValuesFrom, moduleapi.ValuesFromSource{
			ConfigMapRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
				},
				Key:      overrides.ConfigMapRef.Key,
				Optional: overrides.ConfigMapRef.Optional,
			},
		})
	}

	return nil
}

// copy the component config secret to the module namespace and set the module as owner
func (r Reconciler) copySecret(secretRef *corev1.SecretKeySelector, secretName string, module *moduleapi.Module, fromSecretNamespace string) error {
	data, err := modulehelm.GetSecretOverrides(modulelog.DefaultLogger(), r.Client, secretRef, fromSecretNamespace)
	if err != nil {
		return err
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: module.Namespace, Name: secretName},
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[secretRef.Key] = []byte(data)

		// Label the secret so we know what module owns it
		if secret.Labels == nil {
			secret.Labels = make(map[string]string)
		}
		secret.Labels[constants.VerrazzanoModuleOwnerLabel] = module.Spec.ModuleName
		return nil
	})

	return err
}

// copy the component configmap to the module namespace and set the module as owner
func (r Reconciler) copyConfigMap(cmRef *corev1.ConfigMapKeySelector, cmName string, module *moduleapi.Module, fromSecretNamespace string) error {
	data, err := modulehelm.GetConfigMapOverrides(modulelog.DefaultLogger(), r.Client, cmRef, fromSecretNamespace)
	if err != nil {
		return err
	}
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: module.Namespace, Name: cmName},
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[cmRef.Key] = data

		// Label the secret so we know what module owns it
		if cm.Labels == nil {
			cm.Labels = make(map[string]string)
		}
		cm.Labels[constants.VerrazzanoModuleOwnerLabel] = module.Spec.ModuleName
		return nil
	})

	return err
}

func getConfigResourceName(moduleName string, resourceName string) string {
	// suffix this to the secret and configmap for the module config.
	return fmt.Sprintf("%s-%s", moduleName, resourceName)
}

// getOverrideResourceNames returns the configuration override configMap or secret names used by the vz cr
func getOverrideResourceNames(effectiveCR *vzapi.Verrazzano, ovType overrideType) map[string]bool {
	names := make(map[string]bool)

	for _, comp := range registry.GetComponents() {
		if !comp.ShouldUseModule() {
			continue
		}
		if !comp.IsEnabled(effectiveCR) {
			continue
		}
		compOverrideList := comp.GetOverrides(effectiveCR)
		switch castType := compOverrideList.(type) {
		case []vzapi.Overrides:
			overrideList := castType
			for _, o := range overrideList {
				if o.SecretRef != nil && ovType == secretType {
					names[o.SecretRef.Name] = true
				}
				if o.ConfigMapRef != nil && ovType == configMapType {
					names[o.ConfigMapRef.Name] = true
				}
			}
		case []vzapibeta1.Overrides:
			overrideList := castType
			for _, o := range overrideList {
				if o.SecretRef != nil && ovType == secretType {
					names[o.SecretRef.Name] = true
				}
				if o.ConfigMapRef != nil && ovType == configMapType {
					names[o.ConfigMapRef.Name] = true
				}
			}
		}
	}
	return names
}
