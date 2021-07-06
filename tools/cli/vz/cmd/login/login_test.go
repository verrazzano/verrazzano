// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package login

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	projectclientset "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	clustersclientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned/fake"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"io"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

var (
	fakeVerrazzanoAPIURL = "verrazzano.fake.nip.io/12345"
	fakeVerrazzanoClientID = "fakeclient"
	fakeVerrazzanoRealm = "fakerealm"
	fakeAuthCode = "euifgewuhfoieuwhfiuoewhfioewhfoihwfihfgiheriofwerhfiruhgoreihgccccccccccccccccc"
	test1 = []struct {
		args []string
		output string
	}{
		{
			[]string{fakeVerrazzanoAPIURL},
			"Login successful!\n",
		},
	}
	test2 = []struct {
		args []string
		output string
	}{
		{
			[]string{fakeVerrazzanoAPIURL},
			"Already Logged in\n",
		},
	}
)

var status string
var codeChallenge string

type TestKubernetes struct {
	fakeProjectClient  projectclientset.Interface
	fakeClustersClient clustersclientset.Interface
	fakek8sClient      kubernetes.Interface
}

func (o *TestKubernetes) GetKubeConfig() *rest.Config {
	config := &rest.Config{
		Host: "https://1.2.3.4:1234",
	}
	return config
}

func (o *TestKubernetes) NewClustersClientSet() (clustersclientset.Interface, error) {
	return o.fakeClustersClient, nil
}

func (o *TestKubernetes) NewProjectClientSet() (projectclientset.Interface, error) {
	return o.fakeProjectClient, nil
}

func (o *TestKubernetes) NewClientSet() kubernetes.Interface {
	return o.fakek8sClient
}

func authHandle(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse(r.URL.String())
	m, _ := url.ParseQuery(u.RawQuery)
	if m["client_id"][0] != fakeVerrazzanoClientID {
		status = "failure"
	}
	redirectURI := m["response_type"][0]
	codeChallenge = m["code_challenge"][0]
	http.Redirect(w, r, redirectURI+"?code="+fakeAuthCode, 302)
}

func getCodeChallenge(codeVerifier string) string {
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	codeChallenge := base64.StdEncoding.EncodeToString(h.Sum(nil))
	codeChallenge = strings.Replace(codeChallenge, "+", "-", -1)
	codeChallenge = strings.Replace(codeChallenge, "/", "_", -1)
	codeChallenge = strings.Replace(codeChallenge, "=", "", -1)
	return codeChallenge
}

type Token struct  {
	AccessToken string `json:"access_token"`
}

func tokenHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := Token{AccessToken : "myaccesstokendbufbujfbvndvbfurdfgbvnudjkfvbciudrbferfgrefbvrebgfuireffuerf"}
	json.NewEncoder(w).Encode(token)
	r.ParseForm()
	if r.Form.Get("code") != fakeAuthCode && getCodeChallenge(r.Form.Get("code_verifier")) != codeChallenge{
		status = "failure"
	} else {
		status = "success"
	}
}

func TestNewCmdLogin(t *testing.T) {

	asserts := assert.New(t)

	// Create a fake keycloak server at some random available port
	listener, err := net.Listen("tcp", ":0")
	asserts.NoError(err)
	http.HandleFunc("/auth/realms/" + fakeVerrazzanoRealm + "/protocol/openid-connect/auth",
		authHandle,
	)
	http.HandleFunc("/auth/realms/" + fakeVerrazzanoRealm + "/protocol/openid-connect/token",
		tokenHandle,
	)
	go func() {
		http.Serve(listener,
			nil,
		)
	}()

	// Create fake kubeconfig
	createFakeKubeConfig(asserts)
	currentDirectory , err := os.Getwd()
	asserts.NoError(err)


	// Create fake environment variables
	os.Setenv("VZ_CLIENT_ID",fakeVerrazzanoClientID)
	os.Setenv("VZ_REALM",fakeVerrazzanoRealm)
	os.Setenv("VZ_KEYCLOAK_URL","http://localhost:"+strconv.Itoa(listener.Addr().(*net.TCPAddr).Port))
	os.Setenv("KUBECONFIG",currentDirectory+"/fakekubeconfig")

	// Create fake kubernetes interface
	fakeKubernetes := &TestKubernetes{
		fakeClustersClient: fake.NewSimpleClientset(),
		fakek8sClient:      k8sfake.NewSimpleClientset(),
	}
	asserts.NoError(newFakeSecret(fakeKubernetes.fakek8sClient))

	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdLogin(streams, fakeKubernetes)


	for _, test := range test1 {
		status = "pending"
		testCmd.SetArgs(test.args)
		asserts.NoError(testCmd.Execute())
		asserts.Equal(status, "success")
		asserts.Equal(test.output,outBuffer.String())

		kubeConfig, _ := clientcmd.LoadFromFile("fakekubeconfig")
		_ , ok := kubeConfig.Clusters["verrazzano"]
		asserts.Equal(ok, true)
		_ , ok  = kubeConfig.AuthInfos["verrazzano"]
		asserts.Equal(ok, true)
		_ , ok  = kubeConfig.Contexts[kubeConfig.CurrentContext]
		asserts.Equal(ok, true)
		asserts.Equal(strings.Split(kubeConfig.CurrentContext,"@")[0],"verrazzano")

		outBuffer.Reset()
	}
	os.Remove("fakekubeconfig")
}

func TestRepeatedLogin(t *testing.T) {
	asserts := assert.New(t)

	// Create a fake clone of kubeconfig
	createFakeKubeConfig(asserts)
	currentDirectory , err := os.Getwd()
	asserts.NoError(err)

	// Add fake clusters,usernames,contexts..
	verrazzanoAPIURL := "verrazzano.fake.nip.io/12345"
	fakeCAData := []byte("LS0tCmFwaVZlcnNpb246IHYxCmRhdGE6CiAgYWRtaW4ta3ViZWNvbmZpZzogWTJ4MWMzUmxjbk02Q2kwZ1kyeDFjM1JsY2pvS0lDQWdJR05sY25ScFptbGpZWFJsTFdGMWRHaHZjbWwwZVMxa1lYUmhPaUJNVXpCMFRGTXhRMUpWWkVwVWFVSkVVbFpL")
	fakeAccessToken := "fhuiewhfbudsefbiewbfewofnhoewnfoiewhfouewhbfgonewoifnewohfgoewnfgouewbugoewhfgojhew"
	kubeconfig, err := clientcmd.LoadFromFile("fakekubeconfig")
	asserts.NoError(err)

	helpers.SetCluster(kubeconfig,
		"verrazzano",
		verrazzanoAPIURL,
		fakeCAData,
	)

	helpers.SetUser(kubeconfig,
		"verrazzano",
		fakeAccessToken,
	)

	helpers.SetContext(kubeconfig,
		"verrazzano" + "@" + kubeconfig.CurrentContext,
		"verrazzano",
		"verrazzano",
	)

	helpers.SetCurrentContext(kubeconfig,
		"verrazzano"+"@"+ kubeconfig.CurrentContext,
	)
	err = clientcmd.WriteToFile(*kubeconfig,
		"fakekubeconfig",
	)
	asserts.NoError(err)

	// Set environment variable for kubeconfig
	os.Setenv("KUBECONFIG",currentDirectory+"/fakekubeconfig")

	// Create fake kubernetes interface
	fakeKubernetes := &TestKubernetes{
		fakeClustersClient: fake.NewSimpleClientset(),
		fakek8sClient:      k8sfake.NewSimpleClientset(),
	}
	asserts.NoError(newFakeSecret(fakeKubernetes.fakek8sClient))

	streams, _, outBuffer, _ := genericclioptions.NewTestIOStreams()
	testCmd := NewCmdLogin(streams, fakeKubernetes)

	for _, test := range test2 {

		testCmd.SetArgs(test.args)
		asserts.NoError(testCmd.Execute())
		asserts.Equal(test.output,outBuffer.String())
		outBuffer.Reset()
	}
	os.Remove("fakekubeconfig")
}

func createFakeKubeConfig(asserts *assert.Assertions) {
	originalKubeConfigLocation, err := helpers.GetKubeConfigLocation()
	asserts.NoError(err)
	originalKubeConfig, err := os.Open(originalKubeConfigLocation)
	asserts.NoError(err)
	fakeKubeConfig, err := os.Create("fakekubeconfig")
	asserts.NoError(err)
	io.Copy(fakeKubeConfig, originalKubeConfig)
	originalKubeConfig.Close()
	fakeKubeConfig.Close()
}

// Create fake certificate authority data
func newFakeSecret(fakek8sClient kubernetes.Interface) error {
	fakeSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-tls",
			Namespace: "verrazzano-system",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "clusters.verrazzano.io/v1alpha1",
				Kind:       "VerrazzanoSystemCluster",
				Name:       "logintest",
			}},
		},
		// Garbage Data
		Data: map[string][]byte{
			"ca.crt": []byte("LS0tCmFwaVZlcnNpb246IHYxCmRhdGE6CiAgYWRtaW4ta3ViZWNvbmZpZzogWTJ4MWMzUmxjbk02Q2kwZ1kyeDFjM1JsY2pvS0lDQWdJR05sY25ScFptbGpZWFJsTFdGMWRHaHZjbWwwZVMxa1lYUmhPaUJNVXpCMFRGTXhRMUpWWkVwVWFVSkVVbFpL"),
		},
	}
	_, err := fakek8sClient.CoreV1().Secrets("verrazzano-system").Create(context.Background(), fakeSecret, metav1.CreateOptions{})
	return err
}
