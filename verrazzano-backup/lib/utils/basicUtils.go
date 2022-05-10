// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strings"
)

//CreateTempFileWithData used to create temp cloud-creds utilized for object store access
func CreateTempFileWithData(data []byte) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "cloud-creds-*.ini")
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

//GenerateRandom generates a random number between min and max
func GenerateRandom() int {
	randomBig, err := rand.Int(rand.Reader, big.NewInt(constants.Max))
	if err != nil {
		fmt.Println(err)
	}
	randomInt := int(randomBig.Int64())
	if randomInt < constants.Min {
		return (constants.Min + constants.Max) / 2
	}
	return randomInt
}

//HTTPHelper supports net/http calls of type GTE/POST/DELETE
func HTTPHelper(method, requestURL string, body io.Reader, data interface{}, log *zap.SugaredLogger) error {
	log.Debugf("Invoking HTTP '%s' request with url '%s'", method, requestURL)
	var response *http.Response
	var request *http.Request
	var err error
	client := &http.Client{}
	switch method {
	case "GET":
		request, err = http.NewRequest(http.MethodGet, requestURL, body)
		if err != nil {
			log.Error("Error creating request ", zap.Error(err))
			return err
		}
	case "POST":
		request, err = http.NewRequest(http.MethodPost, requestURL, body)
		if err != nil {
			log.Error("Error creating request ", zap.Error(err))
			return err
		}
	case "DELETE":
		request, err = http.NewRequest(http.MethodDelete, requestURL, body)
		if err != nil {
			log.Error("Error creating request ", zap.Error(err))
			return err
		}
	}
	request.Header.Add("Content-Type", constants.HTTPContentType)
	response, err = client.Do(request)
	if err != nil {
		log.Errorf("HTTP '%s' failure while invoking url '%s' due to '%v'", method, requestURL, zap.Error(err))
		return err
	}

	bdata, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("HTTP DELETE failure ", zap.Error(err))
		return err
	}

	if response.StatusCode != 200 {
		log.Errorf("Response code is not 200 OK!. Actual response code '%v' with response body '%v'", response.StatusCode, string(bdata))
		return err
	}

	err = json.Unmarshal(bdata, &data)
	if err != nil {
		log.Errorf("json unmarshalling error %v", err)
		return err
	}

	return nil
}

//ReadTempCredsFile reads object store credentials from a temporary file for registration purpose
func ReadTempCredsFile(filePath string) (string, string, error) {
	var awsAccessKey, awsSecretAccessKey string
	f, err := os.Open(filePath)
	if err != nil {
		return "", "", nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.Contains(line, constants.AwsAccessKeyString) {
				words := strings.Split(line, fmt.Sprintf("%s=", constants.AwsAccessKeyString))
				awsAccessKey = words[len(words)-1]
			}
			if strings.Contains(line, constants.AwsSecretAccessKeyString) {
				words := strings.Split(line, fmt.Sprintf("%s=", constants.AwsSecretAccessKeyString))
				awsSecretAccessKey = words[len(words)-1]
			}
		}
	}
	return awsAccessKey, awsSecretAccessKey, nil
}

// GetEnvWithDefault retrieves env variable with default value
func GetEnvWithDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}
