// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"encoding/json"
	"fmt"

	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	modulehelm "github.com/verrazzano/verrazzano-modules/pkg/helm"
	modulelog "github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/yaml"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type overrideType string

const (
	secretType     overrideType = "secret"
	configMapType  overrideType = "configmap"
	vzOverridesKey              = "verrazzano-generated-overrides"
)

// setModuleValues sets the Module values and valuesFrom fields.
// All VZ CR config override secrets or configmaps need to be copied to the module namespace
func (r Reconciler) setModuleValues(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano, module *moduleapi.Module, comp componentspi.Component) error {

	// Stores component/module overrides based on Verrazzano CR configuration and stores them in an overrides secret
	valuesFromCR, err := r.createVerrazzanoCROverridesForModule(comp, effectiveCR, module)
	if err != nil {
		return err
	}

	module.Spec.Values = nil
	module.Spec.ValuesFrom = valuesFromCR

	// Use the Actual VZ CR instance to get the component user overrides list (either v1alpha1 or v1beta1)
	compOverrideList := comp.GetOverrides(actualCR)
	switch castType := compOverrideList.(type) {
	case []vzv1alpha1.Overrides:
		overrideList := castType
		for _, o := range overrideList {
			var b vzv1beta1.Overrides
			b.Values = o.Values
			b.SecretRef = o.SecretRef
			b.ConfigMapRef = o.ConfigMapRef
			if err := r.setModuleValuesForOneOverride(log, b, actualCR, effectiveCR, module); err != nil {
				return err
			}
		}

	case []vzv1beta1.Overrides:
		overrideList := castType
		for _, o := range overrideList {
			if err := r.setModuleValuesForOneOverride(log, o, actualCR, effectiveCR, module); err != nil {
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

func (r Reconciler) createVerrazzanoCROverridesForModule(comp spi.Component, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module) ([]moduleapi.ValuesFromSource, error) {
	overridesAsHelmValues, err := comp.GetModuleConfigAsHelmValues(effectiveCR)
	if err != nil || overridesAsHelmValues == nil || overridesAsHelmValues.Size() == 0 {
		return []moduleapi.ValuesFromSource{}, err
	}

	overridesSecretName := fmt.Sprintf("%s-overrides", module.Name)
	overridesSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      overridesSecretName,
			Namespace: module.Namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, overridesSecret, func() error {
		overridesSecret.StringData = map[string]string{
			vzOverridesKey: string(overridesAsHelmValues.Raw),
		}
		return nil
	}); err != nil {
		return []moduleapi.ValuesFromSource{}, err
	}

	//optional := true
	overridesValue := moduleapi.ValuesFromSource{
		SecretRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: overridesSecretName,
			},
			Key: vzOverridesKey,
			//Optional: &optional,
		},
	}
	return []moduleapi.ValuesFromSource{
		overridesValue,
	}, nil
}

// Set the module values or valuesFrom for a single override struct
func (r Reconciler) setModuleValuesForOneOverride(log vzlog.VerrazzanoLogger, overrides vzv1beta1.Overrides, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano, module *moduleapi.Module) error {

	if err := r.mergedModuleValuesOverrides(module, overrides); err != nil {
		return err
	}

	// Copy Secret overrides to new secret and add info to the module ValuesFrom
	if overrides.SecretRef != nil {
		secretName := getConfigResourceName(module.Spec.ModuleName, overrides.SecretRef.Name)
		if opResult, err := r.copySecret(overrides.SecretRef, secretName, module, effectiveCR.Namespace); err != nil {
			log.ErrorfThrottled("Failed to create values secret for module %s: %v", module.Name, err)
			return err
		} else if res := r.updateStatusIfNeeded(log, actualCR, opResult); res.IsError() {
			// If the copy operation resulted in an update to the target, set the VZ condition to install started/Reconciling
			return res.GetError()
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

		if opResult, err := r.copyConfigMap(overrides.ConfigMapRef, cmName, module, effectiveCR.Namespace); err != nil {
			log.ErrorfThrottled("Failed to create values configmap for module %s: %v", module.Name, err)
			return err
		} else if res := r.updateStatusIfNeeded(log, actualCR, opResult); res.IsError() {
			// If the copy operation resulted in an update to the target, set the VZ condition to install started/Reconciling
			return res.GetError()
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

func (r Reconciler) mergedModuleValuesOverrides(module *moduleapi.Module, overrides vzv1beta1.Overrides) error {
	if overrides.Values == nil {
		return nil
	}

	if module.Spec.Values == nil {
		module.Spec.Values = &apiextensionsv1.JSON{}
	}

	mergedValues := map[string]interface{}{}
	if module.Spec.Values.Size() > 0 {
		if err := json.Unmarshal(module.Spec.Values.Raw, &mergedValues); err != nil {
			return err
		}
	}

	newValues := map[string]interface{}{}
	if err := json.Unmarshal(overrides.Values.Raw, &newValues); err != nil {
		return err
	}

	if err := yaml.MergeMaps(mergedValues, newValues); err != nil {
		return err
	}
	mergedBytes, err := json.Marshal(mergedValues)
	if err != nil {
		return err
	}
	module.Spec.Values.Raw = mergedBytes
	return nil
}

// copy the component config secret to the module namespace and set the module as owner
func (r Reconciler) copySecret(secretRef *corev1.SecretKeySelector, secretName string, module *moduleapi.Module, fromSecretNamespace string) (controllerutil.OperationResult, error) {
	data, err := modulehelm.GetSecretOverrides(modulelog.DefaultLogger(), r.Client, secretRef, fromSecretNamespace)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: module.Namespace, Name: secretName},
	}
	opResult, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[secretRef.Key] = []byte(data)

		// Label the secret so we know what module owns it
		if secret.Labels == nil {
			secret.Labels = make(map[string]string)
		}
		secret.Labels[vzconst.VerrazzanoModuleOwnerLabel] = module.Spec.ModuleName
		return nil
	})

	return opResult, err
}

// copy the component configmap to the module namespace and set the module as owner
func (r Reconciler) copyConfigMap(cmRef *corev1.ConfigMapKeySelector, cmName string, module *moduleapi.Module, fromSecretNamespace string) (controllerutil.OperationResult, error) {
	data, err := modulehelm.GetConfigMapOverrides(modulelog.DefaultLogger(), r.Client, cmRef, fromSecretNamespace)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: module.Namespace, Name: cmName},
	}
	opResult, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[cmRef.Key] = data

		// Label the secret so we know what module owns it
		if cm.Labels == nil {
			cm.Labels = make(map[string]string)
		}
		cm.Labels[vzconst.VerrazzanoModuleOwnerLabel] = module.Spec.ModuleName
		return nil
	})
	return opResult, err
}

func getConfigResourceName(moduleName string, resourceName string) string {
	// suffix this to the secret and configmap for the module config.
	return fmt.Sprintf("%s-%s", moduleName, resourceName)
}

// getOverrideResourceNames returns the configuration override configMap or secret names used by the vz cr
func getOverrideResourceNames(effectiveCR *vzv1alpha1.Verrazzano, ovType overrideType) map[string]bool {
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
		case []vzv1alpha1.Overrides:
			overrideList := castType
			for _, o := range overrideList {
				if o.SecretRef != nil && ovType == secretType {
					names[o.SecretRef.Name] = true
				}
				if o.ConfigMapRef != nil && ovType == configMapType {
					names[o.ConfigMapRef.Name] = true
				}
			}
		case []vzv1beta1.Overrides:
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

// deleteConfigSecrets deletes all the module config secrets
func (r Reconciler) deleteConfigSecrets(log vzlog.VerrazzanoLogger, namespace string, moduleName string) result.Result {
	secretList := &corev1.SecretList{}
	req, _ := labels.NewRequirement(vzconst.VerrazzanoModuleOwnerLabel, selection.Equals, []string{moduleName})
	selector := labels.NewSelector().Add(*req)
	if err := r.Client.List(context.TODO(), secretList, &client.ListOptions{Namespace: namespace, LabelSelector: selector}); err != nil {
		log.Infof("Failed getting secrets in %s namespace, retrying: %v", namespace, err)
	}

	for i, s := range secretList.Items {
		err := r.Client.Delete(context.TODO(), &secretList.Items[i])
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.Errorf("Failed deleting secret %s/%s, retrying: %v", namespace, s.Name, err)
		}
	}
	return result.NewResult()
}

// deleteConfigMaps deletes all the module config maps
func (r Reconciler) deleteConfigMaps(log vzlog.VerrazzanoLogger, namespace string, moduleName string) result.Result {
	configMapList := &corev1.ConfigMapList{}
	req, _ := labels.NewRequirement(vzconst.VerrazzanoModuleOwnerLabel, selection.Equals, []string{moduleName})
	selector := labels.NewSelector().Add(*req)
	if err := r.Client.List(context.TODO(), configMapList, &client.ListOptions{Namespace: namespace, LabelSelector: selector}); err != nil {
		log.Infof("Failed getting configMaps in %s namespace, retrying: %v", namespace, err)
	}

	for i, s := range configMapList.Items {
		err := r.Client.Delete(context.TODO(), &configMapList.Items[i])
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.Errorf("Failed deleting configMap %s/%s, retrying: %v", namespace, s.Name, err)
		}
	}
	return result.NewResult()
}
