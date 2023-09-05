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
	serviceURLFilename  = "/etc/config/oidcServiceURL"
	externalURLFilename = "/etc/config/oidcExternalURL"
	clientIDFilename    = "/etc/config/oidcClientID"

	watchInterval atomic.Uint64
	keepWatching  atomic.Bool
)

var (
	serviceURL  string
	externalURL string
	clientID    string

	serviceURLFileModTime  time.Time
	externalURLFileModTime time.Time
	clientIDFileModTime    time.Time

	mutex sync.RWMutex
)

func init() {
	watchInterval.Store(uint64(time.Minute))
	keepWatching.Store(true)
}

// GetServiceURL returns the in-cluster service URL of the OIDC provider
func GetServiceURL() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return serviceURL
}

// GetExternalURL returns the external URL of the OIDC provider
func GetExternalURL() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return externalURL
}

// GetClientID returns the client ID
func GetClientID() string {
	mutex.RLock()
	defer mutex.RUnlock()
	return clientID
}

// loadServiceURL loads the in-cluster service URL from a file and stores the file modification time
func loadServiceURL() error {
	mutex.Lock()
	defer mutex.Unlock()

	value, modTime, err := loadConfigValue(serviceURLFilename)
	if err != nil {
		return err
	}

	serviceURL = value
	serviceURLFileModTime = *modTime
	return nil
}

// loadExternalURL loads the external URL from a file and stores the file modification time
func loadExternalURL() error {
	mutex.Lock()
	defer mutex.Unlock()

	value, modTime, err := loadConfigValue(externalURLFilename)
	if err != nil {
		return err
	}

	externalURL = value
	externalURLFileModTime = *modTime
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
	if err := loadServiceURL(); err != nil {
		return err
	}
	if err := loadExternalURL(); err != nil {
		return err
	}
	if err := loadClientID(); err != nil {
		return err
	}

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
		time.Sleep(time.Duration(watchInterval.Load()))
	}
}

// reloadConfigWhenChanged compares the config file modification times and reloads config values
func reloadConfigWhenChanged(log *zap.SugaredLogger) error {
	fileInfo, err := os.Stat(serviceURLFilename)
	if err != nil {
		return err
	}
	if fileInfo.ModTime().After(serviceURLFileModTime) {
		// file has changed
		log.Debugf("Detected change in file %s, reloading contents", serviceURLFilename)
		if err := loadServiceURL(); err != nil {
			return err
		}
	}

	fileInfo, err = os.Stat(externalURLFilename)
	if err != nil {
		return err
	}
	if fileInfo.ModTime().After(externalURLFileModTime) {
		// file has changed
		log.Debugf("Detected change in file %s, reloading contents", externalURLFilename)
		if err := loadExternalURL(); err != nil {
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
