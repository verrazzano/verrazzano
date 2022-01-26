// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"context"
	"errors"
	"fmt"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/common/auth"
	"github.com/oracle/oci-go-sdk/v53/objectstorage"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"io"
	"net/http"
	"os"
	"path"
	"time"
)

var (
	userName = os.Getenv("FILE_READ_USER")
	password = os.Getenv("FILE_READ_PASSWORD")
)

// getObjectStorageClient returns an OCI SDK client for ObjectStorage. If a region is specified then use an instance
// principal auth provider, otherwise use the default provider (auth config comes from
// an OCI config file or environment variables).
func getObjectStorageClient(region string) (objectstorage.ObjectStorageClient, error) {
	var provider common.ConfigurationProvider
	var err error

	if region != "" {
		pkg.Log(pkg.Info, fmt.Sprintf("Using OCI SDK instance principal provider with region: %s\n", region))
		provider, err = auth.InstancePrincipalConfigurationProviderForRegion(common.StringToRegion(region))
	} else {
		pkg.Log(pkg.Info, fmt.Sprintln("Using OCI SDK default provider"))
		provider = common.DefaultConfigProvider()
	}

	if err != nil {
		return objectstorage.ObjectStorageClient{}, err
	}
	return objectstorage.NewObjectStorageClientWithConfigurationProvider(provider)
}

// createBucket creates the bucket with the name name in the compartment compartmentID, if it doesn't exist.
func createBucket(client objectstorage.ObjectStorageClient, compartmentID, namespace, name string) error {
	req := objectstorage.GetBucketRequest{
		NamespaceName: &namespace,
		BucketName:    &name,
	}

	// verify if bucket exists
	response, err := client.GetBucket(context.Background(), req)
	if err == nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Bucket with the name: %s found in the namespace: %s\n", name, namespace))
		return nil
	}

	// Create the bucket for all the HTTP 4xx error
	if response.RawResponse.StatusCode >= http.StatusBadRequest && response.RawResponse.StatusCode <= http.StatusInternalServerError {
		req := objectstorage.CreateBucketRequest{
			CreateBucketDetails: objectstorage.CreateBucketDetails{StorageTier: objectstorage.CreateBucketDetailsStorageTierStandard,
				Versioning:          objectstorage.CreateBucketDetailsVersioningDisabled,
				CompartmentId:       &compartmentID,
				ObjectEventsEnabled: common.Bool(true),
				Name:                &name,
				PublicAccessType:    objectstorage.CreateBucketDetailsPublicAccessTypeObjectreadwithoutlist},
			NamespaceName: common.String(namespace),
		}
		_, err := client.CreateBucket(context.Background(), req)
		if err != nil {
			pkg.Log(pkg.Info, fmt.Sprintf("An error occurred while creating the bucket: %s\n", err))
			return err
		}
		pkg.Log(pkg.Debug, fmt.Sprintf("Successfully created the bucket: %s\n", err))
	}
	return nil
}

// uploadObject uploads the given object to the Object Storage in the given bucket.
func uploadObject(client objectstorage.ObjectStorageClient, fileURL, fileName, namespace, bucket string) error {
	remoteFile := path.Base(fileURL)
	err := downloadFile(fileURL, remoteFile)
	if err != nil {
		return err
	}

	// Open the named file for reading.
	file, openErr := os.Open(remoteFile)
	if openErr != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("There is an error in opening the file for reading: %s\n", openErr))
		return err
	}
	defer file.Close()

	// Get the fileName from the last element of the path
	if fileName == "" {
		fileName = remoteFile
	}

	// Get the FileInfo structure to determine the size
	fileInfo, _ := file.Stat()
	contentLen := fileInfo.Size()

	req := objectstorage.PutObjectRequest{
		NamespaceName: &namespace,
		BucketName:    &bucket,
		ObjectName:    &fileName,
		ContentLength: &contentLen,
		PutObjectBody: file,
		StorageTier:   objectstorage.PutObjectStorageTierStandard,
	}
	_, uploadErr := client.PutObject(context.Background(), req)
	if uploadErr != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("There is an error in uploading the file to the Object Storage: %s\n", uploadErr))
		return uploadErr
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Successfully uploaded %s to the bucket %s\n", fileName, bucket))
	return nil
}

// downloadFile downloads the file at the fileURL to local file system.
func downloadFile(fileURL, localFile string) error {
	client := http.Client{Timeout: 30 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, fileURL, http.NoBody)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error while creating a new request: %s\n", err))
		return err
	}

	// Set environment variables FILE_READ_USER and FILE_READ_PASSWORD, if required to download the file
	if userName != "" && password != "" {
		req.SetBasicAuth(userName, password)
	} else {
		pkg.Log(pkg.Info, fmt.Sprintln("Username and/password used for HTTP Basic Authentication is not set"))
	}

	res, respErr := client.Do(req)
	if respErr != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error while sending an HTTP request: %s\n", respErr))
		return respErr
	}
	defer res.Body.Close()

	if 200 != res.StatusCode {
		pkg.Log(pkg.Error, fmt.Sprintf("Downloading the file failed with error code: %d\n", res.StatusCode))
		return errors.New("HTTP Error other than 200")
	}

	// create the local file
	outFile, createErr := os.Create(localFile)
	if createErr != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error creating the file: %s\n", localFile))
		return createErr
	}

	// Write the body to the file
	_, copyErr := io.Copy(outFile, res.Body)
	if copyErr != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error copying the contents to the out file: %s\n", copyErr))
		return copyErr
	}
	defer outFile.Close()
	return nil
}
