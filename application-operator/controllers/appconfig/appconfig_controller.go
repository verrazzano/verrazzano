// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/application-operator/controllers/ingresstrait"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler struct {
	client.Client
	Log    *zap.SugaredLogger
	Scheme *runtime.Scheme
}

const finalizerName = "appconfig.finalizers.verrazzano.io"

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
	log := r.Log.With("applicationconfiguration", req.NamespacedName)
	log.Info("Reconciling ApplicationConfiguration")
	nsn := types.NamespacedName{Name: req.Name, Namespace: req.Namespace}

	// fetch the appconfig
	var appConfig oamv1.ApplicationConfiguration
	if err := r.Client.Get(ctx, req.NamespacedName, &appConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debug("ApplicationConfiguration has been deleted")
		} else {
			log.Errorf("Failed to fetch ApplicationConfiguration: %v", err)
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// If the application configuration no longer exists or is being deleted then cleanup the associated cert and secret resources
	if isAppConfigBeingDeleted(&appConfig) {
		r.Log.Debugf("App Configuration is being deleted %s", nsn.Name)
		if err := ingresstrait.Cleanup(nsn, r.Client, r.Log); err != nil {
			return reconcile.Result{}, err
		}
		// resource cleanup has succeeded, remove the finalizer
		if err := r.removeFinalizerIfRequired(ctx, &appConfig); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// add finalizer
	if err := r.addFinalizerIfRequired(ctx, &appConfig); err != nil {
		return reconcile.Result{}, err
	}

	// get the user-specified restart version - if it's missing then there's nothing to do here
	restartVersion, ok := appConfig.Annotations[vzconst.RestartVersionAnnotation]
	if !ok || len(restartVersion) == 0 {
		log.Debug("No restart version annotation found, nothing to do")
		return reconcile.Result{}, nil
	}

	// restart all workloads in the appconfig
	log.Debugf("Setting restart version %s for workloads in application %s", restartVersion, appConfig.Name)
	for _, wlStatus := range appConfig.Status.Workloads {
		err := r.restartComponent(ctx, appConfig.Namespace, wlStatus, restartVersion, log)
		if err != nil {
			log.Errorf("Failed marking component %s in namespace %s with restart-version %s: %v", wlStatus.ComponentName, appConfig.Namespace, restartVersion, err)
			return reconcile.Result{}, err
		}
	}
	log.Debug("Successfully reconciled ApplicationConfiguration")
	return reconcile.Result{}, nil
}

func (r *Reconciler) restartComponent(ctx context.Context, wlNamespace string, wlStatus oamv1.WorkloadStatus, restartVersion string, log *zap.SugaredLogger) error {
	// Get the workload as an unstructured object
	var wlName = wlStatus.Reference.Name
	var workload unstructured.Unstructured
	workload.SetAPIVersion(wlStatus.Reference.APIVersion)
	workload.SetKind(wlStatus.Reference.Kind)
	err := r.Client.Get(ctx, types.NamespacedName{Name: wlName, Namespace: wlNamespace}, &workload)
	if err != nil {
		return err
	}
	// Set the annotation based on the workload kind
	switch workload.GetKind() {
	case vzconst.VerrazzanoCoherenceWorkloadKind:
		log.Debugf("Setting Coherence workload %s restart-version", wlName)
		return updateRestartVersion(ctx, r, &workload, restartVersion, log)
	case vzconst.VerrazzanoWebLogicWorkloadKind:
		log.Debugf("Setting WebLogic workload %s restart-version", wlName)
		return updateRestartVersion(ctx, r, &workload, restartVersion, log)
	case vzconst.VerrazzanoHelidonWorkloadKind:
		log.Debugf("Setting Helidon workload %s restart-version", wlName)
		return updateRestartVersion(ctx, r, &workload, restartVersion, log)
	case vzconst.ContainerizedWorkloadKind:
		log.Debugf("Setting Containerized workload %s restart-version", wlName)
		return updateRestartVersion(ctx, r, &workload, restartVersion, log)
	case vzconst.DeploymentWorkloadKind:
		log.Debugf("Setting Deployment workload %s restart-version", wlName)
		return r.restartDeployment(ctx, restartVersion, wlName, wlNamespace, log)
	case vzconst.StatefulSetWorkloadKind:
		log.Debugf("Setting StatefulSet workload %s restart-version", wlName)
		return r.restartStatefulSet(ctx, restartVersion, wlName, wlNamespace, log)
	case vzconst.DaemonSetWorkloadKind:
		log.Debugf("Setting DaemonSet workload %s restart-version", wlName)
		return r.restartDaemonSet(ctx, restartVersion, wlName, wlNamespace, log)
	default:
		log.Debugf("Skip marking restart-version for %s of kind %s in namespace %s", workload.GetName(), workload.GetKind(), wlNamespace)
	}
	return nil
}

func (r *Reconciler) restartDeployment(ctx context.Context, restartVersion string, name, namespace string, log *zap.SugaredLogger) error {
	var deployment = appsv1.Deployment{}
	deploymentKey := types.NamespacedName{Name: name, Namespace: namespace}
	if err := r.Get(ctx, deploymentKey, &deployment); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debugf("Can not find deployment %s in namespace %s", name, namespace)
		} else {
			log.Errorf("Failed to obtain deployment %s in namespace %s: %v", name, namespace, err)
			return err
		}
	}
	log.Debugf("Marking deployment %s in namespace %s with restart-version %s", name, namespace, restartVersion)
	return DoRestartDeployment(ctx, r.Client, restartVersion, &deployment, log)
}

func (r *Reconciler) restartStatefulSet(ctx context.Context, restartVersion string, name, namespace string, log *zap.SugaredLogger) error {
	var statefulSet = appsv1.StatefulSet{}
	statefulSetKey := types.NamespacedName{Name: name, Namespace: namespace}
	if err := r.Get(ctx, statefulSetKey, &statefulSet); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debugf("Can not find statefulSet %s in namespace %s", name, namespace)
		} else {
			log.Errorf("Failed to obtain statefulSet %s in namespace %s: %v", name, namespace, err)
			return err
		}
	}
	log.Debugf("Marking statefulSet %s in namespace %s with restart-version %s", name, namespace, restartVersion)
	return DoRestartStatefulSet(ctx, r.Client, restartVersion, &statefulSet, log)
}

func (r *Reconciler) restartDaemonSet(ctx context.Context, restartVersion string, name, namespace string, log *zap.SugaredLogger) error {
	var daemonSet = appsv1.DaemonSet{}
	daemonSetKey := types.NamespacedName{Name: name, Namespace: namespace}
	if err := r.Get(ctx, daemonSetKey, &daemonSet); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debugf("Can not find daemonSet %s in namespace %s", name, namespace)
		} else {
			log.Errorf("Failed to obtain daemonSet %s in namespace %s: %v", name, namespace, err)
			return err
		}
	}
	log.Debugf("Marking daemonSet %s in namespace %s with restart-version %s", name, namespace, restartVersion)
	return DoRestartDaemonSet(ctx, r.Client, restartVersion, &daemonSet, log)
}

// removeFinalizerIfRequired removes the finalizer from the application configuration if required
// The finalizer is only removed if the application configuration is being deleted and the finalizer had been added
func (r *Reconciler) removeFinalizerIfRequired(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
	if !appConfig.DeletionTimestamp.IsZero() && vzstring.SliceContainsString(appConfig.Finalizers, finalizerName) {
		appName := vznav.GetNamespacedNameFromObjectMeta(appConfig.ObjectMeta)
		r.Log.Debugf("Removing finalizer from application configuration %s", appName)
		appConfig.Finalizers = vzstring.RemoveStringFromSlice(appConfig.Finalizers, finalizerName)
		if err := r.Update(ctx, appConfig); err != nil {
			r.Log.Errorf("Failed to remove finalizer from application configuration %s: %v", appName, err)
			return err
		}
	}
	return nil
}

// addFinalizerIfRequired adds the finalizer to the app config if required
// The finalizer is only added if the app config is not being deleted and the finalizer has not previously been added
func (r *Reconciler) addFinalizerIfRequired(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
	if appConfig.GetDeletionTimestamp().IsZero() && !vzstring.SliceContainsString(appConfig.Finalizers, finalizerName) {
		appName := vznav.GetNamespacedNameFromObjectMeta(appConfig.ObjectMeta)
		r.Log.Debugf("Adding finalizer for appConfig %s", appName)
		appConfig.Finalizers = append(appConfig.Finalizers, finalizerName)
		if err := r.Update(ctx, appConfig); err != nil {
			r.Log.Errorf("Failed to add finalizer to appConfig %s", appName)
			return err
		}
	}
	return nil
}

func DoRestartDeployment(ctx context.Context, client client.Client, restartVersion string, deployment *appsv1.Deployment, log *zap.SugaredLogger) error {
	if deployment.Spec.Paused {
		return fmt.Errorf("deployment %s can't be restarted because it is paused", deployment.Name)
	}
	log.Debugf("The deployment %s/%s restart version is set to %s", deployment.Namespace, deployment.Name, restartVersion)
	_, err := controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		if len(restartVersion) > 0 {
			if deployment.Spec.Template.ObjectMeta.Annotations == nil {
				deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	if err != nil {
		log.Errorf("Failed updating deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
		return err
	}
	return nil
}

func DoRestartStatefulSet(ctx context.Context, client client.Client, restartVersion string, statefulSet *appsv1.StatefulSet, log *zap.SugaredLogger) error {
	log.Debugf("The statefulSet %s/%s restart version is set to %s", statefulSet.Namespace, statefulSet.Name, restartVersion)
	_, err := controllerutil.CreateOrUpdate(ctx, client, statefulSet, func() error {
		if len(restartVersion) > 0 {
			if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
				statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			statefulSet.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	if err != nil {
		log.Errorf("Failed updating statefulSet %s/%s: %v", statefulSet.Namespace, statefulSet.Name, err)
		return err
	}
	return nil
}

func DoRestartDaemonSet(ctx context.Context, client client.Client, restartVersion string, daemonSet *appsv1.DaemonSet, log *zap.SugaredLogger) error {
	log.Debugf("The daemonSet %s/%s restart version is set to %s", daemonSet.Namespace, daemonSet.Name, restartVersion)
	_, err := controllerutil.CreateOrUpdate(ctx, client, daemonSet, func() error {
		if len(restartVersion) > 0 {
			if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
				daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			daemonSet.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	if err != nil {
		log.Errorf("Failed updating daemonSet %s/%s: %v", daemonSet.Namespace, daemonSet.Name, err)
		return err
	}
	return nil
}

// Update the workload annotation with the restart version. This will cause the workload to be restarted if the version changed
func updateRestartVersion(ctx context.Context, client client.Client, u *unstructured.Unstructured, restartVersion string, log *zap.SugaredLogger) error {
	const metadataField = "metadata"
	var metaAnnotationFields = []string{metadataField, "annotations"}

	log.Debugf("Setting workload %s restartVersion to %s", u.GetName(), restartVersion)
	_, err := controllerutil.CreateOrUpdate(ctx, client, u, func() error {
		annotations, found, err := unstructured.NestedStringMap(u.Object, metaAnnotationFields...)
		if err != nil {
			log.Errorf("Failed getting NestedStringMap for workload %s: %v", u.GetName(), err)
			return err
		}
		if !found {
			annotations = map[string]string{}
		}
		annotations[vzconst.RestartVersionAnnotation] = restartVersion
		err = unstructured.SetNestedStringMap(u.Object, annotations, metaAnnotationFields...)
		if err != nil {
			log.Errorf("Failed setting NestedStringMap for workload %s: %v", u.GetName(), err)
			return err
		}
		return nil
	})
	return err
}

// isAppConfigBeingDeleted determines if the app config is in the process of being deleted.
// This is done checking for a non-nil deletion timestamp.
func isAppConfigBeingDeleted(appConfig *oamv1.ApplicationConfiguration) bool {
	return appConfig != nil && appConfig.GetDeletionTimestamp() != nil
}
