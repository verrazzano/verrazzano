// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
)

func decoder() *admission.Decoder {
	scheme := runtime.NewScheme()
	core.AddToScheme(scheme)
	decoder, err := admission.NewDecoder(scheme)
	if err != nil {
		log.Error(err, "error new decoder")
	}
	return decoder
}

// TestAppConfigDefaulterHandleError tests handling an invalid appconfig admission.Request
// GIVEN a AppConfigDefaulter and an appconfig admission.Request
//  WHEN Handle is called with an invalid admission.Request containing no content
//  THEN Handle should return an error with http.StatusBadRequest
func TestAppConfigDefaulterHandleError(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusBadRequest), res.Result.Code)
}

// TestAppConfigDefaulterHandle tests handling an appconfig admission.Request
// GIVEN a AppConfigDefaulter and an appconfig admission.Request
//  WHEN Handle is called with an admission.Request containing appconfig
//  THEN Handle should return a patch response
func TestAppConfigDefaulterHandle(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.NotEqual(t, 0, len(res.Patches))
}

// TestAppConfigWebhookHandleDelete tests handling an appconfig Delete admission.Request
// GIVEN a AppConfigWebhook and an appconfig Delete admission.Request
//  WHEN Handle is called with an admission.Request containing appconfig
//  THEN Handle should return an Allowed response with no patch
func TestAppConfigWebhookHandleDelete(t *testing.T) {
	testAppConfigWebhookHandleDelete(t, true, true, false)
}

// TestAppConfigWebhookHandleDeleteDryRun tests handling a dry run appconfig Delete admission.Request
// GIVEN a AppConfigWebhook and an appconfig Delete admission.Request
//  WHEN Handle is called with an admission.Request containing appconfig and set for a dry run
//  THEN Handle should return an Allowed response with no patch
func TestAppConfigWebhookHandleDeleteDryRun(t *testing.T) {
	testAppConfigWebhookHandleDelete(t, true, true, true)
}

// TestAppConfigWebhookHandleDeleteCertNotFound tests handling an appconfig Delete admission.Request where the app config cert is not found
// GIVEN a AppConfigWebhook and an appconfig Delete admission.Request
//  WHEN Handle is called with an admission.Request containing appconfig and the cert is not found
//  THEN Handle should return an Allowed response with no patch
func TestAppConfigWebhookHandleDeleteCertNotFound(t *testing.T) {
	testAppConfigWebhookHandleDelete(t, false, true, false)
}

// TestAppConfigWebhookHandleDeleteSecretNotFound tests handling an appconfig Delete admission.Request where the app config secret is not found
// GIVEN a AppConfigWebhook and an appconfig Delete admission.Request
//  WHEN Handle is called with an admission.Request containing appconfig and the secret is not found
//  THEN Handle should return an Allowed response with no patch
func TestAppConfigWebhookHandleDeleteSecretNotFound(t *testing.T) {
	testAppConfigWebhookHandleDelete(t, true, false, false)
}

// TestAppConfigDefaulterHandleMarshalError tests handling an appconfig with json marshal error
// GIVEN a AppConfigDefaulter with mock appconfigMarshalFunc
//  WHEN Handle is called with an admission.Request containing appconfig
//  THEN Handle should return error with http.StatusInternalServerError
func TestAppConfigDefaulterHandleMarshalError(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	appconfigMarshalFunc = func(v interface{}) ([]byte, error) {
		return nil, fmt.Errorf("json marshal error")
	}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusInternalServerError), res.Result.Code)
}

type mockErrorDefaulter struct {
}

func (*mockErrorDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	return fmt.Errorf("mockErrorDefaulter error")
}

func (*mockErrorDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	return nil
}

// TestAppConfigDefaulterHandleDefaultError tests handling a defaulter error
// GIVEN a AppConfigDefaulter with mock defaulter
//  WHEN Handle is called with an admission.Request containing appconfig
//  THEN Handle should return error with http.StatusInternalServerError
func TestAppConfigDefaulterHandleDefaultError(t *testing.T) {
	decoder := decoder()
	defaulter := &AppConfigWebhook{Defaulters: []AppConfigDefaulter{&mockErrorDefaulter{}}}
	defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	res := defaulter.Handle(context.TODO(), req)
	assert.False(t, res.Allowed)
	assert.Equal(t, int32(http.StatusInternalServerError), res.Result.Code)
}

func testAppConfigWebhookHandleDelete(t *testing.T, certFound, secretFound, dryRun bool) {
	const istioSystem = "istio-system"
	const certName = "default-hello-app-cert"
	const secretName = "default-hello-app-cert-secret"

	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	if !dryRun {
		// get cert
		mockClient.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: istioSystem, Name: certName}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, cert *certapiv1alpha2.Certificate) error {
				if certFound {
					cert.Namespace = istioSystem
					cert.Name = certName
					return nil
				}
				return k8serrors.NewNotFound(k8sschema.GroupResource{}, certName)
			})

		// delete cert
		if certFound {
			mockClient.EXPECT().
				Delete(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
				DoAndReturn(func(ctx context.Context, cert *certapiv1alpha2.Certificate, opt *client.DeleteOptions) error {
					assert.Equal(t, istioSystem, cert.Namespace)
					assert.Equal(t, certName, cert.Name)
					return nil
				})
		}

		// get secret
		mockClient.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: istioSystem, Name: secretName}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, sec *corev1.Secret) error {
				if secretFound {
					sec.Namespace = istioSystem
					sec.Name = secretName
					return nil
				}
				return k8serrors.NewNotFound(k8sschema.GroupResource{}, secretName)
			})

		// delete secret
		if secretFound {
			mockClient.EXPECT().
				Delete(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
				DoAndReturn(func(ctx context.Context, sec *corev1.Secret, opt *client.DeleteOptions) error {
					assert.Equal(t, istioSystem, sec.Namespace)
					assert.Equal(t, secretName, sec.Name)
					return nil
				})
		}
	}
	decoder := decoder()
	webhook := &AppConfigWebhook{Client: mockClient}
	webhook.InjectDecoder(decoder)
	req := admission.Request{
		AdmissionRequest: admissionv1beta1.AdmissionRequest{Operation: v1beta1.Delete},
	}
	req.OldObject = runtime.RawExtension{Raw: readYaml2Json(t, "hello-conf.yaml")}
	if dryRun {
		dryRun := true
		req.DryRun = &dryRun
	}
	res := webhook.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
	assert.Equal(t, 0, len(res.Patches))
}

func readYaml2Json(t *testing.T, path string) []byte {
	filename, _ := filepath.Abs(fmt.Sprintf("testdata/%s", path))
	yamlBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Error reading %v: %v", path, err)
	}
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		log.Error(err, "Error json marshal")
	}
	return jsonBytes
}
