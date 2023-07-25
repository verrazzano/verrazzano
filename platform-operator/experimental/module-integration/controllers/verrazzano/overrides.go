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
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// setModuleOverrides sets the Module values and valuesFrom fields.
// any VZ CR config override secrets or configmaps need to be copied to the module namespace
func (r Reconciler) setModuleValues(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module, comp spi.Component) error {
	// Put overrides into v1beta1 overrides struct
	compOverrideList := comp.GetOverrides(effectiveCR)
	switch compOverrideList.(type) {
	case []vzapi.Overrides:
		overrideList := compOverrideList.([]vzapi.Overrides)
		for _, o := range overrideList {
			var b vzapibeta1.Overrides
			b.Values = o.Values
			b.SecretRef = o.SecretRef
			b.ConfigMapRef = o.ConfigMapRef
			r.setOverrides(log, b, effectiveCR, module)
		}

	case []vzapibeta1.Overrides:
		overrideList := compOverrideList.([]vzapibeta1.Overrides)
		for _, o := range overrideList {
			r.setOverrides(log, o, effectiveCR, module)
		}
	default:
		err := fmt.Errorf("Failed, component %s Overrides is not a known type", comp.Name())
		log.Error(err)
		return err
	}
	return nil
}

func (r Reconciler) setOverrides(log vzlog.VerrazzanoLogger, overrides vzapibeta1.Overrides, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module) error {
	if overrides.Values != nil {
		// TODO - need to combine with existing values
		module.Spec.Values = overrides.Values
	}

	// Copy Secret overrides to new secret and add info to the module ValuesFrom
	if overrides.SecretRef != nil {
		if err := r.copySecret(overrides.SecretRef, module, effectiveCR.Namespace); err != nil {
			log.ErrorfThrottled("Failed to create values secret for module %s: err", module.Name, err)
			return err
		}
		module.Spec.ValuesFrom = append(module.Spec.ValuesFrom, moduleapi.ValuesFromSource{
			SecretRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: getConfigResourceName(module.Name),
				},
				Key:      overrides.SecretRef.Key,
				Optional: overrides.SecretRef.Optional,
			},
		})
	}

	// Copy ConfigMap overrides to new CM and add info to the module ValuesFrom
	if overrides.ConfigMapRef != nil {
		if err := r.copyConfigMap(overrides.ConfigMapRef, module, effectiveCR.Namespace); err != nil {
			log.ErrorfThrottled("Failed to create values configmap for module %s: err", module.Name, err)
			return err
		}
		module.Spec.ValuesFrom = append(module.Spec.ValuesFrom, moduleapi.ValuesFromSource{
			ConfigMapRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: getConfigResourceName(module.Name),
				},
				Key:      overrides.ConfigMapRef.Key,
				Optional: overrides.ConfigMapRef.Optional,
			},
		})
	}

	return nil
}

// copy the component config secret to the module namespace and set the module as owner
func (r Reconciler) copySecret(secretRef *corev1.SecretKeySelector, module *moduleapi.Module, fromSecretNamespace string) error {
	data, err := modulehelm.GetSecretOverrides(modulelog.DefaultLogger(), r.Client, secretRef, fromSecretNamespace)
	if err != nil {
		return err
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: module.Namespace, Name: getConfigResourceName(module.Name)},
	}
	controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[secretRef.Key] = []byte(data)
		return controllerutil.SetControllerReference(module, &secret, r.Scheme)
	})

	return nil
}

// copy the component configmap to the module namespace and set the module as owner
func (r Reconciler) copyConfigMap(cmRef *corev1.ConfigMapKeySelector, module *moduleapi.Module, fromSecretNamespace string) error {
	data, err := modulehelm.GetConfigMapOverrides(modulelog.DefaultLogger(), r.Client, cmRef, fromSecretNamespace)
	if err != nil {
		return err
	}
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: module.Namespace, Name: getConfigResourceName(module.Name)},
	}
	controllerutil.CreateOrUpdate(context.TODO(), r.Client, &cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[cmRef.Key] = data
		return controllerutil.SetControllerReference(module, &cm, r.Scheme)
	})
	return nil
}

func getConfigResourceName(moduleName string) string {
	// suffix this to the secret and configmap for the module config.
	const suffix = "-values"

	return moduleName + suffix
}
