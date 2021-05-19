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

func TestListDomains(t *testing.T) {
	assert := asserts.New(t)

	res, err := ListDomains("todo-list", "/Users/pmackin/.kube/config-oke")
	assert.NoError(err)
	assert.NotNil(res)
	assert.NoError(checkServerStatus(&res.Items[0]))
}

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
	const template = `{range .status.servers[*]}{.health.overallHealth} {end}`
	j := jsonpath.New("test")
	err = j.Parse(template)
	assert.NoError(err)

	buf := new(bytes.Buffer)
	err = j.Execute(buf, domain)
	assert.NoError(err)

	s := buf.String()
	fmt.Printf("health: %s\n", s)
}
