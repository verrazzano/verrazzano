// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Delete deletes a resource if it exists, not found error is ignored
func Delete(log vzlog.VerrazzanoLogger, cli client.Client, o client.Object) error {
	err := cli.Delete(context.TODO(), o)
	if client.IgnoreNotFound(err) != nil {
		return log.ErrorfNewErr("Failed to delete the %s %s/%s: %v", o.GetObjectKind(), o.GetNamespace(), o.GetName(), err)
	} else if err == nil {
		log.Oncef("Successfully deleted %s %s/%s", o.GetObjectKind(), o.GetNamespace(), o.GetName())
	}
	return nil
}
