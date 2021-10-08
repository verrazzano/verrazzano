// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"go.uber.org/zap"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func (i IstioComponent) IsOperatorInstallSupported() bool {
	return true
}

func (i IstioComponent) IsInstalled(context spi.ComponentContext) (bool, error) {
	return istio.IsInstalled(context.Log())
}

func (i IstioComponent) Install(compContext spi.ComponentContext) error {
	const imagePullSecretHelmKey = "values.global.imagePullSecrets[0].name"
	var tmpFile *os.File
	var kvs []bom.KeyValue
	var err error
	cr := compContext.EffectiveCR()
	log := compContext.Log()
	client := compContext.Client()

	// Only create override file if the CR has an Istio component
	if cr.Spec.Components.Istio != nil {
		istioOperatorYaml, err := BuildIstioOperatorYaml(cr.Spec.Components.Istio)
		if err != nil {
			log.Errorf("Failed to Build IstioOperator YAML: %v", err)
			return err
		}

		// Write the overrides to a tmp file
		tmpFile, err := ioutil.TempFile(os.TempDir(), "istio-*.yaml")
		if err != nil {
			log.Errorf("Failed to create temporary file for Istio install: %v", err)
			return err
		}
		defer os.Remove(tmpFile.Name())
		if _, err = tmpFile.Write([]byte(istioOperatorYaml)); err != nil {
			log.Errorf("Failed to write to temporary file: %v", err)
			return err
		}
		if err := tmpFile.Close(); err != nil {
			log.Errorf("Failed to close temporary file: %v", err)
			return err
		}
		log.Infof("Created values file from Istio install args: %s", tmpFile.Name())
	}

	//// check for global image pull secret
	//kvs, err = AddGlobalImagePullSecretHelmOverride(log, client, IstioNamespace, kvs, imagePullSecretHelmKey)
	//if err != nil {
	//	return err
	//}

	// Build comma separated string of overrides that will be passed to
	// isioctl as --set values.
	// This include BOM image overrides as well as other overrides
	overrideStrings, err := buildOverridesString(log, client, IstioNamespace, kvs...)
	if err != nil {
		return err
	}

	if tmpFile == nil {
		_, _, err := installFunc(log, overrideStrings, i.ValuesFile)
		if err != nil {
			return err
		}
	} else {
		_, _, err := installFunc(log, overrideStrings, i.ValuesFile, tmpFile.Name())
		if err != nil {
			return err
		}
	}

	return nil
}

func (i IstioComponent) PreInstall(compContext spi.ComponentContext) error {
	const IstioCertSecret = "cert"
	log := compContext.Log()
	if compContext.IsDryRun() {
		return nil
	}

	// Ensure Istio namespace exists and label it for network policies
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: IstioNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = IstioNamespace
		return nil
	}); err != nil {
		return err
	}

	// Create the cert used by Istio MTLS if it doesn't exist
	var secret v1.Secret
	if err := compContext.Client().Get(context.TODO(), types.NamespacedName{Namespace: IstioNamespace, Name: IstioCertSecret}, &secret); err == nil {
		if !errors.IsNotFound(err) {
			// Unexpected error
			return err
		}
		// Secret not found - create it
		certScript := filepath.Join(config.GetInstallDir(), "create-istio-cert.sh")
		if _, stderr, err := vzos.RunBash(certScript); err != nil {
			log.Errorf("Failed creating Istio certificate secret %s: %s", err, stderr)
			return err
		}
	}

	return nil
}

func (i IstioComponent) PostInstall(context spi.ComponentContext) error {
	return nil
}

type installFuncSig func(log *zap.SugaredLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

// installFunc is the default install function
var installFunc installFuncSig = istio.Install

//func setInstallFunc(f installFuncSig) {
//	installFunc = f
//}
//
//func setDefaultInstallFunc() {
//	installFunc = istio.Install
//}

// AddGlobalImagePullSecretHelmOverride Adds a helm override Key if the global image pull secret exists and was copied successfully to the target namespace
//func addGlobalImagePullSecretHelmOverride(log *zap.SugaredLogger, client clipkg.Client, ns string, kvs []bom.KeyValue, keyName string) ([]bom.KeyValue, error) {
//	secretExists, err := secret.CheckImagePullSecret(client, ns)
//	if err != nil {
//		log.Errorf("Error copying global image pull secret %s to %s namespace", constants.GlobalImagePullSecName, ns)
//		return kvs, err
//	}
//	if secretExists {
//		kvs = append(kvs, bom.KeyValue{
//			Key:   keyName,
//			Value: constants.GlobalImagePullSecName,
//		})
//	}
//	return kvs, nil
//}
