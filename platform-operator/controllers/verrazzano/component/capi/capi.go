// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"encoding/base64"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// AuthenticationType for auth
type AuthenticationType string

const ociDefaultSecret = "oci"

const (
	// UserPrincipal is default auth type
	UserPrincipal AuthenticationType = "user_principal"
	// InstancePrincipal is used for instance principle auth type
	InstancePrincipal AuthenticationType = "instance_principal"
)

// OCIAuthConfig holds connection parameters for the OCI API.
type OCIAuthConfig struct {
	Region      string             `yaml:"region"`
	Tenancy     string             `yaml:"tenancy"`
	User        string             `yaml:"user"`
	Key         string             `yaml:"key"`
	Fingerprint string             `yaml:"fingerprint"`
	Passphrase  string             `yaml:"passphrase"`
	AuthType    AuthenticationType `yaml:"authtype"`
}

// OCIConfig holds the configuration for OCI authorization.
type OCIConfig struct {
	Auth OCIAuthConfig `yaml:"auth"`
}

// PreInstall implementation for the CAPI Component
func preInstall(ctx spi.ComponentContext) error {
	// Get OCI credentials from secret in the verrazzano-install namespace
	ociSecret := corev1.Secret{}
	// TODO: use secret name from API when available
	if err := ctx.Client().Get(context.TODO(), client.ObjectKey{Name: ociDefaultSecret, Namespace: constants.VerrazzanoInstallNamespace}, &ociSecret); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to find secret %s in the %s namespace: %v", ociDefaultSecret, constants.VerrazzanoInstallNamespace, err)
	}

	var ociYaml []byte
	for key := range ociSecret.Data {
		if key == "oci.yaml" {
			ociYaml = ociSecret.Data[key]
			break
		}
	}

	if ociYaml == nil {
		return ctx.Log().ErrorfNewErr("Failed to find oci.yaml in secret %s in the %s namespace", ociDefaultSecret, constants.VerrazzanoInstallNamespace)
	}

	cfg := OCIConfig{}
	if err := yaml.Unmarshal(ociYaml, &cfg); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to parse oci.yaml in secret %s in the %s namespace", ociDefaultSecret, constants.VerrazzanoInstallNamespace)
	}

	if cfg.Auth.AuthType == UserPrincipal {
		os.Setenv("OCI_TENANCY_ID_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Tenancy)))
		os.Setenv("OCI_CREDENTIALS_FINGERPRINT_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Fingerprint)))
		os.Setenv("OCI_USER_ID_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.User)))
		os.Setenv("OCI_REGION_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Region)))
		os.Setenv("OCI_CREDENTIALS_KEY_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Key)))
		if len(cfg.Auth.Passphrase) != 0 {
			os.Setenv("OCI_PASSPHRASE_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Passphrase)))
		}
	} else if cfg.Auth.AuthType == InstancePrincipal {
		os.Setenv("USE_INSTANCE_PRINCIPAL_B64", base64.StdEncoding.EncodeToString([]byte("true")))
	} else {
		return ctx.Log().ErrorfNewErr("Invalid authtype value %s found for oci.yaml in secret %s in the %s namespace", cfg.Auth.AuthType, ociDefaultSecret, constants.VerrazzanoInstallNamespace)
	}

	return nil
}
