// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqlcheck

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStart(t *testing.T) {
	p := newTestMySQLCheck()
	assert.Nil(t, p.shutdown)
	p.Start()
	assert.NotNil(t, p.shutdown)
	p.Start()
	assert.NotNil(t, p.shutdown)
	p.Pause()
	assert.Nil(t, p.shutdown)
	p.Pause()
	assert.Nil(t, p.shutdown)
}

func newTestMySQLCheck() *MySQLChecker {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	mysqlChecker, _ := NewMySQLChecker(c, 1*time.Second, 1*time.Second)
	return mysqlChecker
}
