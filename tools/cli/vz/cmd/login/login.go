// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package login

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Struct to store Login-command related data. eg.flags,streams,args..
type LoginOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

// Creates a LoginOptions struct to run the login command
func NewLoginOptions(streams genericclioptions.IOStreams) *LoginOptions {
	return &LoginOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// root.go returns this function for login
// Calls the login function to complete login
func NewCmdLogin(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewLoginOptions(streams)
	cmd := &cobra.Command{
		Use:   "login verrazzanoAPIURL",
		Short: "vz login verrazzanoAPIURL",
		Long:  "vz login verrazzanoAPIURL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := login(streams, args, kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func login(streams genericclioptions.IOStreams, args []string, kubernetesInterface helpers.Kubernetes) error {
	// Check if the user is already logged out
	loggedIn,  err := helpers.LoggedIn()
	if err!=nil {
		return err
	}
	if loggedIn {
		_ , err := fmt.Fprintln(streams.Out, "Already Logged in")
		return err
	}

	// Extract parameters from args
	verrazzanoAPIURL := args[0]

	// Obtain the certificate authority data in the form of byte stream
	caData, err := extractCAData(kubernetesInterface)
	if err != nil {
		return err
	}

	// Follow the authorization grant flow to get the json response
	jwtData, err := authFlowLogin(caData)
	if err != nil {
		return err
	}

	// Add the verrazzano cluser into config
	err = helpers.SetClusterInKubeConfig(helpers.Verrazzano,
		verrazzanoAPIURL,
		caData,
	)
	if err!=nil {
		return err
	}

	// Add the logged-in user with nickname verrazzano
	accessToken,ok := jwtData["access_token"]
	if !ok {
		return errors.New("Access Token not found in jwtData")
	}
	refreshToken,ok := jwtData["refresh_token"]
	if !ok {
		return errors.New("Refresh Token not found in jwtData")
	}
	accessTokenExpTime,ok := jwtData["expires_in"]
	if !ok {
		return errors.New("Access Token Expiration Time not found in jwtData")
	}
	refreshTokenExpTime,ok := jwtData["refresh_expires_in"]
	if !ok {
		return errors.New("Refresh Token Expiration Time not found in jwtData")
	}



	err = helpers.SetUserInKubeConfig(helpers.Verrazzano,
		helpers.AuthDetails{
			int64(accessTokenExpTime.(float64)) + time.Now().Unix() - helpers.BufferTime,
			int64(refreshTokenExpTime.(float64)) + time.Now().Unix() - helpers.BufferTime,
			accessToken.(string),
			refreshToken.(string),
		},
	)
	if err!=nil {
		return err
	}

	// Add new context with name verrazzano@oldcontext
	// This context uses verrazzano cluster and logged-in user
	// We need oldcontext to fall back after logout
	currentContext, err := helpers.GetCurrentContextFromKubeConfig()
	if err!=nil {
		return err
	}
	err = helpers.SetContextInKubeConfig(fmt.Sprintf("%v@%v",helpers.Verrazzano,currentContext),
		helpers.Verrazzano,
		helpers.Verrazzano,
	)
	if err!=nil {
		return err
	}

	// Switch over to new context
	err = helpers.SetCurrentContextInKubeConfig(fmt.Sprintf("%v@%v",helpers.Verrazzano,currentContext))
	if err!=nil {
		return err
	}

	_, err = fmt.Fprintln(streams.Out, "Login successful!")
	return err
}

var authCode = "" //	Http handle fills this after keycloak authentication
var serverErr error = nil	// Http handle fills this
var stateFromKeycloak = ""  // Http handle fills the state obtained through redirection

// A function to put together all the requirements of authorization grant flow
// Returns the final jwt token as a map
func authFlowLogin(caData []byte) (map[string]interface{}, error) {

	var jwtData map[string]interface{}
	// Obtain a available port in non-well known port range
	listener,err := getFreePort()
	if err!=nil {
		return jwtData,err
	}

	// Generate random code verifier and code challenge pair
	codeVerifier, codeChallenge := helpers.GenerateRandomCodePair()
	state := helpers.GenerateRandomState()

	// Generate the redirect uri using the port obtained
	redirectURI := helpers.GenerateRedirectURI(listener)

	// Generate the login keycloak url by passing the required url parameters
	loginURL := helpers.GenerateKeycloakAPIURL(codeChallenge,
		redirectURI,
		state,
	)

	// Busy wait when the authorization code is still not filled by http handle
	// Close the listener once we obtain it
	go func() {
		for authCode == "" && serverErr==nil {

		}
		listener.Close()
	}()

	// Make sure the go routine is running
	time.Sleep(time.Second)

	// Open the generated keycloak login url in the browser
	err = helpers.OpenURLInBrowser(loginURL)
	if err != nil {
		return jwtData, err
	}

	// Set the handle function and start the http server
	http.HandleFunc("/",
		handle,
	)
	http.Serve(listener,
		nil,
	)

	if stateFromKeycloak!=state {
		return jwtData, errors.New("State mismatch")
	}

	if serverErr != nil {
		return jwtData, err
	}

	// Obtain the JWT token by exchanging it with authCode
	jwtData, err = requestJWT(redirectURI,
		codeVerifier,
		caData,
	)
	if err != nil {
		return jwtData, err
	}

	return jwtData, nil
}

// Obtain the certificate authority data
// certificate authority data is stored as a secret named system-tls in verrazzano-system namespace
// Use client cmd to extract the secret
func extractCAData(kubernetesInterface helpers.Kubernetes) ([]byte, error) {
	var cert []byte

	kclientset := kubernetesInterface.NewClientSet()
	secret, err := kclientset.CoreV1().Secrets("verrazzano-system").Get(context.Background(),
		"system-tls",
		metav1.GetOptions{},
	)

	if err != nil {
		return cert, err
	}
	cert = (*secret).Data["ca.crt"]
	return cert, nil
}

// Handler function for http server
// The server page's html,js,etc code is embedded here.
func handle(w http.ResponseWriter, r *http.Request) {
	u, serverErr := url.Parse(r.URL.String())
	if serverErr == nil {
		m, serverErr := url.ParseQuery(u.RawQuery)
		if serverErr == nil {
			// Set the auth code obtained through redirection
			authCode = m["code"][0]
			stateFromKeycloak = m["state"][0]
			fmt.Fprintln(w, "<p>You can close this tab now</p>")
		}
	}
	if serverErr != nil{
		fmt.Fprintln(w, "<p>Authentication failed, Please try again</p>")
	}
}

// Fetch an available port
// Return in the form of listener
func getFreePort() (net.Listener,error) {
	listener, err := net.Listen("tcp", ":0")
	return listener,err
}

// Requests the jwt token on our behalf
// Returns the key-value pairs obtained from server in the form of a map
func requestJWT(redirectURI string, codeVerifier string, caData []byte) (map[string]interface{}, error) {

	// The response is going to be filled in this
	var jwtData map[string]interface{}

	// Set all the parameters for the POST request
	grantParams := url.Values{}
	grantParams.Set("grant_type", "authorization_code")
	grantParams.Set("client_id", helpers.GetClientID())
	grantParams.Set("code", authCode)
	grantParams.Set("redirect_uri", redirectURI)
	grantParams.Set("code_verifier", codeVerifier)
	grantParams.Set("scope", "openid")

	// Execute the request
	jwtData, err := executeRequestForJWT(grantParams, caData)
	if err != nil {
		return jwtData, err
	}

	return jwtData, nil
}

// Creates and executes the POST request
// Returns the key-value pairs obtained from server in the form of a map
func executeRequestForJWT(grantParams url.Values, caData []byte) (map[string]interface{}, error) {

	// The response is going to be filled in this
	var jwtData map[string]interface{}

	// Get the keycloak url to obtain tokens
	tokenURL := helpers.GenerateKeycloakTokenURL()

	// Create new http POST request to obtain token as response
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caData)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	request, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(grantParams.Encode()))
	if err != nil {
		return jwtData, err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Send the request and get response
	response, err := client.Do(request)
	if err != nil {
		return jwtData, err
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return jwtData, err
	}

	// Convert the response into a map
	err = json.Unmarshal([]byte(responseBody), &jwtData)

	return jwtData, err
}

// Function that refreshes the access token if required when logged in
func RefreshToken() error {

	// Nothing to do when the user is not logged in
	loggedOut,err := helpers.LoggedOut()
	if err!=nil {
		return err
	}
	if loggedOut {
		return nil
	}

	authDetails,err := helpers.GetAuthDetails()
	if err!=nil {
		return err
	}

	// If the access token is still valid
	if time.Now().Unix()+helpers.BufferTime < authDetails.AccessTokenExpTime {
		return nil
	}

	// If the refresh token has expired, delete all auth data
	if time.Now().Unix()-helpers.BufferTime > authDetails.RefreshTokenExpTime {
		err := helpers.RemoveAllAuthData()
		return err
	}

	grantParams := url.Values{}
	grantParams.Set("grant_type", "refresh_token")
	grantParams.Set("client_id", helpers.GetClientID())
	grantParams.Set("refresh_token", authDetails.RefreshToken)
	grantParams.Set("redirect_uri", "http://localhost:8080")
	grantParams.Set("scope", "openid")

	caDataEncoded,err  := helpers.GetCAData()
	if err!=nil {
		return err
	}
	caData, err := base64.StdEncoding.DecodeString(caDataEncoded)
	if err != nil {
		return err
	}

	// Execute the request
	jwtData, err := executeRequestForJWT(grantParams, []byte(caData))
	if err != nil {
		return err
	}

	// Set new auth details in kubeconfig
	accessToken,ok := jwtData["access_token"]
	if !ok {
		return errors.New("Access Token not found in jwtData")
	}
	refreshToken,ok := jwtData["refresh_token"]
	if !ok {
		return errors.New("Refresh Token not found in jwtData")
	}
	accessTokenExpTime,ok := jwtData["expires_in"]
	if !ok {
		return errors.New("Access Token Expiration Time not found in jwtData")
	}
	refreshTokenExpTime,ok := jwtData["refresh_expires_in"]
	if !ok {
		return errors.New("Refresh Token Expiration Time not found in jwtData")
	}

	err = helpers.SetUserInKubeConfig(helpers.Verrazzano,
		helpers.AuthDetails{
			int64(accessTokenExpTime.(float64)) + time.Now().Unix() - helpers.BufferTime,
			int64(refreshTokenExpTime.(float64)) + time.Now().Unix() - helpers.BufferTime,
			accessToken.(string),
			refreshToken.(string),
		},
	)

	return err
}
