// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var uniq string = string(uuid.NewUUID())

var api *util.ApiEndpoint
var jsonGenericSecret string
var jsonGenericSecretPatch string
var jsonDockerSecret string
var jsonDockerSecretPatch string

type Secret struct {
	Id             string         `json:"id"`
	Cluster        string         `json:"cluster"`
	Type           string         `json:"type"`
	Name           string         `json:"name"`
	Namespace      string         `json:"namespace"`
	Status         string         `json:"status"`
	Data           []Generic      `json:"data,omitempty"`
	DockerRegistry DockerRegistry `json:"dockerRegistry,omitempty"`
}

type Generic struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DockerRegistry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Server   string `json:"server"`
}

const DefaultNamespace = "default"
const TypeGeneric = "generic"
const TypeDocker = "docker-registry"

var GenericSecretName = "testsecret-1" + uniq
var DockerSecretName = "testsecret-2" + uniq

var genericSecret = Secret{
	Id:        "",
	Cluster:   "local",
	Type:      TypeGeneric,
	Name:      GenericSecretName,
	Namespace: "default",
	Status:    "",
	Data: []Generic{
		{Name: "n1", Value: "v1"},
		{Name: "n2", Value: "v2"},
	},
	DockerRegistry: DockerRegistry{},
}

var genericSecretPatch = Secret{
	Id:        "",
	Cluster:   "local",
	Type:      TypeGeneric,
	Name:      GenericSecretName,
	Namespace: "default",
	Status:    "",
	Data: []Generic{
		{Name: "n1", Value: "v1"},
		{Name: "n3", Value: "v3"},
	},
	DockerRegistry: DockerRegistry{},
}

var dockerSecret = Secret{
	Id:        "",
	Cluster:   "local",
	Type:      TypeDocker,
	Name:      DockerSecretName,
	Namespace: "default",
	Status:    "",
	Data:      []Generic{},
	DockerRegistry: DockerRegistry{
		Username: "user1",
		Password: "pw1",
		Email:    "user@company.com",
		Server:   "myserver",
	},
}

var dockerSecretPatch = Secret{
	Id:        "",
	Cluster:   "local",
	Type:      TypeDocker,
	Name:      DockerSecretName,
	Namespace: "default",
	Status:    "",
	Data:      []Generic{},
	DockerRegistry: DockerRegistry{
		Username: "user2",
		Password: "pw2",
		Email:    "user@company.com",
		Server:   "myserver",
	},
}

var _ = ginkgo.BeforeSuite(func() {
	api = util.GetApiEndpoint()
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
	var curUID string

	ginkgo.Context("Fetching all secrets",
		func() {
			ginkgo.It("exist in model/binding namespaces",
				func() {
					var secrets []Secret
					resp, err := api.GetSecrets()
					util.ExpectHttpOk(resp, err, "Error calling Get Secrets REST API")
					gomega.Expect(resp.BodyErr).To(gomega.BeNil(), "Error reading HTTP response body")
					err = json.Unmarshal(resp.Body, &secrets)
					gomega.Expect(err).To(gomega.BeNil(), "Error decoding HTTP JSON response %v", resp.Body)
				})
		})
	ginkgo.Context("generic type",
		func() {
			ginkgo.It("can be created",
				func() {
					kSecret, id, _ := expectCreateSecret(GenericSecretName, jsonGenericSecret)
					curUID = id
					expectSecretMatch(kSecret, genericSecret)
				})
			ginkgo.It("then updated ",
				func() {
					expectUID(curUID)
					resp, err := expectPatchSecret(curUID, jsonGenericSecretPatch)
					util.ExpectHttpOk(resp, err, "Error calling PATCH Secrets REST API")
					kSecret := util.GetSecret(DefaultNamespace, GenericSecretName)
					expectSecretMatch(kSecret, genericSecretPatch)
				})
			ginkgo.It("then deleted ",
				func() {
					expectUID(curUID)
					expectDeleteSecret(curUID, GenericSecretName)
				})
		})
	ginkgo.Context("docker type",
		func() {
			ginkgo.It("can be created",
				func() {
					kSecret, id, _ := expectCreateSecret(DockerSecretName, jsonDockerSecret)
					curUID = id
					expectSecretMatch(kSecret, dockerSecret)
				})
			ginkgo.It("then updated ",
				func() {
					expectUID(curUID)
					resp, err := expectPatchSecret(curUID, jsonDockerSecretPatch)
					util.ExpectHttpOk(resp, err, "Error calling PATCH Secrets REST API")
					kSecret := util.GetSecret(DefaultNamespace, DockerSecretName)
					expectSecretMatch(kSecret, dockerSecretPatch)
				})
			ginkgo.It("then deleted ",
				func() {
					expectUID(curUID)
					expectDeleteSecret(curUID, DockerSecretName)
				})
		})

	ginkgo.Context("negative tests",
		func() {
			ginkgo.It(fmt.Sprintf("create secret with bad payload should fail with  %v", http.StatusBadRequest),
				func() {
					resp, err := api.CreateSecret("badjson{")
					util.ExpectHttpStatus(http.StatusBadRequest, resp, err, "Create with bad payload returned wrong status")
				})
			ginkgo.It(fmt.Sprintf("create duplicate should fail with status %v", http.StatusConflict), func() {
				_, id, _ := expectCreateSecret(GenericSecretName, jsonGenericSecret)
				curUID = id
				resp, err := api.CreateSecret(jsonGenericSecret)
				util.ExpectHttpStatus(http.StatusConflict, resp, err, "Create duplicate secret returned wrong status")
			})
			ginkgo.It(fmt.Sprintf("update secret with wrong UID should fail with status %v", http.StatusNotFound), func() {
				resp, err := expectPatchSecret("badUID", jsonGenericSecretPatch)
				util.ExpectHttpStatus(http.StatusNotFound, resp, err, "Update secret with wrong UID returned wrong status")
			})
			ginkgo.It(fmt.Sprintf("delete secret with wrong UID should fail with status %v", http.StatusNotFound), func() {
				resp, err := api.DeleteSecret("badUID")
				util.ExpectHttpStatus(http.StatusNotFound, resp, err, "Error creating DELETE request")
			})
			ginkgo.It("delete with correct UID ", func() {
				expectUID(curUID)
				expectDeleteSecret(curUID, GenericSecretName)
			})
		})
})

// Check if the secret data returned from the server matches the expected values
func expectSecretMatch(kSecret *v1.Secret, secret Secret) {
	gomega.Expect(kSecret.Name).To(gomega.Equal(secret.Name))
	switch secret.Type {
	case TypeGeneric:
		gomega.Expect(secret.Data).To(gomega.HaveLen(len(kSecret.Data)), "The generic data size doesn't match")
		for _, data := range secret.Data {
			kDataVal, ok := kSecret.Data[data.Name]
			gomega.Expect(ok).To(gomega.BeTrue(), "Generic key data missing map entry")
			gomega.Expect(data.Value).To(gomega.Equal(string(kDataVal)), "Generic key data doesn't match")
		}

	case TypeDocker:
		dockerData := fmt.Sprintf(
			`{"auths":{"%s":{"Username":"%s","Password":"%s","Email":"%s"}}}`,
			secret.DockerRegistry.Server,
			secret.DockerRegistry.Username,
			secret.DockerRegistry.Password,
			secret.DockerRegistry.Email)

		kData := string((kSecret.Data)[".dockerconfigjson"])
		gomega.Expect(dockerData).To(gomega.Equal(kData), "The docker data doesn't match")

	default:
		ginkgo.Fail("Invalid secret type returned from REST API " + secret.Type)
	}
}

// Get UID match and fail on error conditions
func expectUID(uid string) {
	gomega.Expect(uid).To(gomega.Not(gomega.BeEmpty()), "Skipping test since UID is invalid")
}

// submit HTTP POST request and fail on error conditions
func expectCreateSecret(name string, payload string) (*v1.Secret, string, error) {
	resp, err := api.CreateSecret(payload)
	util.ExpectHttpOk(resp, err, "Error calling CREATE REST API")

	kSecret := util.GetSecret(DefaultNamespace, name)
	gomega.Expect(kSecret).NotTo(gomega.BeNil(), "Error getting secret "+name)
	return kSecret, string(kSecret.UID), nil
}

// submit HTTP PATCH request and fail on error conditions
func expectPatchSecret(id, body string) (resp *util.HttpResponse, err error) {
	return api.PatchSecret(id, body)
}

// submit HTTP DELETE request and fail on error conditions
func expectDeleteSecret(id string, name string) error {
	resp, err := api.DeleteSecret(id)
	util.ExpectHttpOk(resp, err, "Error calling DELETE REST API")

	secret := util.GetSecret(DefaultNamespace, name)
	if secret != nil {
		ginkgo.Fail(fmt.Sprintf("Secret %s still exists, should have been deleted", name))
	}
	return nil
}
