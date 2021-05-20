// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"bytes"
	"encoding/json"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic/testdata"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/jsonpath"
)

// TestJsonPathHealth tests the parsing of a JSON string for the server health information
// GIVEN a WebLogic domain JSON string
// WHEN the string is parsed
// THEN the resulting string has the health information of all the servers
func TestJsonPathHealth(t *testing.T) {
	assert := asserts.New(t)

	var domain interface{}
	err := json.Unmarshal([]byte(testdata.Domain), &domain)
	if err != nil {
		t.Error(err)
	}
	const template = `{range .status.servers[*]}{.serverName}:{.health.overallHealth},{end}`
	j := jsonpath.New("test")
	err = j.Parse(template)
	assert.NoError(err)

	buf := new(bytes.Buffer)
	err = j.Execute(buf, domain)
	assert.NoError(err)
	s := buf.String()
	assert.Equal(s, "AdminServer:ok,ManagedServer:bad,", "jsonpath returned wrong value")
}

// TestGetHealthOfServers tests the extraction of WebLogic server health information from an unstructured domain
// GIVEN a WebLogic domain as an unstructured object
// WHEN GetHealthOfServers is called
// THEN the resulting health array has the health information of all the servers
func TestGetHealthOfServers(t *testing.T) {
	assert := asserts.New(t)

	uDomain := unstructured.Unstructured{}
	err := uDomain.UnmarshalJSON([]byte(testdata.Domain))
	assert.NoError(err)
	healths, err := GetHealthOfServers(&uDomain)
	assert.NoError(err)

	assert.NoError(err)
	assert.Len(healths, 2, "wrong number of server health items returned")
	assert.Equal("AdminServer", healths[0].ServerName)
	assert.Equal("ok", healths[0].Health)
	assert.Equal("ManagedServer", healths[1].ServerName)
	assert.Equal("bad", healths[1].Health)
}
