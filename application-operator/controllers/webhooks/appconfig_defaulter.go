// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	oamv1 "github.com/verrazzano/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	if req.Operation == admissionv1.Delete {
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
	err = a.cleanupCert(appConfig)
	if err != nil {
		return
	}

	err = a.cleanupSecret(appConfig)
	if err != nil {
		return
	}

	// Fixup Istio Authorization policies within a project
	ap := &AuthorizationPolicy{
		Client:      a.Client,
		KubeClient:  a.KubeClient,
		IstioClient: a.IstioClient,
	}
	return ap.cleanupAuthorizationPoliciesForProjects(appConfig.Namespace, appConfig.Name)
}

// cleanupCert cleans up the generated certificate for the given app config
func (a *AppConfigWebhook) cleanupCert(appConfig *oamv1.ApplicationConfiguration) (err error) {
	gatewayCertName := fmt.Sprintf("%s-%s-cert", appConfig.Namespace, appConfig.Name)
	namespacedName := types.NamespacedName{Name: gatewayCertName, Namespace: constants.IstioSystemNamespace}
	var cert *certapiv1alpha2.Certificate
	cert, err = fetchCert(context.TODO(), a.Client, namespacedName)
	if err != nil {
		return err
	}
	if cert != nil {
		err = a.Client.Delete(context.TODO(), cert, &client.DeleteOptions{})
	}
	return
}

// cleanupSecret cleans up the generated secret for the given app config
func (a *AppConfigWebhook) cleanupSecret(appConfig *oamv1.ApplicationConfiguration) (err error) {
	gatewaySecretName := fmt.Sprintf("%s-%s-cert-secret", appConfig.Namespace, appConfig.Name)
	namespacedName := types.NamespacedName{Name: gatewaySecretName, Namespace: constants.IstioSystemNamespace}
	var secret *corev1.Secret
	secret, err = fetchSecret(context.TODO(), a.Client, namespacedName)
	if err != nil {
		return err
	}
	if secret != nil {
		err = a.Client.Delete(context.TODO(), secret, &client.DeleteOptions{})
	}
	return
}

// fetchCert gets the cert for the given name; returns nil Certificate if not found
func fetchCert(ctx context.Context, c client.Reader, name types.NamespacedName) (*certapiv1alpha2.Certificate, error) {
	var cert certapiv1alpha2.Certificate
	err := c.Get(ctx, name, &cert)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			appConfDefLog.Info("cert does not exist", "cert", name)
			return nil, nil
		}
		appConfDefLog.Info("failed to fetch cert", "cert", name)
		return nil, err
	}
	return &cert, err
}

// fetchSecret gets the secret for the given name; returns nil Secret if not found
func fetchSecret(ctx context.Context, c client.Reader, name types.NamespacedName) (*corev1.Secret, error) {
	var secret corev1.Secret
	err := c.Get(ctx, name, &secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			appConfDefLog.Info("secret does not exist", "secret", name)
			return nil, nil
		}
		appConfDefLog.Info("failed to fetch secret", "secret", name)
		return nil, err
	}
	return &secret, err
}
