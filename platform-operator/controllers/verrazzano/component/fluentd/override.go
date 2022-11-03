// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"crypto/sha256"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"io/fs"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	tmpFilePrefix        = "verrazzano-fluentd-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix
)

type fluentdComponentValues struct {
	Logging    *loggingValues `json:"logging,omitempty"`
	Fluentd    *fluentdValues `json:"fluentd,omitempty"`
	Monitoring *Monitoring    `json:"monitoring,omitempty"`
}

type loggingValues struct {
	Name              string `json:"name,omitempty"`
	OpenSearchURL     string `json:"osURL,omitempty"`
	CredentialsSecret string `json:"credentialsSecret,omitempty"`
	ClusterName       string `json:"clusterName"`
	UsernameKey       string `json:"usernameKey"`
	PasswordKey       string `json:"passwordKey"`
	ConfigHash        string `json:"configHash,omitempty"`
}

type fluentdValues struct {
	Enabled           bool                `json:"enabled"` // Always write
	ExtraVolumeMounts []volumeMount       `json:"extraVolumeMounts,omitempty"`
	OCI               *ociLoggingSettings `json:"oci,omitempty"`
}

type volumeMount struct {
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
	ReadOnly    bool   `json:"readOnly,omitempty"`
}

type ociLoggingSettings struct {
	DefaultAppLogID string `json:"defaultAppLogId"`
	SystemLogID     string `json:"systemLogId"`
	APISecret       string `json:"apiSecret,omitempty"`
}

type Monitoring struct {
	Enabled       bool `json:"enabled,omitempty"`
	UseIstioCerts bool `json:"useIstioCerts,omitempty"`
}

// appendOverrides appends the overrides for the component
func appendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	effectiveCR := ctx.EffectiveCR()

	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}
	images, err := bomFile.BuildImageOverrides("verrazzano")
	if err != nil {
		return kvs, err
	}

	fluentdFullImageKey := "logging.fluentdImage"
	var fluentdFullImageValue string
	for _, image := range images {
		if image.Key == fluentdFullImageKey {
			fluentdFullImageValue = image.Value
			break
		}
	}
	if fluentdFullImageValue == "" {
		return kvs, ctx.Log().ErrorfNewErr("Failed to construct fluentd image from BOM")
	}

	kvs = append(kvs, bom.KeyValue{Key: fluentdFullImageKey, Value: fluentdFullImageValue})

	// Overrides object to store any user overrides
	overrides := fluentdComponentValues{}
	// append any fluentd overrides
	appendFluentdOverrides(effectiveCR, &overrides)

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(ctx, &overrides)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed generating Verrazzano overrides file: %v", err)
	}

	kvs = append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})
	return kvs, nil
}

func appendManagedClusterOverrides(client clipkg.Client, overrides *fluentdComponentValues) error {
	registrationSecret, err := common.GetManagedClusterRegistrationSecret(client)
	if err != nil {
		return err
	}
	if registrationSecret == nil {
		return nil
	}
	overrides.Logging.OpenSearchURL = string(registrationSecret.Data[vzconst.OpensearchURLData])
	overrides.Logging.ClusterName = string(registrationSecret.Data[vzconst.ClusterNameData])
	overrides.Logging.CredentialsSecret = vzconst.MCRegistrationSecret
	return nil
}

func appendFluentdOverrides(effectiveCR *vzapi.Verrazzano, overrides *fluentdComponentValues) {
	overrides.Fluentd = &fluentdValues{
		Enabled: vzconfig.IsFluentdEnabled(effectiveCR),
	}
	fluentd := effectiveCR.Spec.Components.Fluentd
	if fluentd != nil {
		overrides.Logging = &loggingValues{ConfigHash: HashSum(fluentd)}
		if len(fluentd.ElasticsearchURL) > 0 {
			overrides.Logging.OpenSearchURL = fluentd.ElasticsearchURL
		}
		if len(fluentd.ElasticsearchSecret) > 0 {
			overrides.Logging.CredentialsSecret = fluentd.ElasticsearchSecret
		}
		if len(fluentd.ExtraVolumeMounts) > 0 {
			for _, vm := range fluentd.ExtraVolumeMounts {
				dest := vm.Source
				if vm.Destination != "" {
					dest = vm.Destination
				}
				readOnly := true
				if vm.ReadOnly != nil {
					readOnly = *vm.ReadOnly
				}
				overrides.Fluentd.ExtraVolumeMounts = append(overrides.Fluentd.ExtraVolumeMounts,
					volumeMount{Source: vm.Source, Destination: dest, ReadOnly: readOnly})
			}
		}
		// Overrides for OCI Logging integration
		if fluentd.OCI != nil {
			overrides.Fluentd.OCI = &ociLoggingSettings{
				DefaultAppLogID: fluentd.OCI.DefaultAppLogID,
				SystemLogID:     fluentd.OCI.SystemLogID,
				APISecret:       fluentd.OCI.APISecret,
			}
		}
	}

	// Force the override to be the internal ES secret if the legacy ES secret is being used.
	// This may be the case during an upgrade from a version that was not using the ES internal password for Fluentd.
	if overrides.Logging != nil {
		if overrides.Logging.OpenSearchURL == globalconst.LegacyElasticsearchSecretName {
			overrides.Logging.CredentialsSecret = globalconst.VerrazzanoESInternal
		}
	}

	overrides.Monitoring = &Monitoring{
		Enabled:       vzconfig.IsPrometheusOperatorEnabled(effectiveCR),
		UseIstioCerts: vzconfig.IsIstioEnabled(effectiveCR),
	}
}

func generateOverridesFile(ctx spi.ComponentContext, overrides *fluentdComponentValues) (string, error) {
	bytes, err := yaml.Marshal(overrides)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp(os.TempDir(), tmpFileCreatePattern)
	if err != nil {
		return "", err
	}

	overridesFileName := file.Name()
	if err := writeFileFunc(overridesFileName, bytes, fs.ModeAppend); err != nil {
		return "", err
	}
	ctx.Log().Debugf("Verrazzano install overrides file %s contents: %s", overridesFileName, string(bytes))
	return overridesFileName, nil
}

// cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Failed deleting temp files: %v", err)
	}
}

// HashSum returns the hash sum of the config object
func HashSum(config interface{}) string {
	sha := sha256.New()
	if data, err := yaml.Marshal(config); err == nil {
		sha.Write(data)
		return fmt.Sprintf("%x", sha.Sum(nil))
	}
	return ""
}
