// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"github.com/pkg/errors"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	k8sadmission "k8s.io/api/admission/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// MysqlValuesValidatorV1alpha1 is a struct holding objects used during validation.
type MysqlValuesValidatorV1alpha1 struct {
	decoder    *admission.Decoder
	BomVersion string
}

// InjectDecoder injects the decoder.
func (v *MysqlValuesValidatorV1alpha1) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated Verrazzano resources.
func (v *MysqlValuesValidatorV1alpha1) Handle(ctx context.Context, req admission.Request) admission.Response {
	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "verrazzano-platform-mysqlinstalloverrides")
	vz := &v1alpha1.Verrazzano{}
	err := v.decoder.Decode(req, vz)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if vz.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Update:
			oldVz := v1alpha1.Verrazzano{}
			if err := v.decoder.DecodeRaw(req.OldObject, &oldVz); err != nil {
				return admission.Errored(http.StatusBadRequest, errors.Wrap(err, "unable to decode existing Verrazzano object"))
			}
			return v.validateMysqlValuesV1alpha1(log, oldVz, vz)
		}
	}
	return admission.Allowed("")
}

// validateMysqlValuesV1alpha1 presents the user with a warning if there are podSpecs specified as overrides.
func (v *MysqlValuesValidatorV1alpha1) validateMysqlValuesV1alpha1(log *zap.SugaredLogger, oldVz v1alpha1.Verrazzano, newVz *v1alpha1.Verrazzano) admission.Response {
	response := admission.Allowed("")
	versionToCompare := getVersion(oldVz.Status.Version, newVz.Spec.Version, v.BomVersion)
	if isMinVersion(versionToCompare, MinVersion) {
		log.Debug("Validating v1alpha1 MySQL values")
		if newVz.Spec.Components.Keycloak != nil {
			// compare overrides from current and previous VZ
			newMySQLOverrides := newVz.Spec.Components.Keycloak.MySQL.ValueOverrides
			for _, override := range newMySQLOverrides {
				var err error
				warning, err := inspectOverride(override.Values)
				if err != nil {
					return admission.Errored(http.StatusBadRequest, err)
				}
				if len(warning) > 0 {
					log.Warnf(warning)
					response = admission.Allowed("").WithWarnings(warning)
				}
			}
		}
	}

	return response
}
