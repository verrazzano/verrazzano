// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestFileUploadUtil calls the utility functions defined in file_upload_util
func TestFileUploadUtil(t *testing.T) {
	objectStorageClient, err := getObjectStorageClient("")
	assert.NoError(t, err)
	assert.NotNil(t, objectStorageClient)
	// Invalid format for the compartment id
	compartmentOCID := "ocid1.compartment.oc1..testcompartment"
	err = createBucket(objectStorageClient, compartmentOCID, "testnamespace", "testbucket")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unable to parse OCID as any format")
	// The file url mentioned is invalid, so downloading the same will fail
	err = uploadObject(objectStorageClient, "http://testdomain/testfile.zip", "archive.zip", "testnamespace", "testbucket")
	assert.Contains(t, err.Error(), "HTTP Error other than 200")
}
