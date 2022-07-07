// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Resource struct {
	Namespace string
	Name      string
	Client    client.Client
	Object    client.Object
	Log       vzlog.VerrazzanoLogger
}

// Delete deletes a resource if it exists, not found error is ignored
func (r Resource) Delete() error {
	err := r.Client.Delete(context.TODO(), r.Object)
	if client.IgnoreNotFound(err) != nil {
		return r.Log.ErrorfNewErr("Failed to delete the %s %s/%s: %v", r.Object.GetObjectKind(), r.Object.GetNamespace(), r.Object.GetName(), err)
	} else if err == nil {
		r.Log.Oncef("Successfully deleted %s %s/%s", r.Object.GetObjectKind(), r.Object.GetNamespace(), r.Object.GetName())
	}
	return nil
}
