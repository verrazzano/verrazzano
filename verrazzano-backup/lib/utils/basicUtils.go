// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils

import (
	"crypto/rand"
	"fmt"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
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
func HTTPHelper(method, requestURL string, body io.Reader, log *zap.SugaredLogger) ([]byte, error) {
	log.Debugf("Invoking HTTP '%s' request with url '%s'", method, requestURL)
	var response *http.Response
	var request *http.Request
	var err error
	switch method {
	case "GET":
		response, err = http.Get(requestURL) //#nosec G204
		if err != nil {
			log.Errorf("HTTP GET failure while invoking url '%s'", requestURL, zap.Error(err))
			return nil, err
		}
	case "POST":
		response, err = http.Post(requestURL, constants.HTTPContentType, body) //#nosec G204
		if err != nil {
			log.Errorf("HTTP POST failure while invoking url '%s'", requestURL, zap.Error(err))
			return nil, err
		}
	case "DELETE":
		request, err = http.NewRequest(http.MethodDelete, requestURL, body) //#nosec G204
		if err != nil {
			log.Error("Error creating request ", zap.Error(err))
			return nil, err
		}
		client := &http.Client{}
		request.Header.Add("Content-Type", constants.HTTPContentType)
		response, err = client.Do(request)
		if err != nil {
			log.Errorf("HTTP DELETE failure while invoking url '%s'", requestURL, zap.Error(err))
			return nil, err
		}
	}

	bdata, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("HTTP DELETE failure ", zap.Error(err))
		return nil, err
	}

	if response.StatusCode != 200 {
		log.Errorf("Response code is not 200 OK!. Actual response code '%v' with response body '%v'", response.StatusCode, string(bdata))
		return nil, err
	}

	return bdata, nil
}
