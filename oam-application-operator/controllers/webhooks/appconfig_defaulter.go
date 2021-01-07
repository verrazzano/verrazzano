// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	v1beta12 "k8s.io/api/admission/v1beta1"
	"net/http"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var log = ctrl.Log.WithName("webhooks.appconfig-defaulter")

// AppConfigDefaulterPath specifies the path of AppConfigDefaulter
const AppConfigDefaulterPath = "/appconfig-defaulter"

// +kubebuilder:webhook:verbs=create;update,path=/appconfig-defaulter,mutating=true,failurePolicy=fail,groups=core.oam.dev,resources=ApplicationConfigurations,versions=v1alpha2,name=appconfig-defaulter.kb.io

// AppConfigWebhook uses a list of AppConfigDefaulters to supply appconfig default values
type AppConfigWebhook struct {
	decoder    *admission.Decoder
	Defaulters []AppConfigDefaulter
}

//AppConfigDefaulter supplies appconfig default values
type AppConfigDefaulter interface {
	Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error
	Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error
}

// InjectDecoder injects admission.Decoder
func (a *AppConfigWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

var appconfigMarshalFunc = json.Marshal

// Handle handles appconfig mutate Request
func (a *AppConfigWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	dryRun := req.DryRun != nil && *req.DryRun
	appConfig := &oamv1.ApplicationConfiguration{}
	//This json can be used to curl -X POST the webhook endpoint
	log.V(1).Info("admission.Request", "request", req)

	// if the operation is Delete then decode the old object and call the defaulter to cleanup any app conf defaults
	if req.Operation == v1beta12.Delete {
		log.Info("cleaning up appconfig default",
			"appconfig.Name", appConfig.Name, "appconfig.Kind", appConfig.Kind)
		err := a.decoder.DecodeRaw(req.OldObject, appConfig)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		for _, defaulter := range a.Defaulters {
			err = defaulter.Cleanup(appConfig, dryRun)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
		return admission.Allowed("cleaned up appconfig default")
	}

	log.Info("adding appconfig default",
		"appconfig.Name", appConfig.Name, "appconfig.Kind", appConfig.Kind)
	err := a.decoder.Decode(req, appConfig)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	//mutate the fields in appConfig
	for _, defaulter := range a.Defaulters {
		err = defaulter.Default(appConfig, dryRun)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}
	marshaledAppConfig, err := appconfigMarshalFunc(appConfig)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledAppConfig)
}
