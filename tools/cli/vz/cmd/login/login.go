// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package login

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
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
		Args:  cobra.ExactArgs(1),
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

func login(args []string) error{

	var vz_api_url string
	var cli = false
	credentials := make(map[string]string)
	var err error

	// Extract parameters from args
	for index, element := range args {
		if index==0 {
			vz_api_url = element
			continue
		}
	}

	// Using the passed arguments, choose which grant flow to follow
	var jwtData map[string]interface{}
	if cli {
		jwtData = directFlowLogin(credentials)
	} else {
		jwtData, err = authFlowLogin()
	}
	if err!=nil {
		fmt.Println("Grant flow failed")
		return err
	}

	// Obtain the certificate authority data in the form of byte stream
	caData,err := getCAData()
	if err!=nil {
		fmt.Println("Unable to obtain certificate authority data")
		fmt.Println("Make sure you are in the right context")
		return err
	}

	kubeConfigLoc,err := getKubeconfigLocation()
	if err!=nil {
		return err
	}
	mykubeConfig, _ := clientcmd.LoadFromFile(kubeConfigLoc)
	mykubeConfig.Clusters["verrazzano"] = &clientcmdapi.Cluster{
		Server: vz_api_url,
		CertificateAuthorityData: caData,
	}
	mykubeConfig.AuthInfos["verrazzano"] = &clientcmdapi.AuthInfo{
		Token: fmt.Sprintf("%v",jwtData["access_token"]),
	}
	mykubeConfig.Contexts["verrazzano"+"@"+mykubeConfig.CurrentContext] = &clientcmdapi.Context{
		Cluster: "verrazzano",
		AuthInfo: "verrazzano",
	}
	mykubeConfig.CurrentContext = "verrazzano"+"@"+mykubeConfig.CurrentContext
	err = WriteKubeConfigToDisk( kubeConfigLoc,
		mykubeConfig,
	)
	if err!=nil {
		fmt.Println("Unable to write the new kubconfig to disk")
		return err
	}
	fmt.Println("Login successful!")
	return nil
}

var auth_code = ""	//	Http handle fills this after keycloak authentication

// A function to put together all the requirements of authorization grant flow
// Returns the final jwt token as a map
func authFlowLogin() (map[string]interface{},error)  {

	// Obtain a available port in non-well known port range
	listener := getFreePort()

	// Generate random code verifier and code challenge pair
	code_verifier, code_challenge := generateRandomCodeVerifier()

	// Generate the redirect uri using the port obtained
	redirect_uri := generateRedirectURI(listener)

	// Generate the login keycloak url by passing the required url parameters
	login_url := generateKeycloakAPIURL(code_challenge,
		redirect_uri,
	)

	// Busy wait when the authorization code is still not filled by http handle
	// Close the listener once we obtain it
	go func() {
		for auth_code == "" {

		}
		listener.Close()
	}()

	// Make sure the go routine is running
	time.Sleep(time.Second)

	// Open the generated keycloak login url in the browser
	err := openUrlInBrowser(login_url)
	if err != nil {
		fmt.Println("Unable to open browser")
	}

	// Set the handle function and start the http server
	http.HandleFunc("/",
		handle)
	http.Serve(listener,
		nil)

	// Obtain the JWT token by exchanging it with auth_code
	jwtData, err := requestJWT(redirect_uri,
		code_verifier)
	if err != nil {
		fmt.Println("Unable to obtain the JWT token")
		return jwtData, err
	}
	return jwtData,nil
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

	jwtData, _ := requestJWTDirectAccess(credentials)

	return jwtData
}

// Obtain the certificate authority data
// certificate authority data is stored as a secret named system-tls in verrazzano-system namespace
// Use client cmd to extract the secret
func getCAData() ([]byte, error) {
	var cert []byte
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	namespace := "verrazzano-system"

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil{
		return cert, err
	}
	coreclient, err := corev1client.NewForConfig(restconfig)
	if err != nil{
		return cert, err
	}
	secret, err := coreclient.Secrets(namespace).Get(context.Background(), "system-tls", metav1.GetOptions{})
	if err != nil{
		return cert, err
	}
	cert = (*secret).Data["ca.crt"]
	return cert, nil
}

// Write the kubeconfig object to a file in yaml format
func WriteKubeConfigToDisk(filename string, kubeconfig *clientcmdapi.Config) error {
	err := clientcmd.WriteToFile(*kubeconfig, filename)
	if err != nil {
		return err
	}
	return nil
}

// Helper function to obtain the default kubeconfig location
func getKubeconfigLocation() (string,error) {

	var kubeconfig string
	kubeconfigEnvVar := os.Getenv("KUBECONFIG")

	if len(kubeconfigEnvVar) > 0 {
		// Find using environment variables
		kubeconfig = kubeconfigEnvVar
	} else if home := homedir.HomeDir(); home != "" {
		// Find in the ~/.kube/ directory
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		// give up
		return "", errors.New("Could not find kube config")
	}
	return kubeconfig,nil
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
	// Set the auth code obtained through redirection
	auth_code = m["code"][0]
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

// Returns the oidc client id
func getClientId() string{
	clientId := os.Getenv("VZ_CLIENT_ID")
	// Look for the matching environment variable, return default if not found
	if clientId == "" {
		return "webui"
	} else {
		return clientId
	}
}

// Returns the keycloak base url
func getKeycloakURL() string{
	url := os.Getenv("VZ_KEYCLOAK_URL")
	// Look for the matching environment variable, return default if not found
	if url == "" {
		return "keycloak.default.172.18.0.231.nip.io:443"
	} else {
		return url
	}
}

// Returns the realm name the oidc client is part of
func getVerrazzanoRealm() string{
	realmName := os.Getenv("VZ_REALM")
	// Look for the matching environment variable, return default if not found
	if realmName == "" {
		return "verrazzano-system"
	} else {
		return realmName
	}
}

// Generates the keycloak api url to login
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
func openUrlInBrowser(url string) error{
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
	return err
}

// Gnerates and returns keycloak server api url to get the jwt token
// Return string of the form `https://keycloak.xyz.io:123/auth/realms/verrazzano-system/protocol/openid-connect/token
func generateKeycloakTokenURL() string {
	u := &url.URL{
		Scheme: "https",
		Host:   getKeycloakURL(),
		Path:   "/auth/realms/" + getVerrazzanoRealm() + "/protocol/openid-connect/token",
	}
	return u.String()
}

// Requests the jwt token on our behalf
// Returns the key-value pairs obtained from server in the form of a map
func requestJWT(redirect_uri string, code_verifier string) (map[string]interface{},error) {

	// The response is going to be filled in this
	var jsondata map[string]interface{}

	// Set all the parameters for the POST request
	grantParams := url.Values{}
	grantParams.Set("grant_type", "authorization_code")
	grantParams.Set("client_id", getClientId())
	grantParams.Set("code", auth_code)
	grantParams.Set("redirect_uri", redirect_uri)
	grantParams.Set("code_verifier", code_verifier)
	grantParams.Set("scope","openid")

	// Execute the request
	jsondata, err := executeRequestForJWT(grantParams)
	if err!=nil {
		fmt.Println("Request failed")
		return jsondata, err
	}

	return jsondata,nil
}

// Requests the jwt token  through direct access grant flow.
// Returns the key-value pairs obtained from server in the form of a map
func requestJWTDirectAccess(credentials map[string]string) (map[string]interface{},error) {

	// The response is going to be filled in this
	var jsondata map[string]interface{}

	// Set all the parameters for the POST request
	grantParams := url.Values{}
	grantParams.Set("grant_type", "password")
	grantParams.Set("client_id", getClientId())
	grantParams.Set("username", credentials["username"])
	grantParams.Set("password", credentials["password"])
	grantParams.Set("scope","openid")

	// Execute the request
	jsondata, err := executeRequestForJWT(grantParams)
	if err!=nil {
		fmt.Println("Request failed")
		return jsondata, err
	}

	return jsondata, nil
}

// Creates and executes the POST request
// Returns the key-value pairs obtained from server in the form of a map
func executeRequestForJWT(grantParams url.Values) (map[string]interface{},error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// The response is going to be filled in this
	var jsondata map[string]interface{}

	// Get the keycloak url to obtain tokens
	token_url := generateKeycloakTokenURL()

	// Create new http POST request to obtain token as response
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodPost, token_url, strings.NewReader(grantParams.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err!=nil {
		fmt.Println("Unable to create POST request")
		return jsondata,err
	}

	// Send the request and get response
	response, err := client.Do(request)
	if err!=nil {
		fmt.Println("Error receiving response")
		return jsondata,err
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err!=nil {
		fmt.Println("Unable to read the response body")
		return jsondata,err
	}

	// Convert the response into a map
	json.Unmarshal([]byte(responseBody), &jsondata)

	return jsondata,nil
}