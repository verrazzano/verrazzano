// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	restartVersionAnnotation         = "verrazzano.io/restart-version"
	previousRestartVersionAnnotation = "verrazzano.io/previous-restart-version"
)

type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&oamv1.ApplicationConfiguration{}).
		Complete(r)
}

// Reconcile checks restart version annotations on an ApplicationConfiguration and
// restarts applications as needed. When applications are restarted, the previous restart
// version annotation value is updated.
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("applicationconfiguration", req.NamespacedName)
	log.Info("Reconciling ApplicationConfiguration")

	// fetch the appconfig
	var appConfig oamv1.ApplicationConfiguration
	if err := r.Client.Get(ctx, req.NamespacedName, &appConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("ApplicationConfiguration has been deleted", "name", req.NamespacedName)
		} else {
			log.Error(err, "Failed to fetch ApplicationConfiguration", "name", req.NamespacedName)
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// get the user-specified restart version - if it's missing then there's nothing to do here
	restartVersion, ok := appConfig.Annotations[restartVersionAnnotation]
	if !ok {
		log.Info("No restart version annotation found, nothing to do")
		return reconcile.Result{}, nil
	}

	// get the annotation with the previous restart version - if it's missing or the versions do not
	// match, then we restart apps
	prevRestartVersion, ok := appConfig.Annotations[previousRestartVersionAnnotation]
	if !ok || restartVersion != prevRestartVersion {
		log.Info("Restarting applications")

		// restart apps

		// add/update the previous restart version annotation on the appconfig
		appConfig.Annotations[previousRestartVersionAnnotation] = restartVersion
		if err := r.Client.Update(ctx, &appConfig); err != nil {
			return reconcile.Result{}, err
		}
	}

	log.Info("Successfully reconciled ApplicationConfiguration")
	return reconcile.Result{}, nil
}
