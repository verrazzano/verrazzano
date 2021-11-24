// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"go.uber.org/zap"
	"io/ioutil"
	istiosec "istio.io/api/security/v1beta1"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// IstioCertSecret is the secret name used for Istio MTLS certs
const IstioCertSecret = "cacerts"

// IstioEnvoyFilter is the Envoy header filter name
const IstioEnvoyFilter = "server-header-filter"

// Service name for port patch
const IstioIngressService = "istio-ingressgateway"

// create func vars for unit tests
type installFuncSig func(log *zap.SugaredLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

var installFunc installFuncSig = istio.Install

type bashFuncSig func(inArgs ...string) (string, string, error)

var bashFunc bashFuncSig = vzos.RunBash

func setInstallFunc(f installFuncSig) {
	installFunc = f
}

func setBashFunc(f bashFuncSig) {
	bashFunc = f
}

func (i IstioComponent) IsOperatorInstallSupported() bool {
	return true
}

// IsInstalled checks if Istio is installed by looking for the Istio control plane deployment
func (i IstioComponent) IsInstalled(compContext spi.ComponentContext) (bool, error) {
	deployment := appsv1.Deployment{}
	nsn := types.NamespacedName{Name: IstiodDeployment, Namespace: IstioNamespace}
	if err := compContext.Client().Get(context.TODO(), nsn, &deployment); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		// Unexpected error
		return false, err
	}
	return true, nil
}

func (i IstioComponent) Install(compContext spi.ComponentContext) error {
	// This IstioOperator YAML uses this imagePullSecret key
	const imagePullSecretHelmKey = "values.global.imagePullSecrets[0]"

	var userFileCR *os.File
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
		userFileCR, err = ioutil.TempFile(os.TempDir(), "istio-*.yaml")
		if err != nil {
			log.Errorf("Failed to create temporary file for Istio install: %v", err)
			return err
		}
		defer os.Remove(userFileCR.Name())
		if _, err = userFileCR.Write([]byte(istioOperatorYaml)); err != nil {
			log.Errorf("Failed to write to temporary file: %v", err)
			return err
		}
		if err := userFileCR.Close(); err != nil {
			log.Errorf("Failed to close temporary file: %v", err)
			return err
		}
		log.Infof("Created values file from Istio install args: %s", userFileCR.Name())
	}

	// check for global image pull secret
	kvs, err = secret.AddGlobalImagePullSecretHelmOverride(log, client, IstioNamespace, kvs, imagePullSecretHelmKey)
	if err != nil {
		return err
	}

	// Build comma separated string of overrides that will be passed to
	// isioctl as --set values.
	// This include BOM image overrides as well as other overrides
	overrideStrings, err := buildOverridesString(log, client, IstioNamespace, kvs...)
	if err != nil {
		return err
	}

	if userFileCR == nil {
		_, _, err := installFunc(log, overrideStrings, i.ValuesFile)
		if err != nil {
			return err
		}
	} else {
		_, _, err := installFunc(log, overrideStrings, i.ValuesFile, userFileCR.Name())
		if err != nil {
			return err
		}
	}

	return nil
}

func (i IstioComponent) PreInstall(compContext spi.ComponentContext) error {
	if err := labelNamespace(compContext); err != nil {
		return err
	}
	if err := createCertSecret(compContext); err != nil {
		return err
	}
	return nil
}

func (i IstioComponent) PostInstall(compContext spi.ComponentContext) error {
	if err := createPeerAuthentication(compContext); err != nil {
		return err
	}
	if err := createEnvoyFilter(compContext); err != nil {
		return err
	}
	if err := updatePortSpec(compContext); err != nil {
		return err
	}
	return nil
}

func createCertSecret(compContext spi.ComponentContext) error {
	log := compContext.Log()
	if compContext.IsDryRun() {
		return nil
	}

	// Create the cert used by Istio MTLS if it doesn't exist
	var secret v1.Secret
	nsn := types.NamespacedName{Namespace: IstioNamespace, Name: IstioCertSecret}
	if err := compContext.Client().Get(context.TODO(), nsn, &secret); err != nil {
		if !errors.IsNotFound(err) {
			// Unexpected error
			return err
		}
		// Secret not found - create it
		certScript := filepath.Join(config.GetInstallDir(), "create-istio-cert.sh")
		if _, stderr, err := bashFunc(certScript); err != nil {
			log.Errorf("Failed creating Istio certificate secret %s: %s", err, stderr)
			return err
		}
	}
	return nil
}

// labelNamespace adds the label needed by network polices
func labelNamespace(compContext spi.ComponentContext) error {
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
	return nil
}

// createPeerAuthentication creates the PeerAuthentication resource to enable STRICT MTLS
func createPeerAuthentication(compContext spi.ComponentContext) error {
	peer := istioclisec.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: IstioNamespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &peer, func() error {
		if peer.Spec.Mtls == nil {
			peer.Spec.Mtls = &istiosec.PeerAuthentication_MutualTLS{}
		}
		peer.Spec.Mtls.Mode = istiosec.PeerAuthentication_MutualTLS_STRICT
		return nil
	})
	return err
}

// createEnvoyFilter creates the Envoy filter used by Istio
func createEnvoyFilter(compContext spi.ComponentContext) error {
	filter := istioclinet.EnvoyFilter{}
	const filterName = IstioEnvoyFilter
	nsn := types.NamespacedName{Namespace: IstioNamespace, Name: filterName}
	if err := compContext.Client().Get(context.TODO(), nsn, &filter); err != nil {
		if !errors.IsNotFound(err) {
			// Unexpected error
			return err
		}
		// Filter not found - create it
		script := filepath.Join(config.GetInstallDir(), "create-envoy-filter.sh")
		if _, stderr, err := bashFunc(script); err != nil {
			compContext.Log().Errorf("Failed creating Envoy filter %s: %s", err, stderr)
			return err
		}
	}
	return nil
}

// updatePortSpec updates the port spec for the Istio ingress gateway
func updatePortSpec(compContext spi.ComponentContext) error {
	istioConfig := compContext.EffectiveCR().Spec.Components.Istio

	// If there are no port updates, then return
	if istioConfig == nil {
		return nil
	}
	if len(istioConfig.Ports) == 0 {
		return nil
	}

	// Patch the service with the port updates if it exists
	ingressService := v1.Service{}
	if err := compContext.Client().Get(context.TODO(), types.NamespacedName{Name: IstioIngressService, Namespace: IstioNamespace}, &ingressService); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	mergeFromSvc := client.MergeFrom(ingressService.DeepCopy())
	ingressService.Spec.Ports = istioConfig.Ports
	if err := compContext.Client().Patch(context.TODO(), &ingressService, mergeFromSvc); err != nil {
		return err
	}
	return nil
}
