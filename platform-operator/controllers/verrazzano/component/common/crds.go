// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyCRDYaml(c client.Client) error {
	path := filepath.Join(config.GetHelmAppOpChartsDir(), "/crds")
	yamlApplier := k8sutil.NewYAMLApplier(c, "")
	return yamlApplier.ApplyD(path)
}
