// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package files handles searching
package files

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"regexp"
)

// GetMatchingFiles returns the filenames for files that match a regular expression.
func GetMatchingFiles(log *zap.SugaredLogger, rootDirectory string, fileMatch string) (fileMatches []string, err error) {
	log.Debugf("GetMatchingFiles called rootDirectory: %s fileMatch: %s", rootDirectory, fileMatch)
	if len(rootDirectory) == 0 || len(fileMatch) == 0 {
		log.Debugf("GetMatchingFiles invalid argument. rootDirectory: %s fileMatch: %s", rootDirectory, fileMatch)
		return nil, errors.New("GetMatchingFiles requires a rootDirectory and fileMatch expression")
	}

	fileMatchRe, err := regexp.Compile(fileMatch)
	if err != nil {
		log.Debugf("Failed to compile regular expression: %s", fileMatch, err)
		return nil, err
	}

	walkFunc := func(fileName string, fileInfo os.FileInfo, err error) error {
		if fileMatchRe.MatchString(fileName) == false {
			return nil
		}
		if !fileInfo.IsDir() {
			log.Debugf("GetMatchingFiles %s matched", fileName)
			fileMatches = append(fileMatches, fileName)
		}
		return nil
	}

	err = filepath.Walk(rootDirectory, walkFunc)
	if err != nil {
		log.Debugf("GetMatchingFiles failed to walk the filepath", err)
		return nil, err
	}
	return fileMatches, err
}

// GetMatchingDirectories returns the filenames for directories that match a regular expression.
func GetMatchingDirectories(log *zap.SugaredLogger, rootDirectory string, fileMatch string) (fileMatches []string, err error) {
	log.Debugf("GetMatchingFiles called rootDirectory: %s fileMatch: %s", rootDirectory, fileMatch)
	if len(rootDirectory) == 0 || len(fileMatch) == 0 {
		log.Debugf("GetMatchingDirectories invalid argument. rootDirectory: %s fileMatch: %s", rootDirectory, fileMatch)
		return nil, errors.New("GetMatchingFiles requires a rootDirectory and fileMatch expression")
	}

	fileMatchRe, err := regexp.Compile(fileMatch)
	if err != nil {
		log.Debugf("Failed to compile regular expression: %s", fileMatch, err)
		return nil, err
	}

	walkFunc := func(fileName string, fileInfo os.FileInfo, err error) error {
		if fileMatchRe.MatchString(fileName) == false {
			return nil
		}
		if fileInfo.IsDir() {
			log.Debugf("GetMatchingDirectories %s matched", fileName)
			fileMatches = append(fileMatches, fileName)
		}
		return nil
	}

	err = filepath.Walk(rootDirectory, walkFunc)
	if err != nil {
		log.Debugf("GetMatchingFiles failed to walk the filepath", err)
		return nil, err
	}
	return fileMatches, nil
}

// FindNamespaces relies on the directory structure of the cluster-dump/namespaces to
// determine the namespaces that were found in the dump. It will return the
// namespace only here and not the entire path.
func FindNamespaces(log *zap.SugaredLogger, clusterRoot string) (namespaces []string, err error) {
	fileInfos, err := ioutil.ReadDir(clusterRoot)
	if err != nil {
		log.Debugf("FindNamespaces failed to read directory %s", clusterRoot, err)
		return nil, err
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			namespaces = append(namespaces, filepath.Base(fileInfo.Name()))
		}
	}
	return namespaces, nil
}

//FindPodLogFileName will find the name of the log file given a pod
func FindPodLogFileName(clusterRoot string, pod corev1.Pod) (name string) {
	return fmt.Sprintf("%s/%s/%s/logs.txt", clusterRoot, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
}

// FindEventsJSONFileName finds the events JSON filename
func FindEventsJSONFileName(clusterRoot string, namespace string) (name string) {
	return fmt.Sprintf("%s/%s/events.json", clusterRoot, namespace)
}
