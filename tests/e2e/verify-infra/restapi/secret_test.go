// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var uniq string = string(uuid.NewUUID())

var jsonGenericSecret string
var jsonGenericSecretPatch string
var jsonDockerSecret string
var jsonDockerSecretPatch string

const DefaultNamespace = "default"
const TypeGeneric = "generic"
const TypeDocker = "docker-registry"

var GenericSecretName = "testsecret-1" + uniq
var DockerSecretName = "testsecret-2" + uniq

var genericSecret = v1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      GenericSecretName,
		Namespace: DefaultNamespace,
	},
	Data: map[string][]byte{
		"n1": []byte("v1"),
		"n2": []byte("v2"),
	},
	Type: v1.SecretTypeOpaque,
}

var genericSecretPatch = v1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      GenericSecretName,
		Namespace: DefaultNamespace,
	},
	Data: map[string][]byte{
		"n1": []byte("v1"),
		"n3": []byte("v3"),
	},
	Type: v1.SecretTypeOpaque,
}

var dockerSecret = v1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      DockerSecretName,
		Namespace: DefaultNamespace,
	},
	Data: map[string][]byte{
		".dockerconfigjson": []byte(
			`{"auths":{"myserver":{"Username":"user1","Password":"pw1","Email":"user@company.com"}}}`),
	},
	Type: v1.SecretTypeDockerConfigJson,
}

var dockerSecretPatch = v1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      DockerSecretName,
		Namespace: DefaultNamespace,
	},
	Data: map[string][]byte{
		".dockerconfigjson": []byte(
			`{"auths":{"myserver":{"Username":"user2","Password":"pw2","Email":"user@company.com"}}}`),
	},
	Type: v1.SecretTypeDockerConfigJson,
}

var _ = ginkgo.BeforeSuite(func() {
	api = pkg.GetApiEndpoint()
	b, _ := json.Marshal(genericSecret)
	jsonGenericSecret = string(b)
	b, _ = json.Marshal(genericSecretPatch)
	jsonGenericSecretPatch = string(b)
	b, _ = json.Marshal(dockerSecret)
	jsonDockerSecret = string(b)
	b, _ = json.Marshal(dockerSecretPatch)
	jsonDockerSecretPatch = string(b)
})

var _ = ginkgo.Describe("Secrets", func() {

	ginkgo.Context("Fetching all secrets",
		func() {
			ginkgo.It("exist in default namespaces",
				func() {
					var secrets v1.SecretList
					resp, err := api.Get(fmt.Sprintf("api/v1/namespaces/%s/secrets", DefaultNamespace))
					pkg.ExpectHttpOk(resp, err, fmt.Sprintf("Error calling Get Secrets REST API, response %v, err %v", resp, err))
					gomega.Expect(resp.BodyErr).To(gomega.BeNil(), fmt.Sprintf("Error reading HTTP response body, error %s", resp.BodyErr))
					gomega.Expect(resp.StatusCode >= http.StatusBadRequest).To(gomega.BeFalse(), fmt.Sprintf("Bad status code %d", resp.StatusCode))
					err = json.Unmarshal(resp.Body, &secrets)
					gomega.Expect(err).To(gomega.BeNil(), "Error decoding HTTP JSON response %v, error %v", resp.Body, err)
				})
		})
	ginkgo.Context("generic type",
		func() {
			ginkgo.It("can be created",
				func() {
					kSecret := expectCreateSecret(GenericSecretName, jsonGenericSecret)
					expectSecretMatch(kSecret, &genericSecret)
				})
			ginkgo.It("then updated ",
				func() {
					resp, err := expectPatchSecret(GenericSecretName, jsonGenericSecretPatch)
					pkg.ExpectHttpOk(resp, err, "Error calling PATCH Secrets REST API")
					kSecret, err := pkg.GetSecret(DefaultNamespace, GenericSecretName)
					gomega.Expect(err).To(gomega.BeNil(), "Error getting secret "+GenericSecretName)
					expectSecretMatch(kSecret, &genericSecretPatch)
				})
			ginkgo.It("then deleted ",
				func() {
					expectDeleteSecret(GenericSecretName)
				})
		})
	ginkgo.Context("docker type",
		func() {
			ginkgo.It("can be created",
				func() {
					kSecret := expectCreateSecret(DockerSecretName, jsonDockerSecret)
					expectSecretMatch(kSecret, &dockerSecret)
				})
			ginkgo.It("then updated ",
				func() {
					resp, err := expectPatchSecret(DockerSecretName, jsonDockerSecretPatch)
					pkg.ExpectHttpOk(resp, err, "Error calling PATCH Secrets REST API")
					kSecret, err := pkg.GetSecret(DefaultNamespace, DockerSecretName)
					gomega.Expect(err).To(gomega.BeNil(), "Error getting secret "+DockerSecretName)
					expectSecretMatch(kSecret, &dockerSecretPatch)
				})
			ginkgo.It("then deleted ",
				func() {
					expectDeleteSecret(DockerSecretName)
				})
		})

	ginkgo.Context("negative tests",
		func() {
			ginkgo.It(fmt.Sprintf("create secret with bad payload should fail with  %v", http.StatusBadRequest),
				func() {
					resp, err := api.Post(fmt.Sprintf("api/v1/namespaces/%s/secrets", DefaultNamespace), bytes.NewBuffer([]byte("badjson{")))
					pkg.ExpectHttpStatus(http.StatusBadRequest, resp, err, "Create with bad payload returned wrong status")
				})
			ginkgo.It(fmt.Sprintf("create duplicate should fail with status %v", http.StatusConflict), func() {
				expectCreateSecret(GenericSecretName, jsonGenericSecret)
				resp, err := api.Post(fmt.Sprintf("api/v1/namespaces/%s/secrets", DefaultNamespace), bytes.NewBuffer([]byte(jsonGenericSecret)))
				pkg.ExpectHttpStatus(http.StatusConflict, resp, err, "Create duplicate secret returned wrong status")
			})
			ginkgo.It(fmt.Sprintf("update secret with wrong name should fail with status %v", http.StatusBadRequest), func() {
				resp, err := expectPatchSecret("badSecretName", jsonGenericSecretPatch)
				pkg.ExpectHttpStatus(http.StatusBadRequest, resp, err, "Update secret with wrong name returned wrong status")
			})
			ginkgo.It(fmt.Sprintf("delete secret with wrong name should fail with status %v", http.StatusNotFound), func() {
				resp, err := api.Delete(fmt.Sprintf("api/v1/namespaces/%s/secrets/%s", DefaultNamespace, "badSecretName"))
				pkg.ExpectHttpStatus(http.StatusNotFound, resp, err, "Error creating DELETE request")
			})
			ginkgo.It("delete with correct name ", func() {
				expectDeleteSecret(GenericSecretName)
			})
		})
})

// Check if the secret data returned from the server matches the expected values
func expectSecretMatch(kSecret *v1.Secret, secret *v1.Secret) {
	gomega.Expect(kSecret.Name).To(gomega.Equal(secret.Name))
	gomega.Expect(secret.Data).To(gomega.HaveLen(len(kSecret.Data)), "The generic data size doesn't match")
	for name, data := range secret.Data {
		kDataVal, ok := kSecret.Data[name]
		gomega.Expect(ok).To(gomega.BeTrue(), "key data missing map entry")
		gomega.Expect(string(data)).To(gomega.Equal(string(kDataVal)), "key data doesn't match")
	}

}

// submit HTTP POST request and fail on error conditions
func expectCreateSecret(name string, payload string) *v1.Secret {
	resp, err := api.Post(fmt.Sprintf("api/v1/namespaces/%s/secrets", DefaultNamespace), bytes.NewBuffer([]byte(payload)))
	pkg.ExpectHttpStatus(http.StatusCreated, resp, err, "Error calling CREATE REST API")

	kSecret, err := pkg.GetSecret(DefaultNamespace, name)
	gomega.Expect(err).To(gomega.BeNil(), "Error getting secret "+name)
	return kSecret
}

// submit HTTP PATCH request and fail on error conditions
func expectPatchSecret(name, body string) (resp *pkg.HttpResponse, err error) {
	return api.Patch(fmt.Sprintf("api/v1/namespaces/%s/secrets/%s", DefaultNamespace, name), bytes.NewBuffer([]byte(body)))
}

// submit HTTP DELETE request and fail on error conditions
func expectDeleteSecret(name string) error {
	resp, err := api.Delete(fmt.Sprintf("api/v1/namespaces/%s/secrets/%s", DefaultNamespace, name))
	pkg.ExpectHttpOk(resp, err, "Error calling DELETE REST API")

	_, err = pkg.GetSecret(DefaultNamespace, name)
	if !errors.IsNotFound(err) {
		ginkgo.Fail(fmt.Sprintf("Secret %s still exists, should have been deleted", name))
	}
	return err
}
