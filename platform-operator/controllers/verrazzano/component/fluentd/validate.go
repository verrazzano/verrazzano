// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	"fmt"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapibeta "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ES secret keys
	esUsernameKey = "username"
	esPasswordKey = "password"
)

// existing Fluentd mount paths can be found at platform-operator/helm_config/charts/verrazzano/templates/daemonset.yaml
var existingFluentdMountPaths = [7]string{
	"/fluentd/cacerts", "/fluentd/secret", "/fluentd/etc",
	"/root/.oci", "/var/log", "/var/lib", "/run/log/journal"}

var getControllerRuntimeClient = getClient

func validateFluentd(vz *vzapibeta.Verrazzano) error {
	fluentd := vz.Spec.Components.Fluentd
	if fluentd == nil {
		return nil
	}
	if err := validateExtraVolumeMounts(fluentd); err != nil {
		return err
	}
	if err := validateLogCollector(fluentd); err != nil {
		return err
	}
	return nil
}

func validateExtraVolumeMounts(fluentd *vzapibeta.FluentdComponent) error {
	if len(fluentd.ExtraVolumeMounts) > 0 {
		for _, vm := range fluentd.ExtraVolumeMounts {
			mountPath := vm.Source
			if vm.Destination != "" {
				mountPath = vm.Destination
			}
			for _, existing := range existingFluentdMountPaths {
				if mountPath == existing {
					return fmt.Errorf("duplicate mount path found: %s; Fluentd by default has mount paths: %v", mountPath, existingFluentdMountPaths)
				}
			}
		}
	}
	return nil
}

func validateLogCollector(fluentd *vzapibeta.FluentdComponent) error {
	if fluentd.OCI != nil && fluentd.OpenSearchURL != globalconst.DefaultOpensearchURL && fluentd.OpenSearchURL != "" {
		return fmt.Errorf("fluentd config does not allow both OCI %v and external Opensearch %v", fluentd.OCI, fluentd.OpenSearchURL)
	}
	if err := validateLogCollectorSecret(fluentd); err != nil {
		return err
	}
	return nil
}

func validateLogCollectorSecret(fluentd *vzapibeta.FluentdComponent) error {
	if len(fluentd.OpenSearchURL) > 0 && fluentd.OpenSearchURL != globalconst.VerrazzanoESInternal {
		cli, err := getControllerRuntimeClient()
		if err != nil {
			return err
		}
		secret := &corev1.Secret{}
		if err := getInstallSecret(cli, fluentd.OpenSearchURL, secret); err != nil {
			return err
		}
		if err := validateEntryExist(secret, esUsernameKey); err != nil {
			return err
		}
		return validateEntryExist(secret, esPasswordKey)
	}
	return nil
}

func validateEntryExist(secret *corev1.Secret, entry string) error {
	secretName := secret.Name
	_, ok := secret.Data[entry]
	if !ok {
		return fmt.Errorf("invalid Fluentd configuration, missing %s entry in secret \"%s\"", entry, secretName)
	}
	return nil
}

func getInstallSecret(client client.Client, secretName string, secret *corev1.Secret) error {
	err := client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: vzconst.VerrazzanoInstallNamespace}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("secret \"%s\" must be created in the \"%s\" namespace", secretName, vzconst.VerrazzanoInstallNamespace)
		}
		return err
	}
	return nil
}

// getClient returns a controller runtime client for the Verrazzano resource
func getClient() (client.Client, error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	return client.New(config, client.Options{Scheme: newScheme()})
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)
	return scheme
}
