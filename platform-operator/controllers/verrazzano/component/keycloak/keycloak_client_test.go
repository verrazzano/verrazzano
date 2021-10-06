package keycloak

import (
	"testing"
)

var keycloakUrl = "https://keycloak.default.172.18.255.101.nip.io"

// TestInitKeycloakClient tests the keycloak client init method
func TestInitKeycloakClient(t *testing.T) {
	client := KeycloakClient{}

    tokenEndpoint := keycloakUrl + "/auth/realms/master/protocol/openid-connect/token"

	client.Init(tokenEndpoint, "keycloakadmin", "5X3Gw3cpA8WrKlBc")
}
