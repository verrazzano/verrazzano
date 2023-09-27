// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package custom

import (
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeExporterCleanup cleans up any resources from the old node-exporter that was
// replaced with the node-exporter from the prometheus-operator
func NodeExporterCleanup(cli client.Client, log vzlog.VerrazzanoLogger) error {
	err := resource.Resource{
		Name:   nodeExporterName,
		Client: cli,
		Object: &rbacv1.ClusterRoleBinding{},
		Log:    log,
	}.Delete()
	if err != nil {
		return err
	}
	err = resource.Resource{
		Name:   nodeExporterName,
		Client: cli,
		Object: &rbacv1.ClusterRole{},
		Log:    log,
	}.Delete()
	if err != nil {
		return err
	}

	return nil
}
