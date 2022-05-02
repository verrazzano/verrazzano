// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	vzsecret "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	system = "system"
)

// ResourceRequestValues defines the storage information that will be passed to VMI instance
type ResourceRequestValues struct {
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage"` // Empty string allowed
}

// VMIMutateFunc is the function used to populate the components in VMI
type VMIMutateFunc func(ctx spi.ComponentContext, storage *ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, existingVMI *vmov1.VerrazzanoMonitoringInstance) error

// NewVMI creates a new VerrazzanoMonitoringInstance object with default values
func NewVMI() *vmov1.VerrazzanoMonitoringInstance {
	return &vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      system,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
	}
}

// CreateOrUpdateVMI instantiates the VMI resource
func CreateOrUpdateVMI(ctx spi.ComponentContext, updateFunc VMIMutateFunc) error {
	if !vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		return nil
	}

	effectiveCR := ctx.EffectiveCR()

	var dnsSuffix string
	var envName string
	var err error
	if vzconfig.IsNGINXEnabled(effectiveCR) {
		dnsSuffix, err = vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed getting DNS suffix: %v", err)
		}
		envName = vzconfig.GetEnvName(effectiveCR)
	}

	storage, err := FindStorageOverride(ctx.EffectiveCR())
	if err != nil {
		return ctx.Log().ErrorfNewErr("failed to get storage overrides: %v", err)
	}
	vmi := NewVMI()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), vmi, func() error {
		var existingVMI *vmov1.VerrazzanoMonitoringInstance = nil
		if len(vmi.Spec.SecretsName) > 0 {
			existingVMI = vmi.DeepCopy()
		}

		vmi.Labels = map[string]string{
			"k8s-app":            "verrazzano.io",
			"verrazzano.binding": system,
		}
		if vzconfig.IsNGINXEnabled(effectiveCR) {
			vmi.Spec.URI = fmt.Sprintf("vmi.system.%s.%s", envName, dnsSuffix)
			vmi.Spec.IngressTargetDNSName = fmt.Sprintf("verrazzano-ingress.%s.%s", envName, dnsSuffix)
		}
		vmi.Spec.ServiceType = "ClusterIP"
		vmi.Spec.AutoSecret = true
		vmi.Spec.SecretsName = constants.VMISecret
		vmi.Spec.CascadingDelete = true
		return updateFunc(ctx, storage, vmi, existingVMI)
	})
	if err != nil {
		return ctx.Log().ErrorfNewErr("failed to update VMI: %v", err)
	}
	return nil
}

// EnsureVMISecret creates or updates the VMI secret
func EnsureVMISecret(cli client.Client) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VMISecret,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), cli, secret, func() error {
		if secret.Data["username"] == nil || secret.Data["password"] == nil {
			secret.Data["username"] = []byte(constants.VMISecret)
			pw, err := password.GeneratePassword(16)
			if err != nil {
				return err
			}
			secret.Data["password"] = []byte(pw)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// EnsureGrafanaAdminSecret creates or updates the Grafana admin secret
func EnsureGrafanaAdminSecret(cli client.Client) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.GrafanaSecret,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), cli, secret, func() error {
		if secret.Data["username"] == nil || secret.Data["password"] == nil {
			secret.Data["username"] = []byte(constants.VMISecret)
			pw, err := password.GeneratePassword(32)
			if err != nil {
				return err
			}
			secret.Data["password"] = []byte(pw)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// EnsureBackupSecret creates or updates the VMI backup secret
func EnsureBackupSecret(cli client.Client) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.VMIBackupSecretName,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), cli, secret, func() error {
		// Populating dummy keys for access and secret key so that they are never empty
		if secret.Data[constants.ObjectStoreAccessKey] == nil || secret.Data[constants.ObjectStoreAccessSecretKey] == nil {
			key, err := password.GeneratePassword(32)
			if err != nil {
				return err
			}
			secret.Data[constants.ObjectStoreAccessKey] = []byte(key)
			secret.Data[constants.ObjectStoreAccessSecretKey] = []byte(key)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// FindStorageOverride finds and returns the correct storage override from the effective CR
func FindStorageOverride(effectiveCR *vzapi.Verrazzano) (*ResourceRequestValues, error) {
	if effectiveCR == nil || effectiveCR.Spec.DefaultVolumeSource == nil {
		return nil, nil
	}
	defaultVolumeSource := effectiveCR.Spec.DefaultVolumeSource
	if defaultVolumeSource.EmptyDir != nil {
		return &ResourceRequestValues{
			Storage: "",
		}, nil
	}
	if defaultVolumeSource.PersistentVolumeClaim != nil {
		pvcClaim := defaultVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplate(pvcClaim.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			return nil, fmt.Errorf("Failed, did not find matching storage volume template for claim %s", pvcClaim.ClaimName)
		}
		storageString := storageSpec.Resources.Requests.Storage().String()
		return &ResourceRequestValues{
			Storage: storageString,
		}, nil
	}
	return nil, fmt.Errorf("Failed, unsupported volume source: %v", defaultVolumeSource)
}

// IsVMISecretReady returns true if the VMI secret is present in the system namespace
func IsVMISecretReady(ctx spi.ComponentContext) bool {
	if err := ctx.Client().Get(context.TODO(),
		types.NamespacedName{Name: "verrazzano", Namespace: globalconst.VerrazzanoSystemNamespace},
		&v1.Secret{}); err != nil {
		if !errors.IsNotFound(err) {
			ctx.Log().Errorf("Failed, unexpected error getting verrazzano secret: %v", err)
			return false
		}
		ctx.Log().Debugf("Verrazzano secret not found")
		return false
	}
	return true
}

// CreateAndLabelVMINamespaces creates and labels the namespaces needed for the VMI resources
func CreateAndLabelVMINamespaces(ctx spi.ComponentContext) error {
	if err := namespace.CreateVerrazzanoSystemNamespace(ctx.Client()); err != nil {
		return err
	}
	if _, err := vzsecret.CheckImagePullSecret(ctx.Client(), globalconst.VerrazzanoSystemNamespace); err != nil {
		return ctx.Log().ErrorfNewErr("Failed checking for image pull secret: %v", err)
	}
	return nil
}

// CompareStorageOverrides compares storage override settings for the VMI components
func CompareStorageOverrides(old *vzapi.Verrazzano, new *vzapi.Verrazzano, jsonName string) error {
	// compare the storage overrides and reject if the type or size is different
	oldSetting, err := FindStorageOverride(old)
	if err != nil {
		return err
	}
	newSetting, err := FindStorageOverride(new)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(oldSetting, newSetting) {
		return fmt.Errorf("Can not change volume settings for %s", jsonName)
	}
	return nil
}

// CheckIngressesAndCerts checks the Ingress and Certs for the VMI components in the Post- function
func CheckIngressesAndCerts(ctx spi.ComponentContext, comp spi.Component) error {
	prefix := fmt.Sprintf("Component %s", comp.Name())
	if !status.IngressesPresent(ctx.Log(), ctx.Client(), comp.GetIngressNames(ctx), prefix) {
		return ctrlerrors.RetryableError{
			Source:    comp.Name(),
			Operation: "Check if Ingresses are present",
		}
	}

	if readyStatus, certsNotReady := status.CertificatesAreReady(ctx.Client(), ctx.Log(), ctx.EffectiveCR(), comp.GetCertificateNames(ctx)); !readyStatus {
		ctx.Log().Progressf("Certificates not ready for component %s: %v", comp.Name(), certsNotReady)
		return ctrlerrors.RetryableError{
			Source:    comp.Name(),
			Operation: "Check if certificates are ready",
		}
	}
	return nil
}
