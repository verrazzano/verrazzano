// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"go.uber.org/zap"
	"math/big"
	"os"
	"strings"
	"time"
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
func WaitRandom(message, timeout string, log *zap.SugaredLogger) (int, error) {
	randomBig, err := rand.Int(rand.Reader, big.NewInt(constants.Max))
	if err != nil {
		return 0, fmt.Errorf("Unable to generate random number %v", zap.Error(err))
	}
	randomInt := int(randomBig.Int64())
	if randomInt < constants.Min {
		randomInt = (constants.Min + constants.Max) / 2
	}
	timeParse, err := time.ParseDuration(timeout)
	if err != nil {
		return 0, fmt.Errorf("Unable to parse time duration %v", zap.Error(err))
	}
	// handle timeouts lesser that generated min!
	if float64(randomInt) > timeParse.Seconds() {
		randomInt = int(timeParse.Seconds())
	}
	log.Infof("%v . Wait for '%v' seconds ...", message, randomInt)
	time.Sleep(time.Second * time.Duration(randomInt))
	return randomInt, nil
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

// GetComponent retrieves component info from file
func GetComponent(filePath string) (string, error) {
	var line string
	_, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return "", nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line = strings.TrimSpace(scanner.Text())
	}
	return line, nil
}
