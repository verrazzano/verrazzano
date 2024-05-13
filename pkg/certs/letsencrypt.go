// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certs

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"io"
	"net/http"
)

const (
	intR10PEM = "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-r10.pem"
	intE5PEM  = "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-e5.pem"
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

	resp, err := common.HTTPDo(c.hc, req)
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
	if err := c.appendCertWithHTTP(intR10PEM); err != nil {
		return err
	}
	if err := c.appendCertWithHTTP(intE5PEM); err != nil {
		return err
	}
	if err := c.appendCertWithHTTP(rootX1PEM); err != nil {
		return err
	}
	return nil
}

// CreateLetsEncryptStagingBundle Builds the Let's Encrypt Staging environment CA cert chain
func CreateLetsEncryptStagingBundle() ([]byte, error) {
	builder := &certBuilder{
		hc: &http.Client{},
	}
	if err := builder.buildLetsEncryptStagingChain(); err != nil {
		return []byte{}, err
	}
	return builder.cert, nil
}
