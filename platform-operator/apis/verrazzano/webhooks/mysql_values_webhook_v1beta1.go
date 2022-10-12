// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"go.uber.org/zap"
	k8sadmission "k8s.io/api/admission/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"
)

const (
	MIN_VERSION = "1.5.0"
)

// MysqlValuesValidatorV1beta1 is a struct holding objects used during validation.
type MysqlValuesValidatorV1beta1 struct {
	decoder *admission.Decoder
}

// InjectDecoder injects the decoder.
func (v *MysqlValuesValidatorV1beta1) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated Verrazzano resources.
func (v *MysqlValuesValidatorV1beta1) Handle(ctx context.Context, req admission.Request) admission.Response {

	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "verrazzano-platform-mysqlinstalloverrides")

	log.Infof("Processing MySQL install override values")
	vz := &v1beta1.Verrazzano{}
	err := v.decoder.Decode(req, vz)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if vz.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Update:
			return validateMysqlValuesV1beta1(vz)
		}
	}
	return admission.Allowed("")
}

// validateMysqlValuesV1alpha1 presents the user with a warning if there are podSpecs specified as overrides.
func validateMysqlValuesV1beta1(vz *v1beta1.Verrazzano) admission.Response {
	response := admission.Allowed("")
	if isMinVersion(vz.Spec.Version, MIN_VERSION) {
		if vz.Spec.Components.Keycloak != nil {
			// compare overrides from current and previous VZ
			var overrides []byte
			var err error
			if len(vz.Spec.Components.Keycloak.MySQL.ValueOverrides) > 0 {
				overrides, err = yaml.Marshal(vz.Spec.Components.Keycloak.MySQL.ValueOverrides)
			}
			if err != nil {
				return admission.Errored(http.StatusBadRequest, err)
			}
			currentOverrides := string(overrides)
			serverPodSpecValue, err := extractValueFromOverrideString(currentOverrides, "0.values.podSpec")
			if err != nil {
				return admission.Errored(http.StatusBadRequest, err)
			}

			if serverPodSpecValue != nil {
				response = admission.Allowed("").WithWarnings("Modifications to MySQL server pod specs do not trigger an automatic restart of the stateful set. " +
					"Please refer to the documentation for a rolling restart: https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#rollout")
			}
		}
	}

	return response
}
