// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	r.Object.SetName(r.Name)
	r.Object.SetNamespace(r.Namespace)
	err := r.Client.Delete(context.TODO(), r.Object)
	if client.IgnoreNotFound(err) != nil {
		return r.Log.ErrorfNewErr("Failed to delete the %s %s/%s: %v", r.Object.GetObjectKind().GroupVersionKind().Kind, r.Object.GetNamespace(), r.Object.GetName(), err)
	} else if err == nil {
		r.Log.Oncef("Successfully deleted %s %s/%s", r.Object.GetObjectKind().GroupVersionKind().Kind, r.Object.GetNamespace(), r.Object.GetName())
	}
	return nil
}

// RemoveFinalizers remove all finalizers from a resource
func (r Resource) RemoveFinalizers() error {
	err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, r.Object)
	if err != nil && !errors.IsNotFound(err) {
		return r.Log.ErrorfNewErr("Failed to get the resource of type %s, named %s/%s: %v", r.Object.GetObjectKind().GroupVersionKind().Kind, r.Object.GetNamespace(), r.Object.GetName(), err)
	} else if err == nil {
		r.Object.SetFinalizers([]string{})
		err = r.Client.Update(context.TODO(), r.Object)
		if err != nil {
			return r.Log.ErrorfNewErr("Failed to update the resource of type %s, named %s/%s: %v", r.Object.GetObjectKind().GroupVersionKind().Kind, r.Object.GetNamespace(), r.Object.GetName(), err)
		}
	}
	return nil
}
