// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package files handles searching
package files

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	utillog "github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/log"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// GetMatchingFileNames returns the filenames for files that match a regular expression.
func GetMatchingFileNames(log *zap.SugaredLogger, rootDirectory string, fileMatchRe *regexp.Regexp) (fileMatches []string, err error) {
	utillog.DebugfIfNotNil(log, "GetMatchingFiles called with rootDirectory: %s", rootDirectory)
	if len(rootDirectory) == 0 {
		utillog.DebugfIfNotNil(log, "GetMatchingFiles requires a rootDirectory")
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
			utillog.DebugfIfNotNil(log, "GetMatchingFiles %s matched", fileName)
			fileMatches = append(fileMatches, fileName)
		}
		return nil
	}

	err = filepath.Walk(rootDirectory, walkFunc)
	if err != nil {
		utillog.DebugfIfNotNil(log, "GetMatchingFiles failed to walk the filepath", err)
		return nil, err
	}
	return fileMatches, err
}

// GetMatchingDirectoryNames returns the filenames for directories that match a regular expression.
func GetMatchingDirectoryNames(log *zap.SugaredLogger, rootDirectory string, fileMatchRe *regexp.Regexp) (fileMatches []string, err error) {
	utillog.DebugfIfNotNil(log, "GetMatchingFiles called with rootDirectory: %s", rootDirectory)
	if len(rootDirectory) == 0 {
		utillog.DebugfIfNotNil(log, "GetMatchingDirectories requires a root directory")
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
			utillog.DebugfIfNotNil(log, "GetMatchingDirectories %s matched", fileName)
			fileMatches = append(fileMatches, fileName)
		}
		return nil
	}

	err = filepath.Walk(rootDirectory, walkFunc)
	if err != nil {
		utillog.DebugfIfNotNil(log, "GetMatchingFiles failed to walk the filepath", err)
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
		return nil, fmt.Errorf("FindNamespaces failed to read directory %s: %s", clusterRoot, err.Error())
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			namespaces = append(namespaces, filepath.Base(fileInfo.Name()))
		}
	}
	return namespaces, nil
}

// FormFilePathInClusterRoot will find filename in the cluster root
func FormFilePathInClusterRoot(clusterRoot string, filename string) string {
	return fmt.Sprintf("%s/%s", clusterRoot, filename)
}

// FormFilePathInNamespace will find filename in the namespace
func FormFilePathInNamespace(clusterRoot string, namespace string, filename string) string {
	return fmt.Sprintf("%s/%s/%s", clusterRoot, namespace, filename)
}

// FindPodLogFileName will find the name of the log file given a pod
func FindPodLogFileName(clusterRoot string, pod corev1.Pod) string {
	return fmt.Sprintf("%s/%s/%s/logs.txt", clusterRoot, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
}

// UnmarshallFileInClusterRoot - unmarshall a file into a struct
func UnmarshallFileInClusterRoot(clusterRoot string, filename string, object interface{}) error {
	clusterPath := FormFilePathInClusterRoot(clusterRoot, filename)
	return unmarshallFile(clusterPath, object)
}

// UnmarshallFileInNamespace - unmarshall a file from a namespace into a struct
func UnmarshallFileInNamespace(clusterRoot string, namespace string, filename string, object interface{}) error {
	clusterPath := FormFilePathInNamespace(clusterRoot, namespace, filename)
	return unmarshallFile(clusterPath, object)
}

func unmarshallFile(clusterPath string, object interface{}) error {
	// Parse the json into local struct
	file, err := os.Open(clusterPath)
	if os.IsNotExist(err) {
		// The file may not exist if the component is not installed.
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open file %s from cluster snapshot: %s", clusterPath, err.Error())
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("Failed reading Json file %s: %s", clusterPath, err.Error())
	}

	// Unmarshall file contents into a struct
	err = json.Unmarshal(fileBytes, object)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal %s: %s", clusterPath, err.Error())
	}

	return nil
}

// GetTimeOfCapture parses the metadata.json file and converts the time string into a time.Time object to be used by other functions
func GetTimeOfCapture(log *zap.SugaredLogger, clusterRoot string) (*time.Time, error) {
	metadataFile := FormFilePathInClusterRoot(clusterRoot, constants.MetadataJSON)
	if _, err := os.Stat(metadataFile); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	metadataObjectToUnmarshalInto := &helpers.Metadata{}
	err := unmarshallFile(metadataFile, &metadataObjectToUnmarshalInto)
	if err != nil {
		return nil, err
	}
	timeString := metadataObjectToUnmarshalInto.Time
	timeObject, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		return nil, err
	}
	return &timeObject, err

}
