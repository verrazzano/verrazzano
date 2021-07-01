// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package login

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"net/http"
	"net/url"
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

func NewCmdLogin(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewLoginOptions(streams)
	cmd := &cobra.Command{
		Use:   "login api_url",
		Short: "vz login",
		Long:  "vz_login",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := login(args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func login(args []string, kubernetesInterface helpers.Kubernetes) error {

	var vz_api_url string

	// Extract parameters from args
	for index, element := range args {
		if index == 0 {
			vz_api_url = element
			continue
		}
	}

	// Obtain the certificate authority data in the form of byte stream
	caData, err := getCAData( kubernetesInterface)
	if err != nil {
		fmt.Println("Unable to obtain certificate authority data")
		fmt.Println("Make sure you are in the right context")
		return err
	}

	// Follow the authorization grant flow to get the json response
	jwtData, err := authFlowLogin(caData)
	if err!=nil {
		fmt.Println("Grant flow failed")
		return err
	}


	// Obtain the default kubeconfig's location
	kubeConfigLoc,err := helpers.GetKubeconfigLocation()
	if err!=nil {
		return err
	}

	// Load the default kubeconfig's configuration into clientcmdapi object
	mykubeConfig, _ := clientcmd.LoadFromFile(kubeConfigLoc)

	if err!=nil {
		fmt.Println("Unable to load kubeconfig, check permissions")
		return err
	}

	// Add the verrazzano cluser into config
	helpers.SetCluster(mykubeConfig,
			  "verrazzano",
			   vz_api_url,
			   caData,
			)

	// Add the logged-in user with nickname verrazzano
	helpers.SetUser(mykubeConfig,
			 "verrazzano",
			fmt.Sprintf("%v",jwtData["access_token"]),
			)

	// Add new context with name verrazzano@oldcontext
	// This context uses verrazzano cluster and logged-in user
	// We need oldcontext to fall back after logout
	helpers.SetContext(mykubeConfig,
			  "verrazzano" + "@" + mykubeConfig.CurrentContext,
			  "verrazzano",
			  "verrazzano",
			)

	// Switch over to new context
	helpers.SetCurrentContext(mykubeConfig,
				  "verrazzano"+"@"+mykubeConfig.CurrentContext,
				)

	// Write the new configuration into the default kubeconfig file
	err = clientcmd.WriteToFile(*mykubeConfig,
				    kubeConfigLoc,
				   )
	if err!=nil {
		fmt.Println("Unable to write the new kubconfig to disk")
		return err
	}
	fmt.Println("Login successful!")
	return nil
}

var auth_code = "" //	Http handle fills this after keycloak authentication

// A function to put together all the requirements of authorization grant flow
// Returns the final jwt token as a map
func authFlowLogin(caData []byte) (map[string]interface{}, error) {

	// Obtain a available port in non-well known port range
	listener := getFreePort()

	// Generate random code verifier and code challenge pair
	code_verifier, code_challenge := helpers.GenerateRandomCodePair()

	// Generate the redirect uri using the port obtained
	redirect_uri := helpers.GenerateRedirectURI(listener)

	// Generate the login keycloak url by passing the required url parameters
	login_url := helpers.GenerateKeycloakAPIURL(code_challenge,
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
	err := helpers.OpenUrlInBrowser(login_url)
	if err != nil {
		fmt.Println("Unable to open browser")
	}

	// Set the handle function and start the http server
	http.HandleFunc("/",
			handle,
		       )
	http.Serve(listener,
		   nil,
		  )

	// Obtain the JWT token by exchanging it with auth_code
	jwtData, err := requestJWT(redirect_uri,
				  code_verifier,
				  caData,
				  )
	if err != nil {
		fmt.Println("Unable to obtain the JWT token")
		return jwtData, err
	}
	return jwtData, nil
}

// Obtain the certificate authority data
// certificate authority data is stored as a secret named system-tls in verrazzano-system namespace
// Use client cmd to extract the secret
func getCAData(kubernetesInterface helpers.Kubernetes) ([]byte, error) {
	var cert []byte

	kclientset := kubernetesInterface.NewClientSet()
	secret, err := kclientset.CoreV1().Secrets("verrazzano-system").Get(context.Background(),
									   "system-tls",
									    metav1.GetOptions{},
									)

	if err != nil{
		return cert, err
	}
	cert = (*secret).Data["ca.crt"]
	return cert, nil
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

// Requests the jwt token on our behalf
// Returns the key-value pairs obtained from server in the form of a map
func requestJWT(redirect_uri string, code_verifier string, caData []byte) (map[string]interface{}, error) {

	// The response is going to be filled in this
	var jsondata map[string]interface{}

	// Set all the parameters for the POST request
	grantParams := url.Values{}
	grantParams.Set("grant_type", "authorization_code")
	grantParams.Set("client_id", helpers.GetClientId())
	grantParams.Set("code", auth_code)
	grantParams.Set("redirect_uri", redirect_uri)
	grantParams.Set("code_verifier", code_verifier)
	grantParams.Set("scope", "openid")

	// Execute the request
	jsondata, err := executeRequestForJWT(grantParams, caData)
	if err != nil {
		fmt.Println("Request failed")
		return jsondata, err
	}

	return jsondata, nil
}

// Creates and executes the POST request
// Returns the key-value pairs obtained from server in the form of a map
func executeRequestForJWT(grantParams url.Values, caData []byte) (map[string]interface{}, error) {

	// The response is going to be filled in this
	var jsondata map[string]interface{}

	// Get the keycloak url to obtain tokens
	token_url := helpers.GenerateKeycloakTokenURL()

	// Create new http POST request to obtain token as response
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caData)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
			},
		},
	}

	request, err := http.NewRequest(http.MethodPost, token_url, strings.NewReader(grantParams.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		fmt.Println("Unable to create POST request")
		return jsondata, err
	}

	// Send the request and get response
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Error receiving response")
		return jsondata, err
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Unable to read the response body")
		return jsondata, err
	}

	// Convert the response into a map
	json.Unmarshal([]byte(responseBody), &jsondata)

	return jsondata, nil
}
