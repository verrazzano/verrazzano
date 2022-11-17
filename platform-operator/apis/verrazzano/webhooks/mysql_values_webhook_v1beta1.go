// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"github.com/pkg/errors"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"go.uber.org/zap"
	k8sadmission "k8s.io/api/admission/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"
)

const (
	MinVersion = "1.5.0"
)

// MysqlValuesValidatorV1beta1 is a struct holding objects used during validation.
type MysqlValuesValidatorV1beta1 struct {
	decoder    *admission.Decoder
	BomVersion string
}

// InjectDecoder injects the decoder.
func (v *MysqlValuesValidatorV1beta1) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated Verrazzano resources.
func (v *MysqlValuesValidatorV1beta1) Handle(ctx context.Context, req admission.Request) admission.Response {
	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "verrazzano-platform-mysqlinstalloverrides")

	vz := &v1beta1.Verrazzano{}
	err := v.decoder.Decode(req, vz)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if vz.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Update:
			oldVz := v1beta1.Verrazzano{}
			if err := v.decoder.DecodeRaw(req.OldObject, &oldVz); err != nil {
				return admission.Errored(http.StatusBadRequest, errors.Wrap(err, "unable to decode existing Verrazzano object"))
			}
			return v.validateMysqlValuesV1beta1(log, oldVz, vz)
		}
	}
	log.Debugf("Admission allowed")
	return admission.Allowed("")
}

// validateMysqlValuesV1alpha1 presents the user with a warning if there are podSpecs specified as overrides.
func (v *MysqlValuesValidatorV1beta1) validateMysqlValuesV1beta1(log *zap.SugaredLogger, oldVz v1beta1.Verrazzano, newVz *v1beta1.Verrazzano) admission.Response {
	response := admission.Allowed("")
	versionToCompare := getVersion(oldVz.Status.Version, newVz.Spec.Version, v.BomVersion)
	log.Debugf("Min version required %s, version to compare: %s", MinVersion, versionToCompare)
	if isMinVersion(versionToCompare, MinVersion) {
		if newVz.Spec.Components.Keycloak != nil {
			newMySQLOverrides := newVz.Spec.Components.Keycloak.MySQL.ValueOverrides
			for _, override := range newMySQLOverrides {
				var err error
				hasWarning, warning, err := inspectOverride(override.Values)
				if err != nil {
					return admission.Errored(http.StatusBadRequest, err)
				}
				if hasWarning {
					response = admission.Allowed("").WithWarnings(warning)
				}
			}
		}
	}
	return response
}

func inspectOverride(override *apiextensionsv1.JSON) (bool, string, error) {
	overrideBytes, err := yaml.Marshal(override)
	if err != nil {
		return false, "", err
	}
	serverPodSpecValue, err := extractValueFromOverrideString(string(overrideBytes), "podSpec")
	if err != nil {
		return false, "", err
	}
	if serverPodSpecValue != nil {
		return true, "Modifications to MySQL server pod specs do not trigger an automatic restart of the stateful set. " +
			"Please refer to the documentation for a rolling restart: https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#rollout", nil
	}
	return false, "", nil
}

func getVersion(statusVersion string, newSpecVersion string, bomVersion string) string {
	if len(newSpecVersion) == 0 {
		// Version field in new spec is not set, use the version in the status field
		if len(statusVersion) == 0 {
			return bomVersion
		}
		// Use the version of the BOM in the image; likely there's an install in progress here
		return statusVersion
	}
	return newSpecVersion
}
