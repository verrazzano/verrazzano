// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package files handles searching
package files

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"regexp"
)

// GetMatchingFiles returns the filenames for files that match a regular expression.
func GetMatchingFiles(log *zap.SugaredLogger, rootDirectory string, fileMatchRe *regexp.Regexp) (fileMatches []string, err error) {
	log.Debugf("GetMatchingFiles called with rootDirectory: %s", rootDirectory)
	if len(rootDirectory) == 0 {
		log.Debugf("GetMatchingFiles requires a rootDirectory")
		return nil, errors.New("GetMatchingFiles requires a rootDirectory")
	}

	if fileMatchRe == nil {
		return nil, fmt.Errorf("GetMatchingFiles requires a regular expression")
	}

	walkFunc := func(fileName string, fileInfo os.FileInfo, err error) error {
		if !fileMatchRe.MatchString(fileName) {
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
func GetMatchingDirectories(log *zap.SugaredLogger, rootDirectory string, fileMatchRe *regexp.Regexp) (fileMatches []string, err error) {
	log.Debugf("GetMatchingFiles called with rootDirectory: %s", rootDirectory)
	if len(rootDirectory) == 0 {
		log.Debugf("GetMatchingDirectories requires a root directory")
		return nil, errors.New("GetMatchingDirectories requires a rootDirectory")
	}

	if fileMatchRe == nil {
		return nil, fmt.Errorf("GetMatchingDirectories requires a regular expression")
	}

	walkFunc := func(fileName string, fileInfo os.FileInfo, err error) error {
		if !fileMatchRe.MatchString(fileName) {
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

// FindNamespaces relies on the directory structure of the cluster-snapshot/namespaces to
// determine the namespaces that were found in the dump. It will return the
// namespace only here and not the entire path.
func FindNamespaces(log *zap.SugaredLogger, clusterRoot string) (namespaces []string, err error) {
	fileInfos, err := os.ReadDir(clusterRoot)
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

// FindFileInClusterRoot will find filename in the cluster root
func FindFileInClusterRoot(clusterRoot string, filename string) string {
	return fmt.Sprintf("%s/%s", clusterRoot, filename)
}

// FindFileNameInNamespace will find filename in the namespace
func FindFileInNamespace(clusterRoot string, namespace string, filename string) string {
	return fmt.Sprintf("%s/%s/%s", clusterRoot, namespace, filename)
}

// FindPodLogFileName will find the name of the log file given a pod
func FindPodLogFileName(clusterRoot string, pod corev1.Pod) string {
	return fmt.Sprintf("%s/%s/%s/logs.txt", clusterRoot, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
}
