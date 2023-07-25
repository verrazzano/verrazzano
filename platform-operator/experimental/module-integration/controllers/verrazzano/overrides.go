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

// suffix this to the secret and configmap for the module config.
const suffix = "-values"

// copy the secret to the module namespace and set the module as owner
func (r Reconciler) setModuleOverrides(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module, comp spi.Component, vzVersion string, moduleVersion string) error {
	var beta1ov vzapibeta1.Overrides

	// Put overrides into v1beta1 overrides struct
	o := comp.GetOverrides(effectiveCR)
	switch o.(type) {
	case vzapi.Overrides:
		alpha1ov := o.(vzapi.Overrides)
		beta1ov.Values = alpha1ov.Values
		beta1ov.SecretRef = alpha1ov.SecretRef
		beta1ov.ConfigMapRef = alpha1ov.ConfigMapRef
	case vzapibeta1.Overrides:
		beta1ov = o.(vzapibeta1.Overrides)
	default:
		err := fmt.Errorf("Failed, component %s Overrides is not a known type", comp.Name())
		log.Error(err)
		return err
	}

	// Copy overrides to module.
	// Copy configmap and secret to Verrazzano install
	module.Spec.Values = beta1ov.Values
	if err := r.copySecret(beta1ov.SecretRef, module, effectiveCR.Namespace); err != nil {
		log.ErrorfThrottled("Failed to create values secret for module %s: err", module.Name, err)
		return err
	}
	if err := r.copyConfigMap(beta1ov.ConfigMapRef, module, effectiveCR.Namespace); err != nil {
		log.ErrorfThrottled("Failed to create values configmap for module %s: err", module.Name, err)
		return err
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
		ObjectMeta: metav1.ObjectMeta{Namespace: module.Namespace, Name: module.Name + suffix},
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
		ObjectMeta: metav1.ObjectMeta{Namespace: module.Namespace, Name: module.Name + suffix},
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
