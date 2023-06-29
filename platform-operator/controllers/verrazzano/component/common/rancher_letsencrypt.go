// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"io"
	"net/http"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	intR3PEM  = "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-r3.pem"
	intE1PEM  = "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-e1.pem"
	rootX1PEM = "https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem"
)

type certBuilder struct {
	cert []byte
	hc   *http.Client
}

func (c *certBuilder) appendCertWithHTTP(uri string) error {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	resp, err := HTTPDo(c.hc, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed downloading cert from %s: %s", uri, resp.Status)
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.cert = append(c.cert, bytes...)
	return nil
}

// buildLetsEncryptStagingChain builds the LetsEncrypt Staging certificate chain
// LetsEncrypt staging provides a certificate chain for staging environments, mimicking production.
// Verrazzano uses the LetsEncrypt staging certificate chain for Rancher ingress on ACME staging environments.
// See https://letsencrypt.org/docs/staging-environment/ for more information.
func (c *certBuilder) buildLetsEncryptStagingChain() error {
	if err := c.appendCertWithHTTP(intR3PEM); err != nil {
		return err
	}
	if err := c.appendCertWithHTTP(intE1PEM); err != nil {
		return err
	}
	if err := c.appendCertWithHTTP(rootX1PEM); err != nil {
		return err
	}
	return nil
}

func useAdditionalCAs(acme vzapi.Acme) bool {
	return acme != vzapi.Acme{} && acme.Environment != "production"
}

func ProcessAdditionalCertificates(log vzlog.VerrazzanoLogger, cli client.Client, vz *vzapi.Verrazzano) error {
	// Skip updating Rancher certificates if Rancher is disabled.
	if !vzcr.IsRancherEnabled(vz) {
		return nil
	}
	cm := vz.Spec.Components.CertManager
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: CattleSystem,
			Name:      constants.AdditionalTLS,
		},
	}
	if cm != nil && useAdditionalCAs(cm.Certificate.Acme) {
		log.Debugf("Creating additional Rancher certificates for non-production environment")
		return createAdditionalCertificates(log, cli, secret)
	}
	if err := cli.Delete(context.TODO(), secret); err != nil {
		log.Debugf("Delete secret %v error: %v", secret, err)
	}
	return nil
}

func createAdditionalCertificates(log vzlog.VerrazzanoLogger, cli client.Client, secret *v1.Secret) error {
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CattleSystem}}
	key := client.ObjectKeyFromObject(&ns)
	err := cli.Get(context.TODO(), key, &ns)
	if err != nil && apierrors.IsNotFound(err) {
		_, err = controllerutil.CreateOrUpdate(context.TODO(), cli, &ns, func() error {
			return nil
		})
		if err != nil {
			log.Debugf("Failed to create namespace: %v", err)
			return err
		}
	}
	_, err = controllerruntime.CreateOrUpdate(context.TODO(), cli, secret, func() error {
		builder := &certBuilder{
			hc: &http.Client{},
		}
		if err := builder.buildLetsEncryptStagingChain(); err != nil {
			return err
		}
		secret.Data = map[string][]byte{
			constants.AdditionalTLSCAKey: builder.cert,
		}
		return nil
	})
	if err != nil {
		log.Debugf("Create secret %v error: %v", secret, err)
		return err
	}
	return nil
}
