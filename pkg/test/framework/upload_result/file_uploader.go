// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"errors"
	"flag"
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

var fileURL string

var objectName string

var region string

var bucketName string

var nameSpace string

var createBucket string

var compartmentID = os.Getenv("COMPARTMENT_ID")

var userName = os.Getenv("USER_NAME")

var password = os.Getenv("PASSWORD")

// getNamespace returns the name of the Object Storage namespace for the user making the request.
func getNamespace(client objectstorage.ObjectStorageClient) string {
	req := objectstorage.GetNamespaceRequest{}
	r, err := client.GetNamespace(context.Background(), req)
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("An error occurred while getting the namespace from the ObjectStorageClient: %s\n", err))
		os.Exit(1)
	}
	return *r.Value
}

func createBucketInternal(client objectstorage.ObjectStorageClient, namespace string, name string, compartmentOCID string) error {
	req := objectstorage.GetBucketRequest{
		NamespaceName: &namespace,
		BucketName:    &name,
	}

	// verify if bucket exists
	response, err := client.GetBucket(context.Background(), req)
	if err == nil {
		return err
	}

	// Create the bucket when the bucket is not found and createBucket is set to true
	if 404 == response.RawResponse.StatusCode && "true" == createBucket {
		req := objectstorage.CreateBucketRequest{
			CreateBucketDetails: objectstorage.CreateBucketDetails{StorageTier: objectstorage.CreateBucketDetailsStorageTierStandard,
				Versioning:          objectstorage.CreateBucketDetailsVersioningDisabled,
				CompartmentId:       &compartmentOCID,
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

	} else {
		// Either it is an error other than 404 or createBucket is set to false.
		pkg.Log(pkg.Info, fmt.Sprintf("Creating the bucket %s failed, either the response code is other than 404 or the command line argument create-object is set to false\n", name))
		pkg.Log(pkg.Info, fmt.Sprintln(err))
		return err
	}
	return nil
}

// uploadObject uploads the given object to the Object Storage, in the given bucket.
func uploadObject(c objectstorage.ObjectStorageClient, namespace, bucket, objectName string, contentLen int64, content io.ReadCloser, metadata map[string]string) error {
	req := objectstorage.PutObjectRequest{
		NamespaceName: &namespace,
		BucketName:    &bucket,
		ObjectName:    &objectName,
		ContentLength: &contentLen,
		PutObjectBody: content,
		OpcMeta:       metadata,
		StorageTier:   objectstorage.PutObjectStorageTierStandard,
	}
	_, err := c.PutObject(context.Background(), req)
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("There is an error in uploading the file to the Object Storage: %s\n", err))
		return err
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Successfully uploaded %s to the bucket %s\n", objectName, bucket))
	return nil
}

// getObjectStorageClient returns an OCI SDK client for ObjectStorage. If a region is specified then use an instance
// principal auth provider, otherwise use the default provider (auth config comes from
// an OCI config file or environment variables). Instance principals are used when running in the
// CI/CD pipelines while the default provider is suitable for running locally.
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

// downloadArtifact downloads the file at the artifactURL to local file system
func downloadArtifact(artifactURL, localFile string) error {
	client := http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, artifactURL, http.NoBody)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error while creating a new request: %s\n", err))
		return err
	}

	if userName != "" && password != "" {
		req.SetBasicAuth(userName, password)
	} else {
		pkg.Log(pkg.Info, fmt.Sprintln("Username and/password to be used for HTTP Basic Authentication is not set"))
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
	defer outFile.Close()

	// Writer the body to file
	_, copyErr := io.Copy(outFile, res.Body)
	if copyErr != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error copying the contents to the out file: %s\n", copyErr))
		return copyErr
	}
	return nil
}

func main() {
	if compartmentID == "" {
		pkg.Log(pkg.Info, fmt.Sprintln("Set an environment variable COMPARTMENT_ID containing the OCID of the compartment"))
		os.Exit(1)
	}

	fileName := path.Base(fileURL)
	err := downloadArtifact(fileURL, fileName)
	if err != nil {
		os.Exit(1)
	}

	// Open the named file for reading.
	file, openErr := os.Open(fileName)
	if openErr != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("There is an error in opening the file for reading: %s\n", openErr))
		os.Exit(1)
	}
	defer file.Close()

	// Get the FileInfo structure to determine the size
	fileInfo, _ := file.Stat()

	// Get the objectName from the last element of the path
	if objectName == "" {
		objectName = fileName
	}

	// Get an OCI SDK client for ObjectStorage
	objectStorageClient, clientErr := getObjectStorageClient(region)
	if clientErr != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("There is an error in obtaining an OCI SDK client for ObjectStorage: %s\n", clientErr))
		os.Exit(1)
	}

	// Derive namespace, if not provided as the command line argument
	if nameSpace == "" {
		nameSpace = getNamespace(objectStorageClient)
	}

	// Create bucket if it doesn't exist and command line argument create-bucket is set to true
	createErr := createBucketInternal(objectStorageClient, nameSpace, bucketName, compartmentID)
	if createErr != nil {
		os.Exit(1)
	}

	err = uploadObject(objectStorageClient, nameSpace, bucketName, objectName, fileInfo.Size(), file, nil)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&fileURL, "file-url", "", "URL of the artifact to be uploaded to Object Storage")
	flag.StringVar(&objectName, "object-name", "", "The name of the object in the Object Storage, if different than the actual file name")
	flag.StringVar(&nameSpace, "namespace", "", "Object Storage namespace")
	flag.StringVar(&bucketName, "bucket-name", "", "Name of the bucket")
	flag.StringVar(&createBucket, "create-bucket", "false", "Flag to indicate whether to create the bucket, if it is not there")
	flag.StringVar(&region, "region", "", "OCI region")
	flag.Parse()

	if fileURL == "" {
		pkg.Log(pkg.Info, fmt.Sprintln("Required command line argument archive-url is not specified, exiting."))
		printUsage()
		os.Exit(1)
	}

	if bucketName == "" {
		pkg.Log(pkg.Info, fmt.Sprintln("Required command line argument bucket-name is not specified, exiting."))
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	usageString := "Usage: go run file_uploader.go --file-url=<URL of the artifact to be uploaded to Object Storage>" +
		" --object-name=<name of the object in the Object Storage, if different than the actual file name>" +
		" --namespace=<Object Storage namespace used for the request> " +
		" --bucket-name=<name of the bucket - container for storing objects in a compartment within an Object Storage namespace>" +
		" --create-bucket=<flag to indicate whether to create the bucket, if it doesn't exist> " +
		" --region=<OCI region> "
	fmt.Println(usageString)
}
