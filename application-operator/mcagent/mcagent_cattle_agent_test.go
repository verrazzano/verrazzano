// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestSyncer_syncCattleClusterAgent tests the syncCattleClusterAgent of Syncer
// GIVEN a call to syncCattleClusterAgent
// WHEN there exists no previous cattle agent hash
// THEN update the cattle-cluster-agent deployment, create the cattle-credential secret and update the hash
func TestSyncer_syncCattleClusterAgent(t *testing.T) {
	asserts := assert.New(t)
	log := zap.S().With("test")

	// Override the getDeployment function because there is a bug in the fake logic that does not
	// handle names containing more than one hyphen.
	setDeploymentFunc(func(config *rest.Config, namespace string, name string) (*appsv1.Deployment, error) {
		return nil, errors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "deployments"}, name)
	})
	defer resetDeploymentFunc()

	scheme := runtime.NewScheme()
	err := corev1.SchemeBuilder.AddToScheme(scheme)
	asserts.NoError(err)

	secret, err := getSampleSecret("testdata/registration-manifest.yaml")
	asserts.NoError(err)

	adminClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&secret).Build()

	server := newServer()
	defer server.Close()

	err = createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	s := Syncer{
		AdminClient:        adminClient,
		Log:                log,
		ManagedClusterName: "cluster1",
		Context:            context.TODO(),
	}

	newCattleAgentHash, err := s.syncCattleClusterAgent("", kubeConfigPath)
	asserts.NoError(err)
	asserts.NotEmpty(newCattleAgentHash)
}

// TestSyncer_syncCattleClusterAgentNoRancherManifest tests the syncCattleClusterAgent of Syncer
// GIVEN a call to syncCattleClusterAgent
// WHEN the registration manifest doesn't have the required rancher resources
// THEN do nothing and return
func TestSyncer_syncCattleClusterAgentNoRancherManifest(t *testing.T) {
	asserts := assert.New(t)
	log := zap.S().With("test")

	scheme := runtime.NewScheme()
	err := corev1.SchemeBuilder.AddToScheme(scheme)
	asserts.NoError(err)

	secret, err := getSampleSecret("testdata/incomplete-registration-manifest.yaml")
	asserts.NoError(err)

	adminClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&secret).Build()

	s := Syncer{
		AdminClient:        adminClient,
		Log:                log,
		ManagedClusterName: "cluster1",
		Context:            context.TODO(),
	}

	newCattleAgentHash, err := s.syncCattleClusterAgent("", "")
	asserts.NoError(err)
	asserts.Empty(newCattleAgentHash)
}

// TestSyncer_syncCattleClusterAgentHashExists tests the syncCattleClusterAgent of Syncer
func TestSyncer_syncCattleClusterAgentHashExists(t *testing.T) {
	asserts := assert.New(t)
	log := zap.S().With("test")

	// Override the getDeployment function because there is a bug in the fake logic that does not
	// handle names containing more than one hyphen.
	setDeploymentFunc(func(config *rest.Config, namespace string, name string) (*appsv1.Deployment, error) {
		return nil, errors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "deployments"}, name)
	})
	defer resetDeploymentFunc()

	scheme := runtime.NewScheme()
	err := corev1.SchemeBuilder.AddToScheme(scheme)
	asserts.NoError(err)

	secret, err := getSampleSecret("testdata/registration-manifest.yaml")
	asserts.NoError(err)

	adminClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&secret).Build()

	server := newServer()
	defer server.Close()

	err = createFakeKubeConfig(server.URL)
	defer deleteFakeKubeConfig()
	asserts.NoError(err)

	kubeConfigPath, err := getFakeKubeConfigPath()
	asserts.NoError(err)

	// Initialise the cattleAgentHash with a random hash string
	previousCattleAgentHash := "[1 61 83 140 196 214 70 37 227 165 192 170 234 13 222 47 123]"

	s := Syncer{
		AdminClient:        adminClient,
		Log:                log,
		ManagedClusterName: "cluster1",
		Context:            context.TODO(),
	}

	// GIVEN a call to syncCattleClusterAgent
	// WHEN a hash already exists
	// THEN if the hash has changed, update the resources and the hash
	newCattleAgentHash, err := s.syncCattleClusterAgent(previousCattleAgentHash, kubeConfigPath)
	asserts.NoError(err)
	asserts.NotEmpty(newCattleAgentHash)
	asserts.NotEqual(previousCattleAgentHash, newCattleAgentHash)

	previousCattleAgentHash = newCattleAgentHash

	// GIVEN a call to syncCattleClusterAgent
	// WHEN a hash already exists
	// THEN if the hash has not changed, do nothing
	newerCattleAgentHash, err := s.syncCattleClusterAgent(previousCattleAgentHash, kubeConfigPath)
	asserts.NoError(err)
	asserts.NotEmpty(newerCattleAgentHash)
	asserts.Equal(previousCattleAgentHash, newerCattleAgentHash)
}

// newServer returns a httptest server which the
// dynamic client and discovery client can send
// GET/POST/PATCH/DELETE requests to instead of a real cluster
func newServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var obj interface{}
		switch req.URL.Path {
		case "/api/v1/namespaces/cattle-system":
			obj = &metav1.APIVersions{
				TypeMeta: metav1.TypeMeta{
					Kind: "APIVersions",
				},
				Versions: []string{
					"v1",
				},
			}
		case "/api":
			obj = &metav1.APIVersions{
				Versions: []string{
					"v1",
				},
			}
		case "/api/v1":
			obj = &metav1.APIResourceList{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{
					{Name: "secrets", Namespaced: true, Kind: "Secret"},
				},
			}
		case "/apis/apps/v1/namespaces/cattle-system/deployments/cattle-cluster-agent":
			body, _ := io.ReadAll(req.Body)
			w.Write(body)
			return
		case "/api/v1/namespaces/cattle-system/secrets":
			body, _ := io.ReadAll(req.Body)
			w.Write(body)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		output, err := json.Marshal(obj)
		if err != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}))
	return server
}

// createFakeKubeConfig creates a fake kubeconfig
// in the pwd with the url of the httptest server
func createFakeKubeConfig(url string) error {
	fakeKubeConfig, err := os.Create("dummy-kubeconfig")
	defer fakeKubeConfig.Close()

	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(fakeKubeConfig, `apiVersion: v1
clusters:
- cluster:
    # This is dummy data
    certificate-authority-data: RFVNTVkgQ0VSVElGSUNBVEU=
    server: %s
  name: user-test
users:
- name: user-test
contexts:
- context:
    cluster: user-test
    user: user-test
  name: user-test
current-context: user-test`, url)

	return err
}

func getFakeKubeConfigPath() (string, error) {
	pwd, err := os.Getwd()

	if err != nil {
		return pwd, err
	}

	pwd = pwd + "/dummy-kubeconfig"
	return pwd, nil
}

func deleteFakeKubeConfig() error {
	err := os.Remove("dummy-kubeconfig")
	return err
}
