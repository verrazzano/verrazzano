// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// these can be changed for unit testing
var (
	issuerURLFilename = "/etc/config/oidcIssuerURL"
	clientIDFilename  = "/etc/config/oidcClientID"

	watchInterval = time.Minute
	keepWatching  atomic.Bool
)

var (
	issuerURL string
	clientID  string

	issuerURLFileModTime time.Time
	clientIDFileModTime  time.Time

	mutex sync.RWMutex
)

// GetIssuerURL returns the issuer URL
func GetIssuerURL() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return issuerURL
}

// GetClientID returns the client ID
func GetClientID() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return clientID
}

// loadIssuerURL loads the issuer URL from a file and stores the file modification time
func loadIssuerURL() error {
	mutex.Lock()
	defer mutex.Unlock()

	value, modTime, err := loadConfigValue(issuerURLFilename)
	if err != nil {
		return err
	}

	issuerURL = value
	issuerURLFileModTime = *modTime
	return nil
}

// loadClientID loads the client ID from a file and stores the file modification time
func loadClientID() error {
	mutex.Lock()
	defer mutex.Unlock()

	value, modTime, err := loadConfigValue(clientIDFilename)
	if err != nil {
		return err
	}

	clientID = value
	clientIDFileModTime = *modTime
	return nil
}

// loadConfigValue loads a configuration value from a file and stores the file modification time
func loadConfigValue(filename string) (string, *time.Time, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return "", nil, err
	}

	fileInfo, err := os.Stat(filename)
	if err != nil {
		return "", nil, err
	}

	modTime := fileInfo.ModTime()
	return string(bytes), &modTime, nil
}

// InitConfiguration loads the configuration from files and starts a goroutine to watch for configuration changes and reloads
// config values when changes are detected
func InitConfiguration(log *zap.SugaredLogger) error {
	if err := loadIssuerURL(); err != nil {
		return err
	}
	if err := loadClientID(); err != nil {
		return err
	}

	keepWatching.Store(true)
	go watchConfigForChanges(log)
	return nil
}

// watchConfigForChanges watches the configuration files for changes and reloads as necessary. This function generally
// runs forever but the keepWatching atomic bool can be set to false in unit tests to stop the loop.
func watchConfigForChanges(log *zap.SugaredLogger) {
	for keepWatching.Load() {
		if err := reloadConfigWhenChanged(log); err != nil {
			log.Warnf("Error reloading configuration: %v", err)
		}
		time.Sleep(watchInterval)
	}
}

// reloadConfigWhenChanged compares the config file modification times and reloads config values
func reloadConfigWhenChanged(log *zap.SugaredLogger) error {
	fileInfo, err := os.Stat(issuerURLFilename)
	if err != nil {
		return err
	}
	if fileInfo.ModTime().After(issuerURLFileModTime) {
		// file has changed
		log.Debugf("Detected change in file %s, reloading contents", issuerURLFilename)
		if err := loadIssuerURL(); err != nil {
			return err
		}
	}

	fileInfo, err = os.Stat(clientIDFilename)
	if err != nil {
		return err
	}
	if fileInfo.ModTime().After(clientIDFileModTime) {
		// file has changed
		log.Debugf("Detected change in file %s, reloading contents", clientIDFilename)
		if err := loadClientID(); err != nil {
			return err
		}
	}

	return nil
}
