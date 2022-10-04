// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"go.uber.org/zap"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

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

// AppConfigDefaulter supplies appconfig default values
type AppConfigDefaulter interface {
	Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool, log *zap.SugaredLogger) error
	Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool, log *zap.SugaredLogger) error
}

// InjectDecoder injects admission.Decoder
func (a *AppConfigWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

var appconfigMarshalFunc = json.Marshal

// Handle handles appconfig mutate Request
func (a *AppConfigWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	counterMetricObject, errorCounterMetricObject, handleDurationMetricObject, zapLogForMetrics, err := metricsexporter.ExposeControllerMetrics("AppConfigDefaulter", metricsexporter.AppconfigHandleCounter, metricsexporter.AppconfigHandleError, metricsexporter.AppconfigHandleDuration)
	if err != nil {
		return admission.Response{}
	}
	handleDurationMetricObject.TimerStart()
	defer handleDurationMetricObject.TimerStop()

	log := zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldWebhook, "appconfig-defaulter")

	dryRun := req.DryRun != nil && *req.DryRun
	appConfig := &oamv1.ApplicationConfiguration{}
	//This json can be used to curl -X POST the webhook endpoint
	log.Debugw("admission.Request", "request", req)
	log.Infow("Handling appconfig default",
		"request.Operation", req.Operation, "request.Name", req.Name)

	// if the operation is Delete then decode the old object and call the defaulter to cleanup any app conf defaults
	if req.Operation == v1.Delete {
		err := a.decoder.DecodeRaw(req.OldObject, appConfig)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		for _, defaulter := range a.Defaulters {
			err = defaulter.Cleanup(appConfig, dryRun, log)
			if err != nil {
				errorCounterMetricObject.Inc(zapLogForMetrics, err)
				return admission.Errored(http.StatusInternalServerError, err)
			}
		}
		if !dryRun {
			err = a.cleanupAppConfig(appConfig, log)
			if err != nil {
				errorCounterMetricObject.Inc(zapLogForMetrics, err)
				log.Errorf("Failed cleaning up app config %s: %v", req.Name, err)
			}
		}
		return admission.Allowed("cleaned up appconfig default")
	}

	err = a.decoder.Decode(req, appConfig)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	//mutate the fields in appConfig
	for _, defaulter := range a.Defaulters {
		err = defaulter.Default(appConfig, dryRun, log)
		if err != nil {
			errorCounterMetricObject.Inc(zapLogForMetrics, err)
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}
	marshaledAppConfig, err := appconfigMarshalFunc(appConfig)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	counterMetricObject.Inc(zapLogForMetrics, err)
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledAppConfig)
}

// cleanupAppConfig cleans up the generated certificates and secrets associated with the given app config
func (a *AppConfigWebhook) cleanupAppConfig(appConfig *oamv1.ApplicationConfiguration, log *zap.SugaredLogger) (err error) {
	// Fixup Istio Authorization policies within a project
	ap := &AuthorizationPolicy{
		Client:      a.Client,
		KubeClient:  a.KubeClient,
		IstioClient: a.IstioClient,
	}
	return ap.cleanupAuthorizationPoliciesForProjects(appConfig.Namespace, appConfig.Name, log)
}
