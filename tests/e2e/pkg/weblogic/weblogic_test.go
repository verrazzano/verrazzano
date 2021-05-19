// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic/testdata"
	"k8s.io/client-go/util/jsonpath"
)

//func TestGetDomains(t *testing.T) {
//	assert := asserts.New(t)
//
//	domain, err := GetDomain("todo-list", "todo-domain")
//	assert.NoError(err)
//	assert.NotNil(domain)
//
//	sh, err := GetHealthOfServers(domain)
//	assert.NoError(err)
//	assert.Equal("AdminServer", sh[0].ServerName)
//	assert.Equal("ok", sh[0].Health)
//
//	sh, err = GetHealthOfServersUsingStruct(domain)
//	assert.NoError(err)
//	assert.Equal("AdminServer", sh[0].ServerName)
//	assert.Equal("ok", sh[0].Health)
//}
//
//func TestListDomains(t *testing.T) {
//	assert := asserts.New(t)
//
//	domains, err := ListDomains("todo-list")
//	assert.NoError(err)
//	assert.NotNil(domains)
//
//	sh, err := GetHealthOfServers(&domains.Items[0])
//	assert.NoError(err)
//	assert.Equal("AdminServer", sh[0].ServerName)
//	assert.Equal("ok", sh[0].Health)
//
//	sh, err = GetHealthOfServersUsingStruct(&domains.Items[0])
//	assert.NoError(err)
//	assert.Equal("AdminServer", sh[0].ServerName)
//	assert.Equal("ok", sh[0].Health)
//}

func TestJsonPathKind(t *testing.T) {
	assert := asserts.New(t)

	var domain interface{}
	err := json.Unmarshal([]byte(testdata.Domain), &domain)
	if err != nil {
		t.Error(err)
	}
	const template = "{.kind}"
	j := jsonpath.New("test")
	err = j.Parse(template)
	assert.NoError(err)

	buf := new(bytes.Buffer)
	err = j.Execute(buf, domain)
	assert.NoError(err)

	s := buf.String()
	fmt.Printf("kind: %s\n", s)
}

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
	fmt.Printf("servers: %s\n", s)
}
