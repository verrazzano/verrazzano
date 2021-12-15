// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	v1beta12 "k8s.io/api/admission/v1beta1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var appConfDefLog = ctrl.Log.WithName("webhooks.appconfig-defaulter")

// AppConfigDefaulterPath specifies the path of AppConfigDefaulter
const AppConfigDefaulterPath = "/appconfig-defaulter"

// +kubebuilder:webhook:verbs=create;update,path=/appconfig-defaulter,mutating=true,failurePolicy=fail,groups=core.oam.dev,resources=ApplicationConfigurations,versions=v1alpha2,name=appconfig-defaulter.kb.io

// AppConfigWebhook uses a list of AppConfigDefaulters to supply appconfig default values
type AppConfigWebhook struct {
	decoder     *admission.Decoder
	Client      client.Client
	KubeClient  kubernetes.Interface
	IstioClient istioversionedclient.Interface
	Defaulters  []AppConfigDefaulter
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
	appConfDefLog.V(1).Info("admission.Request", "request", req)
	appConfDefLog.Info("Handling appconfig default",
		"request.Operation", req.Operation, "appconfig.Name", req.Name)

	// if the operation is Delete then decode the old object and call the defaulter to cleanup any app conf defaults
	if req.Operation == v1beta12.Delete {
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
		if !dryRun {
			err = a.cleanupAppConfig(appConfig)
			if err != nil {
				appConfDefLog.Error(err, "error cleaning up app config", "appconfig.Name", req.Name)
			}
		}
		return admission.Allowed("cleaned up appconfig default")
	}

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

// cleanupAppConfig cleans up the generated certificates and secrets associated with the given app config
func (a *AppConfigWebhook) cleanupAppConfig(appConfig *oamv1.ApplicationConfiguration) (err error) {
	// Fixup Istio Authorization policies within a project
	ap := &AuthorizationPolicy{
		Client:      a.Client,
		KubeClient:  a.KubeClient,
		IstioClient: a.IstioClient,
	}
	return ap.cleanupAuthorizationPoliciesForProjects(appConfig.Namespace, appConfig.Name)
}
