// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"io"
	"os"

	"go.uber.org/zap"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	corev1 "k8s.io/api/core/v1"
)

type cattleStatus struct {
	Conditions []cattleCondition `json:"conditions,omitempty"`
}
type cattleCondition struct {
	Status  corev1.ConditionStatus `json:"status"`
	Type    string                 `json:"type"`
	Reason  string                 `json:"reason,omitempty"`
	Message string                 `json:"message,omitempty"`
}

func readFileFromClusterPath(log *zap.SugaredLogger, clusterRoot string, filename string) ([]byte, string, error) {
	clusterPath := files.FindFileInClusterRoot(clusterRoot, filename)

	// Parse the json into local struct
	file, err := os.Open(clusterPath)
	if err != nil {
		// The file may not exist if Rancher is not installed.
		log.Debugf("file %s not found", clusterPath)
		return []byte{}, "", nil
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Debugf("Failed reading Json file %s", clusterPath)
		return []byte{}, "", err
	}
	return fileBytes, clusterPath, nil
}
