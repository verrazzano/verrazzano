// Copyright (c) 2021, Oracle and/or its affiliates.
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
	isLoggedIn, err := helpers.LoggedIn()
	if err != nil {
		return err
	}
	if isLoggedIn {
		_, err := fmt.Fprintln(streams.Out, "Already Logged in")
		return err
	}

	// Extract parameters from args
	verrazzanoAPIURL := args[0]

	// Obtain the certificate authority data in the form of byte stream
	caData, err := extractCAData(kubernetesInterface)
	if err != nil {
		if err.Error() == "ca.crt not found" {
			// Assume well-known certificate authority data
		} else {
			return err
		}
	}

	// Follow the authorization grant flow to get the json response
	jwtData, err := authFlowLogin(caData)
	if err != nil {
		return err
	}

	// Add the verrazzano cluster into config
	err = helpers.SetClusterInKubeConfig(helpers.NickNameInKubeConfig,
		verrazzanoAPIURL,
		caData,
	)
	if err != nil {
		return err
	}

	// Add the logged-in user with nickname verrazzano
	if !checkNonEmptyJWTData(jwtData) {
		return errors.New("Fields missing in jwtData")
	}

	err = helpers.SetUserInKubeConfig(helpers.NickNameInKubeConfig,
		helpers.AuthDetails{
			AccessTokenExpTime:  int64(jwtData["expires_in"].(float64)) + time.Now().Unix() - helpers.BufferTime,
			RefreshTokenExpTime: int64(jwtData["refresh_expires_in"].(float64)) + time.Now().Unix() - helpers.BufferTime,
			AccessToken:         jwtData["access_token"].(string),
			RefreshToken:        jwtData["refresh_token"].(string),
		},
	)
	if err != nil {
		return err
	}

	// Add new context with name verrazzano@oldcontext
	// This context uses verrazzano cluster and logged-in user
	// We need oldcontext to fall back after logout
	currentContext, err := helpers.GetCurrentContextFromKubeConfig()
	if err != nil {
		return err
	}
	err = helpers.SetContextInKubeConfig(fmt.Sprintf("%v@%v", helpers.NickNameInKubeConfig, currentContext),
		helpers.NickNameInKubeConfig,
		helpers.NickNameInKubeConfig,
	)
	if err != nil {
		return err
	}

	// Switch over to new context
	err = helpers.SetCurrentContextInKubeConfig(fmt.Sprintf("%v@%v", helpers.NickNameInKubeConfig, currentContext))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(streams.Out, "Login successful!")
	return err
}

// A function to put together all the requirements of authorization grant flow
// Returns the final jwt token as a map
func authFlowLogin(caData []byte) (map[string]interface{}, error) {

	var jwtData map[string]interface{}
	// Obtain a available port in non-well known port range
	listener, err := getFreePort()
	if err != nil {
		return jwtData, err
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

	// Make sure the go routine is running
	time.Sleep(time.Second)

	// Open the generated keycloak login url in the browser
	err = helpers.OpenURLInBrowser(loginURL)
	if err != nil {
		return jwtData, err
	}

	urlParamChannel := make(chan keycloakRedirectionURLParams)

	// Set the handle function and start the http server
	go func() {
		http.HandleFunc("/",
			keycloakRedirectionHandle(urlParamChannel),
		)
		http.Serve(listener,
			nil,
		)
	}()

	keycloakRedirectionParamsStruct := <-urlParamChannel
	authCode := keycloakRedirectionParamsStruct.authCode
	stateFromKeycloak := keycloakRedirectionParamsStruct.state
	err = keycloakRedirectionParamsStruct.err

	if err != nil {
		return jwtData, err
	}

	if stateFromKeycloak != state {
		return jwtData, errors.New("State mismatch")
	}

	// Obtain the JWT token by exchanging it with authCode
	jwtData, err = requestJWT(redirectURI,
		codeVerifier,
		caData,
		authCode,
	)
	if err != nil {
		return jwtData, err
	}

	return jwtData, nil
}

func checkNonEmptyJWTData(jwtData map[string]interface{}) bool {
	_, ok := jwtData["access_token"]
	if !ok {
		return false
	}
	_, ok = jwtData["refresh_token"]
	if !ok {
		return false
	}
	_, ok = jwtData["expires_in"]
	if !ok {
		return false
	}
	_, ok = jwtData["refresh_expires_in"]
	return ok
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
	cert, ok := (*secret).Data["ca.crt"]
	if !ok {
		return cert, errors.New("ca.crt not found")
	}
	return cert, nil
}

type keycloakRedirectionURLParams struct {
	authCode string
	state    string
	err      error
}

// Handler function for http server
// The server page's html,js,etc code is embedded here.
func keycloakRedirectionHandle(cliChannel chan keycloakRedirectionURLParams) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(r.URL.String())
		if err == nil {
			m, err := url.ParseQuery(u.RawQuery)
			if err == nil {
				// Set the auth code obtained through redirection
				authCode := m["code"][0]
				stateFromKeycloak := m["state"][0]
				cliChannel <- keycloakRedirectionURLParams{
					authCode: authCode,
					state:    stateFromKeycloak,
					err:      nil,
				}
				fmt.Fprintln(w, "<p>You can close this tab now</p>")
			}
		}
		if err != nil {
			cliChannel <- keycloakRedirectionURLParams{
				authCode: "",
				state:    "",
				err:      err,
			}
			fmt.Fprintln(w, "<p>Authentication failed, Please try again</p>")
		}
	}
}

// Fetch an available port
// Return in the form of listener
func getFreePort() (net.Listener, error) {
	listener, err := net.Listen("tcp", ":0")
	return listener, err
}

// Requests the jwt token on our behalf
// Returns the key-value pairs obtained from server in the form of a map
func requestJWT(redirectURI string, codeVerifier string, caData []byte, authCode string) (map[string]interface{}, error) {

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

	var jwtData map[string]interface{}

	// Get the keycloak url to obtain tokens
	tokenURL := helpers.GenerateKeycloakTokenURL()

	// Create new http POST request to obtain token as response
	var client *http.Client

	// When the caData is empty, assume trusted certificate data authority
	if len(caData) == 0 {
		client = &http.Client{}
	} else {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caData)

		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}
	}

	request, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(grantParams.Encode()))
	if err != nil {
		return jwtData, err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	var response *http.Response
	// Attempting request a maximum of maxAttempts times
	maxAttempts := 5
	// Waiting for a interval of sleepTime seconds before next attempt
	sleepTime := time.Duration(5)
	// Send the request and get response
	for count := 0; count < maxAttempts; count++ {
		response, err = client.Do(request)
		if err == nil {
			break
		}
		time.Sleep(sleepTime * time.Second)
	}
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
	isLoggedOut, err := helpers.LoggedOut()
	if err != nil {
		return err
	}
	if isLoggedOut {
		return nil
	}

	authDetails, err := helpers.GetAuthDetails(helpers.NickNameInKubeConfig)
	if err != nil {
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

	caDataEncoded, err := helpers.GetCAData(helpers.NickNameInKubeConfig)
	if err != nil {
		return err
	}
	var caData []byte
	if len(caDataEncoded) != 0 {
		caData, err = base64.StdEncoding.DecodeString(caDataEncoded)
		if err != nil {
			return err
		}
	}

	// Execute the request
	jwtData, err := executeRequestForJWT(grantParams, []byte(caData))
	if err != nil {
		return err
	}

	// Set new auth details in kubeconfig
	if !checkNonEmptyJWTData(jwtData) {
		return errors.New("Fields missing in jwtData")
	}

	err = helpers.SetUserInKubeConfig(helpers.NickNameInKubeConfig,
		helpers.AuthDetails{
			AccessTokenExpTime:  int64(jwtData["expires_in"].(float64)) + time.Now().Unix() - helpers.BufferTime,
			RefreshTokenExpTime: int64(jwtData["refresh_expires_in"].(float64)) + time.Now().Unix() - helpers.BufferTime,
			AccessToken:         jwtData["access_token"].(string),
			RefreshToken:        jwtData["refresh_token"].(string),
		},
	)

	return err
}
