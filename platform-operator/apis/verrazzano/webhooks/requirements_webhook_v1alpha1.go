// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"github.com/pkg/errors"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/vzchecks"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
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
		case k8sadmission.Create:
			return validateRequirementsV1alpha1(log, v.client, vz)
		case k8sadmission.Update:
			oldVz := v1alpha1.Verrazzano{}
			if err := v.decoder.DecodeRaw(req.OldObject, &oldVz); err != nil {
				return admission.Errored(http.StatusBadRequest, errors.Wrap(err, "unable to decode existing Verrazzano object"))
			}
			return validateUpdate(log, v.client, oldVz, vz)
		}
	}
	return admission.Allowed("")
}

// validateRequirementsV1alpha1 presents the user with a warning if the prerequisite checks are not met.
func validateRequirementsV1alpha1(log *zap.SugaredLogger, client client.Client, vz *v1alpha1.Verrazzano) admission.Response {
	response := admission.Allowed("")
	warnings := getWarningArray(vz)
	if errs := vzchecks.PrerequisiteCheck(client, vzchecks.ProfileType(vz.Spec.Profile)); len(errs) > 0 {
		for _, err := range errs {
			log.Warnf(err.Error())
			warnings = append(warnings, err.Error())
		}
	}
	if len(warnings) > 0 {
		return admission.Allowed("").WithWarnings(warnings...)
	}
	return response
}

func validateUpdate(log *zap.SugaredLogger, client client.Client, oldvz v1alpha1.Verrazzano, newvz *v1alpha1.Verrazzano) admission.Response {
	response := admission.Allowed("")
	newvzv1beta1 := &v1beta1.Verrazzano{}
	oldvzv1beta1 := &v1beta1.Verrazzano{}
	err1 := newvz.ConvertTo(newvzv1beta1)
	err2 := oldvz.ConvertTo(oldvzv1beta1)
	if err1 == nil && err2 == nil {
		return validateUpdatev1beta1(log, client, *oldvzv1beta1, newvzv1beta1)
	}
	return response
}

func getWarningArray(vz *v1alpha1.Verrazzano) []string {
	var warnings []string
	vzv1beta1 := &v1beta1.Verrazzano{}
	if err := vz.ConvertTo(vzv1beta1); err == nil {
		return getWarningArrayv1beta1(vzv1beta1)
	}
	return warnings
}
