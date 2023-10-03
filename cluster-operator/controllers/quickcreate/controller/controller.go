// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

// Reusable code for Quick Create controllers

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Base struct {
	clipkg.Client
	Log vzlog.VerrazzanoLogger
}

func (b *Base) SetNewResourceLogger(o clipkg.Object) error {
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           o.GetName(),
		Namespace:      o.GetNamespace(),
		ID:             string(o.GetUID()),
		Generation:     o.GetGeneration(),
		ControllerName: o.GetObjectKind().GroupVersionKind().Kind,
	})
	if err != nil {
		return err
	}
	b.Log = log
	return nil
}

func (b *Base) UpdateStatus(ctx context.Context, o clipkg.Object) (ctrl.Result, error) {
	if err := b.Status().Update(ctx, o); err != nil {
		return RequeueDelay(), err
	}
	return ctrl.Result{}, nil
}

func (b *Base) Cleanup(ctx context.Context, o clipkg.Object, finalizerKey string) error {
	if o.GetDeletionTimestamp().IsZero() {
		if err := b.Delete(ctx, o); err != nil {
			return err
		}
	}
	if vzstring.SliceContainsString(o.GetFinalizers(), finalizerKey) {
		o.SetFinalizers(vzstring.RemoveStringFromSlice(o.GetFinalizers(), finalizerKey))
		err := b.Update(ctx, o)
		if err != nil && !apierrors.IsConflict(err) {
			return err
		}
	}
	return nil
}

func (b *Base) SetFinalizers(ctx context.Context, o clipkg.Object, finalizers ...string) (ctrl.Result, error) {
	o.SetFinalizers(append(o.GetFinalizers(), finalizers...))
	if err := b.Update(ctx, o); err != nil {
		return RequeueDelay(), err
	}
	return ctrl.Result{}, nil
}

func RequeueDelay() ctrl.Result {
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: time.Duration(30) * time.Second,
	}
}

func ApplyTemplates(cli clipkg.Client, props any, ns string, templates ...[]byte) error {
	applier := k8sutil.NewYAMLApplier(cli, ns)
	for _, tmpl := range templates {
		if err := applier.ApplyBT(tmpl, props); err != nil {
			return err
		}
	}
	return nil
}
