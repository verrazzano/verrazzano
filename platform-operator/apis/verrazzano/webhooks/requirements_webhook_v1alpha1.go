// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/vzchecks"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	k8sadmission "k8s.io/api/admission/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// RequirementsValidatorV1alpha1 is a struct holding objects used during validation.
type RequirementsValidatorV1alpha1 struct {
	client  client.Client
	decoder *admission.Decoder
}

// InjectClient injects the client.
func (v *RequirementsValidatorV1alpha1) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *RequirementsValidatorV1alpha1) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of the Verrazzano prerequisites based on the profiled used.
func (v *RequirementsValidatorV1alpha1) Handle(ctx context.Context, req admission.Request) admission.Response {
	var log = zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, RequirementsWebhook)

	log.Infof("Processing requirements validator webhook")
	vz := &v1alpha1.Verrazzano{}
	err := v.decoder.Decode(req, vz)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if vz.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Create, k8sadmission.Update:
			return validateRequirementsV1alpha1(log, v.client, vz)
		}
	}
	return admission.Allowed("")
}

// validateRequirementsV1alpha1 presents the user with a warning if the prerequisite checks are not met.
func validateRequirementsV1alpha1(log *zap.SugaredLogger, client client.Client, vz *v1alpha1.Verrazzano) admission.Response {
	response := admission.Allowed("")
	if errs := vzchecks.PrerequisiteCheck(client, vzchecks.ProfileType(vz.Spec.Profile)); len(errs) > 0 {
		var warnings []string
		for _, err := range errs {
			log.Warnf(err.Error())
			warnings = append(warnings, err.Error())
		}
		return admission.Allowed("").WithWarnings(warnings...)
	}
	return response
}
