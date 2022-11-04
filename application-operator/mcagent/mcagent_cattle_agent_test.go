// Copyright (c) 2022, Oracle and/or its affiliates.
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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestSyncer_syncCattleClusterAgent tests the syncCattleClusterAgent of Syncer
// WHEN trying to sync the cattle-cluster-agent
// IF there exists no previous cattle agent hash
// THEN update the cattle-agent deployment, create the cattle credential and update the hash
func TestSyncer_syncCattleClusterAgent(t *testing.T) {
	asserts := assert.New(t)
	log := zap.S().With("test")

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
		CattleAgentHash:    "",
	}

	err = s.syncCattleClusterAgent(kubeConfigPath)
	asserts.NoError(err)
	asserts.NotEmpty(s.CattleAgentHash)
}

// TestSyncer_syncCattleClusterAgent2 tests the syncCattleClusterAgent of Syncer
// WHEN trying to sync the cattle-cluster-agent
// IF the registration manifest doesn't have all the resources
// THEN do nothing and return
func TestSyncer_syncCattleClusterAgent2(t *testing.T) {
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
		CattleAgentHash:    "",
	}

	err = s.syncCattleClusterAgent("")
	asserts.NoError(err)
	asserts.Empty(s.CattleAgentHash)
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
