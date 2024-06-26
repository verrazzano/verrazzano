// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/authproxy/internal/testutil/file"
	"go.uber.org/zap"
)

// TestInitConfiguration tests the InitConfiguration function
func TestInitConfiguration(t *testing.T) {
	const testServiceURL = "provider.namespace.svc.cluster.local"
	const testExternalURL = "provider.com"
	const testClientID = "unit-test-client-id"

	// create temporary files with test data and override the filenames
	serviceURLFile, err := file.MakeTempFile(testServiceURL)
	if serviceURLFile != nil {
		defer os.Remove(serviceURLFile.Name())
	}
	assert.NoError(t, err)

	externalURLFile, err := file.MakeTempFile(testExternalURL)
	if externalURLFile != nil {
		defer os.Remove(externalURLFile.Name())
	}
	assert.NoError(t, err)

	clientIDFile, err := file.MakeTempFile(testClientID)
	if clientIDFile != nil {
		defer os.Remove(clientIDFile.Name())
	}
	assert.NoError(t, err)

	// restore the filenames when this test is done
	oldServiceURLFilename := serviceURLFilename
	defer func() { serviceURLFilename = oldServiceURLFilename }()
	serviceURLFilename = serviceURLFile.Name()

	oldExternalURLFilename := externalURLFilename
	defer func() { externalURLFilename = oldExternalURLFilename }()
	externalURLFilename = externalURLFile.Name()

	oldClientIDFilename := clientIDFilename
	defer func() { clientIDFilename = oldClientIDFilename }()
	clientIDFilename = clientIDFile.Name()

	// override the watch interval
	oldWatchInterval := watchInterval
	defer func() { watchInterval = oldWatchInterval }()
	watchInterval = 500 * time.Millisecond

	// GIVEN initial configuration files
	// WHEN the InitConfiguration function is called
	// THEN the fetched configuration values match the file contents
	err = InitConfiguration(zap.S())
	assert.NoError(t, err)

	assert.Equal(t, testServiceURL, GetServiceURL())
	assert.Equal(t, testExternalURL, GetExternalURL())
	assert.Equal(t, testClientID, GetClientID())

	// GIVEN the file contents are changed
	// WHEN we fetch the configuration values
	// THEN the values eventually match the expected updated file contents
	const newTestServiceURL = "new-provider.namespace.svc.cluster.local"
	const newTestExternalURL = "new-provider.com"
	const newTestClientID = "new-unit-test-client-id"

	// update the file contents and validate that the new values are loaded
	err = os.WriteFile(serviceURLFilename, []byte(newTestServiceURL), 0)
	assert.NoError(t, err)

	err = os.WriteFile(externalURLFilename, []byte(newTestExternalURL), 0)
	assert.NoError(t, err)

	err = os.WriteFile(clientIDFilename, []byte(newTestClientID), 0)
	assert.NoError(t, err)

	eventually(func() bool { return GetServiceURL() == newTestServiceURL })
	eventually(func() bool { return GetExternalURL() == newTestExternalURL })
	eventually(func() bool { return GetClientID() == newTestClientID })

	// stop the goroutine
	keepWatching.Store(false)
}

// eventually executes the provided function until it either returns true or exceeds a number of attempts
func eventually(f func() bool) bool {
	for i := 0; i < 10; i++ {
		if f() == true {
			return true
		}
		time.Sleep(watchInterval)
	}
	return false
}
