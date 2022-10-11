// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"context"
	"github.com/Jeffail/gabs/v2"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"go.uber.org/zap"
	k8sadmission "k8s.io/api/admission/v1"
	"net/http"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"
)

const (
	MIN_VERSION = "1.5.0"
	WebhookName = "v1beta1-verrazzano-platform-mysqlinstalloverrides"
)

// MysqlValuesValidator is a struct holding objects used during validation.
type MysqlValuesValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// InjectClient injects the client.
func (v *MysqlValuesValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *MysqlValuesValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated Verrazzano resources.
func (v *MysqlValuesValidator) Handle(ctx context.Context, req admission.Request) admission.Response {

	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, WebhookName)

	log.Infof("Processing MySQL install override values")
	vz := &Verrazzano{}
	err := v.decoder.Decode(req, vz)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	log.Infof("new VZ: %v", vz)

	oldVz := &Verrazzano{}
	err = v.decoder.DecodeRaw(req.OldObject, oldVz)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	log.Infof("old VZ: %v", vz)

	log.Infof ("Request operation: %v", req.Operation)
	if vz.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Update:
			return validateMysqlValues(vz, oldVz, log)
		}
	}
	return admission.Allowed("")
}

// validateMysqlValues presents the user with a warning if there are podSpecs specified as overrides.
func validateMysqlValues(vz *Verrazzano, oldVz *Verrazzano, log *zap.SugaredLogger) admission.Response {
	response := admission.Allowed(MIN_VERSION)
	log.Infof("old version: %s", oldVz.Spec.Version)
	log.Infof("new version: %s", vz.Spec.Version)
	if isMinVersion(vz.Spec.Version, MIN_VERSION) && isSameVersion(vz.Spec.Version, oldVz.Spec.Version) {
		log.Info("Proceeding with override validation")
		if vz.Spec.Components.Keycloak != nil {
			// compare overrides from current and previous VZ
			overrides, err := yaml.Marshal(vz.Spec.Components.Keycloak.MySQL.ValueOverrides)
			if err != nil {
				return admission.Errored(http.StatusBadRequest, err)
			}
			currentOverrides := string(overrides)
			log.Infof("Current Overrides: %s", currentOverrides)
			overrides, err = yaml.Marshal(oldVz.Spec.Components.Keycloak.MySQL.ValueOverrides)
			if err != nil {
				return admission.Errored(http.StatusBadRequest, err)
			}
			previousOverrides := string(overrides)
			log.Infof("Previous Overrides: %s", previousOverrides)

			response := compareOverride(previousOverrides, currentOverrides, "0.values.podSpec")
			if !response.Allowed || len(response.Warnings) > 0 {
				return response
			}
			response = compareOverride(previousOverrides, currentOverrides, "0.values.router.podSpec")
			if !response.Allowed || len(response.Warnings) > 0 {
				return response
			}
		}
	}

	return response
}

// compareOverride compares the override values to decide whether a change occurred
func compareOverride(previousOverrides string, currentOverrides string, key string) admission.Response {
	previousValue, err := extractValueFromOverrideString(previousOverrides, key)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	currentValue, err := extractValueFromOverrideString(currentOverrides, key)
	if currentValue != nil && previousValue != nil && !reflect.DeepEqual(currentValue, previousValue) {
		return admission.Allowed("").WithWarnings("Modifications to pod specs do not trigger an automatic restart of a stateful set. " +
			"Please refer to the documentation for a rolling restart: https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#rollout")
	}

	return admission.Allowed("")
}

// isMinVersion indicates whether the provide version is greater than the min version provided
func isMinVersion(vzVersion, minVersion string) bool {
	vzSemver, err := semver.NewSemVersion(vzVersion)
	if err != nil {
		return false
	}
	minSemver, err := semver.NewSemVersion(minVersion)
	if err != nil {
		return false
	}
	return !vzSemver.IsLessThan(minSemver)
}

// isSameVersion indicates whether the two version are the same
func isSameVersion(vzVersion, oldVzVersion string) bool {
	vzSemver, err := semver.NewSemVersion(vzVersion)
	if err != nil {
		return false
	}
	oldVzSemver, err := semver.NewSemVersion(oldVzVersion)
	if err != nil {
		return false
	}
	return vzSemver.IsEqualTo(oldVzSemver)
}

// extractValueFromOverrideString extracts  a given value from override.
func extractValueFromOverrideString(overrideStr string, field string) (interface{}, error) {
	jsonConfig, err := yaml.YAMLToJSON([]byte(overrideStr))
	if err != nil {
		return nil, err
	}
	jsonString, err := gabs.ParseJSON(jsonConfig)
	if err != nil {
		return nil, err
	}
	return jsonString.Path(field).Data(), nil
}
