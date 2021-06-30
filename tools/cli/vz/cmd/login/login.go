// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package login

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type LoginOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewLoginOptions(streams genericclioptions.IOStreams) *LoginOptions {
	return &LoginOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdLogin(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewLoginOptions(streams)
	cmd := &cobra.Command{
		Use:   "login api_url",
		Short: "vz login",
		Long:  "vz_login",
		//Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := login(args); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

var auth_code = ""

func login(args []string) error{

	var vz_api_url string
	var cli = false
	credentials := make(map[string]string)
	for index, element := range args {
		if index==0 {
			vz_api_url = element
			continue
		}
		if element=="u"{
			cli = true
			credentials["username"] = args[index+1]
		}
		if element=="p"{
			credentials["password"] = args[index+1]
		}
	}
	var jwtData map[string]interface{}
	if cli {
		jwtData = directFlowLogin(credentials)
	} else {
		jwtData = authFlowLogin()
	}
	fmt.Println(vz_api_url)
	fmt.Println(jwtData)
	caData := getCAData()
	f := CreateWithToken(vz_api_url,"verrazzano","verrazzano",caData,fmt.Sprintf("%v",jwtData["access_token"]))
	WriteToDisk(getKubeconfigLocation(),f)
	return nil
}

func getCAData() []byte {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	namespace := "verrazzano-system"

	restconfig, _ := kubeconfig.ClientConfig()
	coreclient, _ := corev1client.NewForConfig(restconfig)
	secret, err := coreclient.Secrets(namespace).Get(context.Background(), "system-tls", metav1.GetOptions{})
	if err != nil{
		fmt.Println("no secrets")
	}
	return (*secret).Data["ca.crt"]
}

func CreateWithToken(serverURL, clusterName, userName string, caCert []byte, token string) *clientcmdapi.Config {
	config := CreateBasic(serverURL, clusterName, userName, caCert)
	config.AuthInfos[userName] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	return config
}

func CreateBasic(serverURL, clusterName, userName string, caCert []byte) *clientcmdapi.Config {
	// Use the cluster and the username as the context name
	contextName := fmt.Sprintf("%s@%s", userName, clusterName)

	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			clusterName: {
				Server: serverURL,
				CertificateAuthorityData: caCert,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{},
		CurrentContext: contextName,
	}
}

func WriteToDisk(filename string, kubeconfig *clientcmdapi.Config) error {
	err := clientcmd.WriteToFile(*kubeconfig, filename)
	if err != nil {
		return err
	}

	return nil
}

func getKubeconfigLocation() string {
	kubeconfig := ""
	kubeconfigEnvVar := ""
	testKubeConfigEnvVar := os.Getenv("TEST_KUBECONFIG")
	if len(testKubeConfigEnvVar) > 0 {
		kubeconfigEnvVar = testKubeConfigEnvVar
	}

	if kubeconfigEnvVar == "" {
		kubeconfigEnvVar = os.Getenv("KUBECONFIG")
	}

	if len(kubeconfigEnvVar) > 0 {
		kubeconfig = kubeconfigEnvVar
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		// give up
		fmt.Println("Could not find kube config")
	}
	return kubeconfig
}

func directFlowLogin(credentials map[string]string) map[string]interface{}  {

	_ , userPresent := credentials["username"]
	if userPresent == false {
		fmt.Print("Username:")
		var username string
		fmt.Scanln(&username)
		credentials["username"] = username
	}

	_ , passwordPresent := credentials["password"]
	if passwordPresent == false {
		fmt.Print("Password:")
		var password string
		fmt.Scanln(&password)
		credentials["password"] = password
	}

	jwtData := requestJWTDirectAccess(credentials)

	fmt.Println(jwtData["access_token"])
	fmt.Println(jwtData["id_token"])
	fmt.Println()
	fmt.Println()
	return jwtData
}

func authFlowLogin() map[string]interface{}  {
	listener := getFreePort()
	code_verifier, code_challenge := generateRandomCodeVerifier()
	redirect_uri := generateRedirectURI(listener)
	login_url := generateKeycloakAPIURL(code_challenge, redirect_uri)

	go func() {
		for auth_code == "" {

		}
		listener.Close()
	}()

	time.Sleep(time.Second)

	openUrlInBrowser(login_url)

	fmt.Println("Using port:", listener.Addr().(*net.TCPAddr).Port)
	http.HandleFunc("/", handle)
	http.Serve(listener, nil)

	time.Sleep(time.Second)

	for auth_code == "" {

	}

	fmt.Println(code_verifier)

	jwtData := requestJWT(redirect_uri, code_verifier)

	fmt.Println(jwtData["access_token"])
	fmt.Println(jwtData["refresh_token"])
	fmt.Println()
	fmt.Println()
	return jwtData
}


// Encodes byte stream to base64-url string
// Returns the encoded string
func encode(msg []byte) string {
	encoded := base64.StdEncoding.EncodeToString(msg)
	encoded = strings.Replace(encoded, "+", "-", -1)
	encoded = strings.Replace(encoded, "/", "_", -1)
	encoded = strings.Replace(encoded, "=", "", -1)
	return encoded
}

// Generates a random code verifier and then produces a code challenge using it.
// Returns the produced code_verifier and code_challenge pair
func generateRandomCodeVerifier() (string, string) {
	length := 32
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length, length)
	for i := 0; i < length; i++ {
		b[i] = byte(r.Intn(255))
	}
	code_verifier := encode(b)
	h := sha256.New()
	h.Write([]byte(code_verifier))
	code_challenge := encode(h.Sum(nil))
	return code_verifier, code_challenge
}

// Handler function for http server
// The server page's html,js,etc code is embedded here.
func handle(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse(r.URL.String())
	m, _ := url.ParseQuery(u.RawQuery)
	auth_code = m["code"][0]
	fmt.Println("authcode")
	fmt.Println("authcode")
	fmt.Println("authcode")
	fmt.Println(auth_code)
	fmt.Fprintln(w, "<p>You can close this tab now</p>")
}

// Fetch an available port
// Return in the form of listener
func getFreePort() net.Listener {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	return listener
}

// Generates redirect_uri using the given port number
// return string of the form `http://localhost:1234`
func generateRedirectURI(listener net.Listener) string {
	port := listener.Addr().(*net.TCPAddr).Port
	u := &url.URL{
		Scheme: "http",
		Host:   "localhost:" + strconv.Itoa(port),
	}
	return u.String()
}

// Accepts url parameters in the form of map[string][string]
// Returns concatenated list of url parameters
// Return string of the form `code=xyz&status=abc`
func concatURLParams(urlParams map[string]string) string {
	var params []string
	for k, v := range urlParams {
		params = append(params, k+"="+v)
	}
	return strings.Join(params, "&")
}

func getClientId() string{
	clientId := os.Getenv("VZ_CLIENT_ID")
	if clientId == "" {
		return "webui"
	} else {
		return clientId
	}
}

func getKeycloakURL() string{
	url := os.Getenv("VZ_KEYCLOAK_URL")
	if url == "" {
		return "keycloak.default.172.18.0.231.nip.io:443"
	} else {
		return url
	}
}

func getVerrazzanoRealm() string{
	realmName := os.Getenv("VZ_REALM")
	if realmName == "" {
		return "verrazzano-system"
	} else {
		return realmName
	}
}

// Generates the keycloak api url and returns it
// Return string of the form `https://keycloak.xyz.io:123/auth/realms/verrazzano-system/protocol/openid-connect/auth?redirect_uri=abc&state=xyz...`
func generateKeycloakAPIURL(code_challenge string, redirect_uri string) string {
	urlParams := map[string]string{
		"client_id":             getClientId(),
		"response_type":         "code",
		"state":                 "fj8o3n7bdy1op5",
		"redirect_uri":          redirect_uri,
		"code_challenge":        code_challenge,
		"code_challenge_method": "S256",
	}
	u := &url.URL{
		Scheme:   "https",
		Host:     getKeycloakURL(),
		Path:     "auth/realms/" + getVerrazzanoRealm() + "/protocol/openid-connect/auth",
		RawQuery: concatURLParams(urlParams),
	}
	return u.String()
}

// Non-blocking browser opener function
func openUrlInBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

// Gnerates and returns keycloak server api url to get the jwt token
func generateKeycloakTokenURL() string {
	u := &url.URL{
		Scheme: "https",
		Host:   getKeycloakURL(),
		Path:   "/auth/realms/" + getVerrazzanoRealm() + "/protocol/openid-connect/token",
	}
	return u.String()
}

// Requests the jwt token on out behalf
// Returns the key-value pairs obtained from server in the form of a map
func requestJWT(redirect_uri string, code_verifier string) map[string]interface{} {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	token_url := generateKeycloakTokenURL()

	grantParams := url.Values{}
	grantParams.Set("grant_type", "authorization_code")
	grantParams.Set("client_id", getClientId())
	grantParams.Set("code", auth_code)
	grantParams.Set("redirect_uri", redirect_uri)
	grantParams.Set("code_verifier", code_verifier)
	grantParams.Set("scope","openid")

	client := &http.Client{}
	request, _ := http.NewRequest(http.MethodPost, token_url, strings.NewReader(grantParams.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	response, _ := client.Do(request)
	fmt.Println(response)
	responseBody, _ := ioutil.ReadAll(response.Body)

	var jsondata map[string]interface{}
	json.Unmarshal([]byte(responseBody), &jsondata)
	return jsondata
}

// Requests the jwt token  through direct access grant flow.
// Returns the key-value pairs obtained from server in the form of a map
func requestJWTDirectAccess(credentials map[string]string) map[string]interface{} {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	token_url := generateKeycloakTokenURL()

	grantParams := url.Values{}
	grantParams.Set("grant_type", "password")
	grantParams.Set("client_id", getClientId())
	grantParams.Set("username", credentials["username"])
	grantParams.Set("password", credentials["password"])
	grantParams.Set("scope","openid")


	client := &http.Client{}
	request, _ := http.NewRequest(http.MethodPost, token_url, strings.NewReader(grantParams.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	response, _ := client.Do(request)
	fmt.Println(response)
	responseBody, _ := ioutil.ReadAll(response.Body)

	var jsondata map[string]interface{}
	json.Unmarshal([]byte(responseBody), &jsondata)
	fmt.Println(jsondata)
	return jsondata
}
