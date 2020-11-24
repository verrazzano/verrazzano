// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"github.com/onsi/ginkgo"
	"os"
	"path/filepath"
)

// GetKubeconfig returns the KubeConfig in string format
func GetKubeconfig() string {
	var kubeconfig string
	if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		ginkgo.Fail("Could not get kubeconfig")
	}
	return kubeconfig
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return ""
}
