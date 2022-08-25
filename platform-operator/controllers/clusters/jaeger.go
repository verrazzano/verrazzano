// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"path/filepath"
)

const (
	jaegerNamespace        = constants.VerrazzanoMonitoringNamespace
	jaegerCreateField      = "jaeger.create"
	jaegerSecNameField     = "jaeger.spec.storage.secretName"
	jaegerStorageTypeField = "jaeger.spec.storage.type"
	jaegerOSURLField       = "jaeger.spec.storage.options.es.server-urls"
	jaegerOSCAField        = "jaeger.spec.storage.options.es.tls.ca"
	jaegerOSTLSKeyField    = "jaeger.spec.storage.options.es.tls.key"
	jaegerOSTLSCertField   = "jaeger.spec.storage.options.es.tls.cert"
)

type jaegerSpecConfig struct {
	jaegerCreate    bool
	secName         string
	storageType     string
	OSURL           string
	CAFileName      string
	TLSKeyFileName  string
	TLSCertFileName string
}

type jaegerOpenSearchConfig struct {
	URL      string
	username []byte
	password []byte
	CA       []byte
	TLSKey   []byte
	TLSCert  []byte
}

// getJaegerOpenSearchConfig gets Jaeger OpenSearch Storage data if it exists
func (r *VerrazzanoManagedClusterReconciler) getJaegerOpenSearchConfig(vzList *vzapi.VerrazzanoList) (*jaegerOpenSearchConfig, error) {
	jc := &jaegerOpenSearchConfig{}
	// Get the OpenSearch URL, secret and other details configured in Jaeger spec
	jsc, err := r.getJaegerSpecConfig(vzList)
	if err != nil {
		return jc, r.log.ErrorfNewErr("Failed to fetch the Jaeger spec details from the Verrazzano CR %v", err)
	}
	// If Jaeger instance creation is disabled in the VZ CR, then just return
	if !jsc.jaegerCreate {
		r.log.Once("Jaeger instance creation is disabled in the Verrazzano CR. Skipping multicluster Jaeger" +
			" configuration.")
		return jc, nil
	}
	// If OpenSearch storage is not configured for the Jaeger instance, then just return
	if jsc.storageType != "elasticsearch" {
		r.log.Once("A Jaeger instance with OpenSearch storage is not configured. Skipping multicluster Jaeger" +
			" configuration.")
		return jc, nil
	}

	jaegerSecret, err := r.getSecret(jaegerNamespace, jsc.secName, true)
	if err != nil {
		return jc, err
	}

	// Decide which OpenSearch URL to use.
	// If the Jaeger OpenSearch URL is the default URL, use VMI OpenSearch ingress URL.
	// If the Jaeger OpenSearch URL  is not the default, meaning it is a custom OpenSearch, use the external OpenSearch URL.
	if jsc.OSURL == vzconstants.DefaultJaegerOSURL {
		jc.URL, err = r.getVmiESURL()
		if err != nil {
			return jc, err
		}
		// Get the CA bundle needed to connect to the admin keycloak
		jc.CA, err = r.getAdminCaBundle()
		if err != nil {
			return jc, r.log.ErrorfNewErr("Failed to get the CA bundle used by Verrazzano ingress %v", err)
		}
	} else {
		jc.URL = jsc.OSURL
		jc.CA = jaegerSecret.Data[jsc.CAFileName]
		jc.TLSKey = jaegerSecret.Data[jsc.TLSKeyFileName]
		jc.TLSCert = jaegerSecret.Data[jsc.TLSCertFileName]
	}
	jc.username = jaegerSecret.Data[mcconstants.JaegerOSUsernameKey]
	jc.password = jaegerSecret.Data[mcconstants.JaegerOSPasswordKey]
	return jc, nil
}

// getJaegerSpecConfig returns the OpenSearch URL, secret and other details configured for Jaeger instance
func (r *VerrazzanoManagedClusterReconciler) getJaegerSpecConfig(vzList *vzapi.VerrazzanoList) (jaegerSpecConfig, error) {
	jsc := jaegerSpecConfig{}
	for _, vz := range vzList.Items {
		if !isJaegerOperatorEnabled(vz) {
			continue
		}
		// Set values for default Jaeger instance
		if canUseVZOpenSearchStorage(vz) {
			jsc.jaegerCreate = true
			jsc.OSURL = vzconstants.DefaultJaegerOSURL
			jsc.secName = vzconstants.DefaultJaegerSecretName
			jsc.storageType = "elasticsearch"
		}
		overrides := vz.Spec.Components.JaegerOperator.ValueOverrides
		overrideYAMLs, err := common.GetInstallOverridesYAMLUsingClient(r.Client, overrides, jaegerNamespace)
		if err != nil {
			return jsc, err
		}
		for _, overrideYAML := range overrideYAMLs {
			value, err := common.ExtractValueFromOverrideString(overrideYAML, jaegerCreateField)
			if err != nil {
				return jsc, err
			}
			if value != nil {
				jsc.jaegerCreate = value.(bool)
			}
			// Check if there are any Helm chart override values defined for Jaeger storage
			value, err = common.ExtractValueFromOverrideString(overrideYAML, jaegerStorageTypeField)
			if err != nil {
				return jsc, err
			}
			if value != nil {
				jsc.storageType = value.(string)
			}
			value, err = common.ExtractValueFromOverrideString(overrideYAML, jaegerOSURLField)
			if err != nil {
				return jsc, err
			}
			if value != nil {
				jsc.OSURL = value.(string)
			}
			value, err = common.ExtractValueFromOverrideString(overrideYAML, jaegerOSCAField)
			if err != nil {
				return jsc, err
			}
			if value != nil {
				jsc.CAFileName = filepath.Base(value.(string))
			}
			value, err = common.ExtractValueFromOverrideString(overrideYAML, jaegerOSTLSKeyField)
			if err != nil {
				return jsc, err
			}
			if value != nil {
				jsc.TLSKeyFileName = filepath.Base(value.(string))
			}
			value, err = common.ExtractValueFromOverrideString(overrideYAML, jaegerOSTLSCertField)
			if err != nil {
				return jsc, err
			}
			if value != nil {
				jsc.TLSCertFileName = filepath.Base(value.(string))
			}
			value, err = common.ExtractValueFromOverrideString(overrideYAML, jaegerSecNameField)
			if err != nil {
				return jsc, err
			}
			if value != nil {
				jsc.secName = value.(string)
			}
		}
	}
	return jsc, nil
}

// canUseVZOpenSearchStorage determines if Verrazzano's OpenSearch can be used as a storage for Jaeger instance.
// As default Jaeger uses Authproxy to connect to OpenSearch storage, check if Keycloak component is also enabled.
func canUseVZOpenSearchStorage(vz vzapi.Verrazzano) bool {
	if vzconfig.IsOpenSearchEnabled(&vz) && vzconfig.IsKeycloakEnabled(&vz) {
		return true
	}
	return false
}

func isJaegerOperatorEnabled(vz vzapi.Verrazzano) bool {
	return vzconfig.IsJaegerOperatorEnabled(&vz)
}
