// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"context"
	spiErrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Resource struct {
	Namespace string
	Name      string
	Client    client.Client
	Object    client.Object
	Log       vzlog.VerrazzanoLogger
}

// CleanupResources deletes and removes finalizers from resources
func CleanupResources(ctx spi.ComponentContext, objects []client.Object) error {
	for _, object := range objects {
		err := Resource{
			Name:      object.GetName(),
			Namespace: object.GetNamespace(),
			Client:    ctx.Client(),
			Object:    object,
			Log:       ctx.Log(),
		}.RemoveFinalizersAndDelete()
		if err != nil {
			ctx.Log().ErrorfThrottled("Unexpected error cleaning up resource %s/%s still exists: %s", object.GetName(), object.GetNamespace(), err.Error())
			return spiErrors.RetryableError{
				Source:    ctx.GetComponent(),
				Operation: ctx.GetOperation(),
				Cause:     err,
			}
		}
	}
	return nil
}

// VerifyResourcesDeleted verifies resources have been fully cleaned up
func VerifyResourcesDeleted(ctx spi.ComponentContext, objects []client.Object) error {
	for _, object := range objects {
		exists, err := Resource{
			Name:      object.GetName(),
			Namespace: object.GetNamespace(),
			Client:    ctx.Client(),
			Object:    object,
			Log:       ctx.Log(),
		}.Exists()
		if err != nil {
			ctx.Log().ErrorfThrottled("Unexpected error checking if resource %s/%s still exists: %s", object.GetName(), object.GetNamespace(), err.Error())
			return spiErrors.RetryableError{
				Source:    ctx.GetComponent(),
				Operation: ctx.GetOperation(),
				Cause:     err,
			}
		}
		if exists {
			return spiErrors.RetryableError{
				Source: ctx.GetComponent(),
			}
		}
		ctx.Log().Debugf("Verified that resource %s/%s has been successfully deleted", object.GetName(), object.GetNamespace())
	}
	return nil
}

// Delete deletes a resource if it exists, not found error is ignored
func (r Resource) Delete() error {
	r.Object.SetName(r.Name)
	r.Object.SetNamespace(r.Namespace)
	val := reflect.ValueOf(r.Object)
	kind := val.Elem().Type().Name()
	if kind == "Unstructured" {
		kind = r.Object.GetObjectKind().GroupVersionKind().Kind
	}
	err := r.Client.Delete(context.TODO(), r.Object)
	if err != nil {
		// Ignore if CRD doesn't exist
		if _, ok := err.(*meta.NoKindMatchError); ok {
			return nil
		}
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return r.Log.ErrorfNewErr("Failed to delete the resource of type %s, named %s/%s: %v", kind, r.Object.GetNamespace(), r.Object.GetName(), err)
	}
	r.Log.Oncef("Successfully deleted %s %s/%s", kind, r.Object.GetNamespace(), r.Object.GetName())
	return nil
}

// RemoveFinializersAndDelete removes all finalizers from a resource and deletes the resource
func (r Resource) RemoveFinalizersAndDelete() error {
	// always delete first, then remove finalizer to reduce the chance that a Rancher webhook
	// will add it back (since the deletion timestamp will be non-zero)
	err := r.Delete()
	if err != nil {
		return err
	}
	return r.RemoveFinalizers()
}

// RemoveFinalizers removes all finalizers from a resource
func (r Resource) RemoveFinalizers() error {
	val := reflect.ValueOf(r.Object)
	kind := val.Elem().Type().Name()
	if kind == "Unstructured" {
		kind = r.Object.GetObjectKind().GroupVersionKind().Kind
	}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, r.Object)
	if client.IgnoreNotFound(err) != nil {
		return r.Log.ErrorfNewErr("Failed to get the resource of type %s, named %s/%s: %v", kind, r.Object.GetNamespace(), r.Object.GetName(), err)
	} else if err == nil {
		if len(r.Object.GetFinalizers()) != 0 {
			r.Object.SetFinalizers([]string{})
			err = r.Client.Update(context.TODO(), r.Object)
			if err != nil {
				return r.Log.ErrorfNewErr("Failed to update the resource of type %s, named %s/%s: %v", kind, r.Object.GetNamespace(), r.Object.GetName(), err)
			}
		}
	}
	return nil
}

// Exists returns true if the Resource exists
func (r Resource) Exists() (bool, error) {
	err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, r.Object)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		// Ignore if CRD doesn't exist
		if _, ok := err.(*meta.NoKindMatchError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
