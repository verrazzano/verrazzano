// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak_test

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"os/exec"
	"path"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
	keycloakNamespace string = "keycloak"
)

// KeycloakClients represents an array of clients currently configured in Keycloak
type KeycloakClients []struct {
	ID       string `json:"id"`
	ClientID string `json:"clientId"`
}

type Client struct {
	ID                        string   `json:"id"`
	ClientID                  string   `json:"clientId"`
	SurrogateAuthRequired     bool     `json:"surrogateAuthRequired"`
	Enabled                   bool     `json:"enabled"`
	AlwaysDisplayInConsole    bool     `json:"alwaysDisplayInConsole"`
	ClientAuthenticatorType   string   `json:"clientAuthenticatorType"`
	RedirectUris              []string `json:"redirectUris"`
	WebOrigins                []string `json:"webOrigins"`
	NotBefore                 int      `json:"notBefore"`
	BearerOnly                bool     `json:"bearerOnly"`
	ConsentRequired           bool     `json:"consentRequired"`
	StandardFlowEnabled       bool     `json:"standardFlowEnabled"`
	ImplicitFlowEnabled       bool     `json:"implicitFlowEnabled"`
	DirectAccessGrantsEnabled bool     `json:"directAccessGrantsEnabled"`
	ServiceAccountsEnabled    bool     `json:"serviceAccountsEnabled"`
	PublicClient              bool     `json:"publicClient"`
	FrontchannelLogout        bool     `json:"frontchannelLogout"`
	Protocol                  string   `json:"protocol"`
	Attributes                struct {
		SamlAssertionSignature                string `json:"saml.assertion.signature"`
		SamlForcePostBinding                  string `json:"saml.force.post.binding"`
		SamlMultivaluedRoles                  string `json:"saml.multivalued.roles"`
		SamlEncrypt                           string `json:"saml.encrypt"`
		SamlServerSignature                   string `json:"saml.server.signature"`
		SamlServerSignatureKeyinfoExt         string `json:"saml.server.signature.keyinfo.ext"`
		ExcludeSessionStateFromAuthResponse   string `json:"exclude.session.state.from.auth.response"`
		SamlForceNameIDFormat                 string `json:"saml_force_name_id_format"`
		SamlClientSignature                   string `json:"saml.client.signature"`
		TLSClientCertificateBoundAccessTokens string `json:"tls.client.certificate.bound.access.tokens"`
		SamlAuthnstatement                    string `json:"saml.authnstatement"`
		DisplayOnConsentScreen                string `json:"display.on.consent.screen"`
		PkceCodeChallengeMethod               string `json:"pkce.code.challenge.method"`
		SamlOnetimeuseCondition               string `json:"saml.onetimeuse.condition"`
	} `json:"attributes"`
	AuthenticationFlowBindingOverrides struct {
	} `json:"authenticationFlowBindingOverrides"`
	FullScopeAllowed          bool `json:"fullScopeAllowed"`
	NodeReRegistrationTimeout int  `json:"nodeReRegistrationTimeout"`
	ProtocolMappers           []struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Protocol        string `json:"protocol"`
		ProtocolMapper  string `json:"protocolMapper"`
		ConsentRequired bool   `json:"consentRequired"`
		Config          struct {
			Multivalued        string `json:"multivalued"`
			UserinfoTokenClaim string `json:"userinfo.token.claim"`
			UserAttribute      string `json:"user.attribute"`
			IDTokenClaim       string `json:"id.token.claim"`
			AccessTokenClaim   string `json:"access.token.claim"`
			ClaimName          string `json:"claim.name"`
			JSONTypeLabel      string `json:"jsonType.label"`
		} `json:"config,omitempty"`
	} `json:"protocolMappers"`
	DefaultClientScopes  []string `json:"defaultClientScopes"`
	OptionalClientScopes []string `json:"optionalClientScopes"`
	Access               struct {
		View      bool `json:"view"`
		Configure bool `json:"configure"`
		Manage    bool `json:"manage"`
	} `json:"access"`
}

var volumeClaims map[string]*corev1.PersistentVolumeClaim

var _ = BeforeSuite(func() {
	Eventually(func() (map[string]*corev1.PersistentVolumeClaim, error) {
		var err error
		volumeClaims, err = pkg.GetPersistentVolumes(keycloakNamespace)
		return volumeClaims, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
})

var _ = Describe("Verify Keycloak configuration", func() {
	var _ = Context("Verify password policies", func() {
		isManagedClusterProfile := pkg.IsManagedClusterProfile()
		It("Verify master realm password policy", func() {
			if !isManagedClusterProfile {
				// GIVEN the password policy setup for the master realm during installation
				// WHEN valid and invalid password changes are attempted
				// THEN verify valid passwords are accepted and invalid passwords are rejected.
				Eventually(verifyKeycloakMasterRealmPasswordPolicyIsCorrect, waitTimeout, pollingInterval).Should(BeTrue())
			}
		})
		It("Verify verrazzano-system realm password policy", func() {
			if !isManagedClusterProfile {
				// GIVEN the password policy setup for the verrazzano-system realm during installation
				// WHEN valid and invalid password changes are attempted
				// THEN verify valid passwords are accepted and invalid passwords are rejected.
				Eventually(verifyKeycloakVerrazzanoRealmPasswordPolicyIsCorrect, waitTimeout, pollingInterval).Should(BeTrue())
			}
		})
	})
})

var _ = Describe("Verify MySQL Persistent Volumes based on install profile", func() {
	var _ = Context("Verify Persistent volumes allocated per install profile", func() {

		size := "8Gi" // based on values set in platform-operator/thirdparty/charts/mysql
		kubeconfigPath, _ := k8sutil.GetKubeConfigLocation()
		override, _ := pkg.GetEffectiveKeyCloakPersistenceOverride(kubeconfigPath)
		if override != nil {
			size = override.Spec.Resources.Requests.Storage().String()
		}

		if pkg.IsDevProfile() {
			expectedKeyCloakPVCs := 0
			if override != nil {
				expectedKeyCloakPVCs = 1
			}
			It("Verify persistent volumes in namespace keycloak based on Dev install profile", func() {
				// There is no Persistent Volume for MySQL in a dev install
				Expect(len(volumeClaims)).To(Equal(expectedKeyCloakPVCs))
				if expectedKeyCloakPVCs > 0 {
					assertPersistentVolume("mysql", size)
				}
			})
		} else if pkg.IsManagedClusterProfile() {
			It("Verify namespace keycloak doesn't exist based on Managed Cluster install profile", func() {
				// There is no keycloak namespace in a managed cluster install
				Eventually(func() bool {
					_, err := pkg.GetNamespace(keycloakNamespace)
					return err != nil && errors.IsNotFound(err)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		} else if pkg.IsProdProfile() {
			It("Verify persistent volumes in namespace keycloak based on Prod install profile", func() {
				// 50 GB Persistent Volume create for MySQL in a prod install
				Expect(len(volumeClaims)).To(Equal(1))
				assertPersistentVolume("mysql", size)
			})
		}
	})
})

var _ = Describe("Verify Keycloak URIs", func() {
	var _ = Context("Verify redirect and weborigins URIs", func() {
		isManagedClusterProfile := pkg.IsManagedClusterProfile()
		It("Verify redirect and weborigins URIs", func() {
			if !isManagedClusterProfile {
				// GIVEN installation/upgrade of Keycloak has happened
				// THEN verify that the correct redirect and weborigins URIs are created for verrazzano
				Eventually(verifyKeycloakClientURIs, waitTimeout, pollingInterval).Should(BeTrue())
			}
		})
	})
})

func verifyKeycloakVerrazzanoRealmPasswordPolicyIsCorrect() bool {
	return verifyKeycloakRealmPasswordPolicyIsCorrect("verrazzano-system")
}

func verifyKeycloakMasterRealmPasswordPolicyIsCorrect() bool {
	return verifyKeycloakRealmPasswordPolicyIsCorrect("master")
}

func verifyKeycloakRealmPasswordPolicyIsCorrect(realm string) bool {
	kc, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		fmt.Printf("Failed to create Keycloak REST client: %v\n", err)
		return false
	}

	var realmData map[string]interface{}
	realmData, err = kc.GetRealm(realm)
	if err != nil {
		fmt.Printf("Failed to get realm %s\n", realm)
		return false
	}
	if realmData["passwordPolicy"] == nil {
		fmt.Printf("Failed to find password policy for realm: %s\n", realm)
		return false
	}
	policy := realmData["passwordPolicy"].(string)
	if len(policy) == 0 || !strings.Contains(policy, "length") {
		fmt.Printf("Failed to find password policy for realm: %s\n", realm)
		return false
	}

	salt := time.Now().Format("20060102150405.000000000")
	userName := fmt.Sprintf("test-user-%s", salt)
	firstName := fmt.Sprintf("test-first-%s", salt)
	lastName := fmt.Sprintf("test-last-%s", salt)
	validPassword := fmt.Sprintf("test-password-12-!@-AB-%s", salt)
	userURL, err := kc.CreateUser(realm, userName, firstName, lastName, validPassword)
	if err != nil {
		fmt.Printf("Failed to create user %s/%s: %v\n", realm, userName, err)
		return false
	}
	userID := path.Base(userURL)
	defer func() {
		err = kc.DeleteUser(realm, userID)
		if err == nil {
			fmt.Printf("Failed to delete user %s/%s: %v\n", realm, userID, err)
		}
	}()
	err = kc.SetPassword(realm, userID, "invalid")
	if err == nil {
		fmt.Printf("Should not have been able to set password for %s/%s\n", realm, userID)
		return false
	}
	newValidPassword := fmt.Sprintf("test-new-password-12-!@-AB-%s", salt)
	err = kc.SetPassword(realm, userID, newValidPassword)
	if err != nil {
		fmt.Printf("Failed to set password for %s/%s: %v\n", realm, userID, err)
		return false
	}
	return true
}

func verifyKeycloakClientURIs() bool {
	var keycloakClients KeycloakClients
	var keycloakClient Client

	// Get the Keycloak admin password
	secret, err := pkg.GetSecret("keycloak", "keycloak-http")
	if err != nil {
		fmt.Printf("Failed to get KeyCloak secret: %s\n", err)
		return false
	}
	pw := secret.Data["password"]
	keycloakpw := string(pw)
	if keycloakpw == "" {
		fmt.Print("Invalid Keycloak password. Empty String returned")
		return false
	}

	// Login to Keycloak
	cmd := exec.Command("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--",
		"/opt/jboss/keycloak/bin/kcadm.sh", "config", "credentials", "--server", "http://localhost:8080/auth", "--realm", "master", "--user", "keycloakadmin", "--password", keycloakpw)
	_, err = cmd.Output()
	if err != nil {
		fmt.Printf("Error logging into Keycloak: %s\n", err)
		return false
	}

	// Get the Client ID JSON array
	cmd = exec.Command("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", "clients", "-r", "verrazzano-system", "--fields", "id,clientId")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error retrieving ID for client ID, zero length: %s\n", err)
		return false
	}
	if len(string(out)) == 0 {
		fmt.Print("Error retrieving Clients JSON from Keycloak, zero length, zero length\n")
		return false
	}
	json.Unmarshal([]byte(out), &keycloakClients)

	// Extract the id associated with ClientID verrazzano-pkce
	var id = ""
	for _, client := range keycloakClients {
		if client.ClientID == "verrazzano-pkce" {
			id = client.ID
			fmt.Printf("Keycloak Clients ID found = %s\n", id)
		}
	}
	if id == "" {
		fmt.Print("Error retrieving ID for Keycloak user, zero length\n")
		return false
	}

	// Get the client Info
	client := "clients/" + id
	cmd = exec.Command("kubectl", "exec", "keycloak-0", "-n", "keycloak", "-c", "keycloak", "--", "/opt/jboss/keycloak/bin/kcadm.sh", "get", client, "-r", "verrazzano-system")
	out, err = cmd.Output()
	if err != nil {
		fmt.Printf("Error retrieving clientID JSON: %s\n", err)
		return false
	}
	if len(string(out)) == 0 {
		fmt.Print("Error retrieving Client JSON from Keycloak, zero length, zero length\n")
		return false
	}
	json.Unmarshal([]byte(out), &keycloakClient)

	// Verify Correct number of RedirectURIs
	if len(keycloakClient.RedirectUris) != 12 {
		fmt.Printf("Incorrect Number of Redirect URIs returned for client %+v\n", keycloakClient.RedirectUris)
		return false
	}

	// Verify Correct number of WebOrigins
	if len(keycloakClient.WebOrigins) != 6 {
		fmt.Printf("Incorrect Number of WebOrigins returned for client %+v\n", keycloakClient.WebOrigins)
		return false
	}

	// Verify Num URIs per product endpoint
	// Kiali
	if !verifyURIs(keycloakClient.RedirectUris, "kiali.vmi.system.default", 2) {
		fmt.Printf("Expected 2 Kiali redirect URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	if !verifyURIs(keycloakClient.WebOrigins, "kiali.vmi.system.default", 1) {
		fmt.Printf("Expected 1 Kiali weborigin URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	// Prometheus
	if !verifyURIs(keycloakClient.RedirectUris, "prometheus.vmi.system.default", 2) {
		fmt.Printf("Expected 2 Prometheus redirect URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	if !verifyURIs(keycloakClient.WebOrigins, "prometheus.vmi.system.default", 1) {
		fmt.Printf("Expected 1 Prometheus weborigin URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	// Grafana
	if !verifyURIs(keycloakClient.RedirectUris, "grafana.vmi.system.default", 2) {
		fmt.Printf("Expected 2 Grafana redirect URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	if !verifyURIs(keycloakClient.WebOrigins, "grafana.vmi.system.default", 1) {
		fmt.Printf("Expected 1 Grafana weborigin URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	// Elasticsearch
	if !verifyURIs(keycloakClient.RedirectUris, "elasticsearch.vmi.system.default", 2) {
		fmt.Printf("Expected 2 Elasticsearch redirect URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	if !verifyURIs(keycloakClient.WebOrigins, "elasticsearch.vmi.system.default", 1) {
		fmt.Printf("Expected 1 Elasticsearch weborigin URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	// Kibana
	if !verifyURIs(keycloakClient.RedirectUris, "kibana.vmi.system.default", 2) {
		fmt.Printf("Expected 2 Kibana redirect URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	if !verifyURIs(keycloakClient.WebOrigins, "kibana.vmi.system.default", 1) {
		fmt.Printf("Expected 1 Kibana weborigin URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	// Verrazzano
	if !verifyURIs(keycloakClient.RedirectUris, "verrazzano.default", 2) {
		fmt.Printf("Expected 2 Verrazzano redirect URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	if !verifyURIs(keycloakClient.WebOrigins, "verrazzano.default", 1) {
		fmt.Printf("Expected 1 Verrazzano weborigin URIs. Found %+v\n", keycloakClient.RedirectUris)
		return false
	}

	return true
}

func assertPersistentVolume(key string, size string) {
	Expect(volumeClaims).To(HaveKey(key))
	pvc := volumeClaims[key]
	Expect(pvc.Spec.Resources.Requests.Storage().String()).To(Equal(size))
}

func verifyURIs(uriArray []string, name string, numToFind int) bool {
	ctr := 0
	for _, uri := range uriArray {
		if strings.Contains(uri, name) {
			ctr++
		}
	}
	return ctr != numToFind
}
