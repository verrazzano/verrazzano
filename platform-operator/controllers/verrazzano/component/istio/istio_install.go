// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	vzns "github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

func (i IstioComponent) IsOperatorInstallSupported() bool {
	return true
}

func (i IstioComponent) IsInstalled(_ *zap.SugaredLogger, _ clipkg.Client, _ string) (bool, error) {
	return false, nil
}

func (i IstioComponent) Install(log *zap.SugaredLogger, vz *vzapi.Verrazzano, client clipkg.Client, _ string, _ bool) error {
	const imagePullSecretHelmKey = "values.global.imagePullSecrets[0].name"
	var tmpFile *os.File
	var kvs []bom.KeyValue
	var err error

	// Only create override file if the CR has an Istio component
	if vz.Spec.Components.Istio != nil {
		istioOperatorYaml, err := BuildIstioOperatorYaml(vz.Spec.Components.Istio)
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

	// check for global image pull secret
	kvs, err = addGlobalImagePullSecretHelmOverride(log, client, IstioNamespace, kvs, imagePullSecretHelmKey)
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

func (i IstioComponent) PreInstall(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	nsLabelForNetPol := map[string]string{"verrazzano.io/namespace": "istio-system"}

	// Ensure Istio namespace exists and label it for network policies
	if err := vzns.EnsureExists(log, client, IstioNamespace); err != nil {
		log.Errorf("Failed to ensure Istio namespace %s exists: %v", IstioNamespace, err)
		return err
	}

	if err := vzns.AddLabels(log, client, IstioNamespace, nsLabelForNetPol); err != nil {
		log.Errorf("Failed to set NetworkPolicy labels on Istio namespace %s: %v", IstioNamespace, err)
		return err
	}
	if err := vzns.AddLabels(log, client, IstioNamespace, nsLabelForNetPol); err != nil {
		log.Errorf("Failed to set NetworkPolicy labels on Istio namespace %s: %v", IstioNamespace, err)
		return err
	}

	// Create the cert used by Istio MTLS
	certDir := os.TempDir()
	config := certificate.CreateIstioCertConfig(certDir)
	cert, err := certificate.CreateSelfSignedCert(config)
	if err != nil {
		log.Errorf("Failed to create Certificate for Istio: %v", err)
		retur
	}

	return nil
}

func (i IstioComponent) PostInstall(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}

type installFuncSig func(log *zap.SugaredLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

// installFunc is the default install function
var installFunc installFuncSig = istio.Install

func SetIstioInstallFunction(fn installFuncSig) {
	installFunc = fn
}

func ResetIstioInstallFunction() {
	installFunc = istio.Install
}

func setInstallFunc(f installFuncSig) {
	installFunc = f
}

func setDefaultInstallFunc() {
	installFunc = istio.Install
}

// createCert creates certificates and istio secret to hold certificates if it doesn't exist
func createCert(log *zap.SugaredLogger, client clipkg.Client, _ string, namespace string) error {

	return nil
}

// # Create
//if ! kubectl get secret cacerts -n istio-system > /dev/null 2>&1 ; then
//  action "Generating Istio CA bundle" create_secret || exit 1
//fi

// function create_secret {
//  CERTS_OUT=$SCRIPT_DIR/build/istio-certs
//
//  rm -rf $CERTS_OUT || true
//  rm -f ./index.txt* serial serial.old || true
//
//  mkdir -p $CERTS_OUT
//  touch ./index.txt
//  echo 1000 > ./serial
//
//  if ! kubectl get secret cacerts -n istio-system > /dev/null 2>&1; then
//    log "Generating CA bundle for Istio"
//
//    # Create the private key for the root CA
//    openssl genrsa -out $CERTS_OUT/root-key.pem 4096 || return $?
//
//    # Generate a root CA with the private key
//    openssl req -config $CONFIG_DIR/istio_root_ca_config.txt -key $CERTS_OUT/root-key.pem -new -x509 -days 7300 -sha256 -extensions v3_ca -out $CERTS_OUT/root-cert.pem || return $?
//
//    # Create the private key for the intermediate CA
//    openssl genrsa -out $CERTS_OUT/ca-key.pem 4096 || return $?
//
//    # Generate certificate signing request (CSR)
//    openssl req -config $CONFIG_DIR/istio_intermediate_ca_config.txt -new -sha256 -key $CERTS_OUT/ca-key.pem -out $CERTS_OUT/intermediate-csr.pem || return $?
//
//    # create intermediate cert using the root CA
//    openssl ca -batch -config $CONFIG_DIR/istio_root_ca_config.txt -extensions v3_intermediate_ca -days 3650 -notext -md sha256 \
//        -keyfile $CERTS_OUT/root-key.pem \
//        -cert $CERTS_OUT/root-cert.pem \
//        -in $CERTS_OUT/intermediate-csr.pem \
//        -out $CERTS_OUT/ca-cert.pem \
//        -outdir $CERTS_OUT || return $?
//
//    # Create certificate chain file
//    cat $CERTS_OUT/ca-cert.pem $CERTS_OUT/root-cert.pem > $CERTS_OUT/cert-chain.pem || return $?
//
//    kubectl create secret generic cacerts -n istio-system \
//        --from-file=$CERTS_OUT/ca-cert.pem \
//        --from-file=$CERTS_OUT/ca-key.pem  \
//        --from-file=$CERTS_OUT/root-cert.pem \
//        --from-file=$CERTS_OUT/cert-chain.pem || return $?
//  else
//    log "Istio CA Certs bundle and secret already created"
//  fi
//
//  rm -rf $CERTS_OUT
//  rm -f ./index.txt* serial serial.old
//
//  return 0
//}
//
