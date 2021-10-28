// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"
	"fmt"

	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	RestartVersionAnnotation         = "verrazzano.io/restart-version"
	observedRestartVersionAnnotation = "verrazzano.io/observed-restart-version"
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
			log.Info("ApplicationConfiguration has been deleted")
		} else {
			log.Error(err, "Failed to fetch ApplicationConfiguration")
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// get the user-specified restart version - if it's missing then there's nothing to do here
	restartVersion, ok := appConfig.Annotations[RestartVersionAnnotation]
	if !ok {
		log.Info("No restart version annotation found, nothing to do")
		return reconcile.Result{}, nil
	}

	// get the annotation with the previous restart version - if it's missing or the versions do not
	// match, then we restart apps
	hasErrorsRestarting := false
	if r.isMarkedForRestart(restartVersion, appConfig.Annotations) {
		log.Info(fmt.Sprintf("Restarting application with restart-version %s", restartVersion))

		// restart all components
		for index := range appConfig.Spec.Components {
			componentName := appConfig.Spec.Components[index].ComponentName
			componentNamespace := appConfig.Namespace
			log.Info(fmt.Sprintf("Restarting component %s in namespace %s with restart-version %s", componentName, componentNamespace, restartVersion))
			err := r.restartComponent(ctx, componentName, componentNamespace, restartVersion, log)
			if err != nil {
				log.Error(err, fmt.Sprintf("Enountered error restarting component %s in namespace %swith restart-version %s", componentName, componentNamespace, restartVersion))
				hasErrorsRestarting = true
			}
		}

		// add/update the previous restart version annotation on the appconfig if the restarting was successful
		if !hasErrorsRestarting {
			appConfig.Annotations[observedRestartVersionAnnotation] = restartVersion
			log.Info(fmt.Sprintf("Setting observed-restart-version with %s", restartVersion))
			if err := r.Client.Update(ctx, &appConfig); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			log.Info(fmt.Sprintf("Encountered errors restarting with %s.  Retry in the next Reconcile", restartVersion))
		}

		return reconcile.Result{}, nil
	}

	log.Info("Successfully reconciled ApplicationConfiguration")
	return reconcile.Result{}, nil
}

func (r *Reconciler) restartComponent(ctx context.Context, componentName, componentNamespace string, restartVersion string, log logr.Logger) error {
	var component oamv1.Component
	err := r.Client.Get(ctx, types.NamespacedName{Name: componentName, Namespace: componentNamespace}, &component)
	if err != nil {
		return err
	}

	workload, err := vznav.ConvertRawExtensionToUnstructured(&component.Spec.Workload)
	if err != nil {
		return err
	}

	switch workload.GetKind() {
	case "VerrazzanoCoherenceWorkload":
		// "verrazzano.io/restart-version" will be automatically set on VerrazzanoCoherenceWorkload
		// VerrazzanoCoherenceWorkload reconciler processes the annotation on its own
		// nothing needs to be done here
	case "VerrazzanoWebLogicWorkload":
		// "verrazzano.io/restart-version" will be automatically set on VerrazzanoWebLogicWorkload
		// VerrazzanoWebLogicWorkload reconciler processes the annotation on its own
		// nothing needs to be done here
	case "VerrazzanoHelidonWorkload":
		// "verrazzano.io/restart-version" will be automatically set on VerrazzanoHelidonWorkload
		// VerrazzanoHelidonWorkload reconciler processes the annotation on its own
		// nothing needs to be done here
	case "Deployment":
		err = r.restartDeployment(ctx, restartVersion, workload.GetName(), componentNamespace, log)
		if err != nil {
			return err
		}
	case "StatefulSet":
		err = r.restartStatefulSet(ctx, restartVersion, workload.GetName(), componentNamespace, log)
		if err != nil {
			return err
		}
	case "DaemonSet":
		err = r.restartDaemonSet(ctx, restartVersion, workload.GetName(), componentNamespace, log)
		if err != nil {
			return err
		}
	default:
		log.Info(fmt.Sprintf("Skip restarting for %s of kind %s in namespace %s", workload.GetName(), workload.GetKind(), componentNamespace))
	}

	return nil
}

func (r *Reconciler) isMarkedForRestart(restartVersion string, annotations map[string]string) bool {
	observedRestartVersion, ok := annotations[observedRestartVersionAnnotation]
	if !ok || restartVersion != observedRestartVersion {
		return true
	}
	return false
}

func (r *Reconciler) restartDeployment(ctx context.Context, restartVersion string, name, namespace string, log logr.Logger) error {
	var deployment = appsv1.Deployment{}
	deploymentKey := types.NamespacedName{Name: name, Namespace: namespace}
	if err := r.Get(ctx, deploymentKey, &deployment); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Can not find deployment %s in namespace %s", name, namespace))
		} else {
			log.Error(err, fmt.Sprintf("An error occurred trying to obtain deployment %s in namespace %s", name, namespace))
			return err
		}
	}
	if r.isMarkedForRestart(restartVersion, deployment.Annotations) {
		log.Info(fmt.Sprintf("Restarting deployment %s in namespace %s with restart-version %s", name, namespace, restartVersion))
		return DoRestartDeployment(r.Client, restartVersion, &deployment, log)
	}
	return nil
}

func (r *Reconciler) restartStatefulSet(ctx context.Context, restartVersion string, name, namespace string, log logr.Logger) error {
	var statefulSet = appsv1.StatefulSet{}
	statefulSetKey := types.NamespacedName{Name: name, Namespace: namespace}
	if err := r.Get(ctx, statefulSetKey, &statefulSet); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Can not find statefulSet %s in namespace %s", name, namespace))
		} else {
			log.Error(err, fmt.Sprintf("An error occurred trying to obtain statefulSet %s in namespace %s", name, namespace))
			return err
		}
	}
	if r.isMarkedForRestart(restartVersion, statefulSet.Annotations) {
		log.Info(fmt.Sprintf("Restarting statefulSet %s in namespace %s with restart-version %s", name, namespace, restartVersion))
		return DoRestartStatefulSet(r.Client, restartVersion, &statefulSet, log)
	}
	return nil
}

func (r *Reconciler) restartDaemonSet(ctx context.Context, restartVersion string, name, namespace string, log logr.Logger) error {
	var daemonSet = appsv1.DaemonSet{}
	daemonSetKey := types.NamespacedName{Name: name, Namespace: namespace}
	if err := r.Get(ctx, daemonSetKey, &daemonSet); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Can not find daemonSet %s in namespace %s", name, namespace))
		} else {
			log.Error(err, fmt.Sprintf("An error occurred trying to obtain daemonSet %s in namespace %s", name, namespace))
			return err
		}
	}
	if r.isMarkedForRestart(restartVersion, daemonSet.Annotations) {
		log.Info(fmt.Sprintf("Restarting daemonSet %s in namespace %s with restart-version %s", name, namespace, restartVersion))
		return DoRestartDaemonSet(r.Client, restartVersion, &daemonSet, log)
	}
	return nil
}

func DoRestartDeployment(client client.Client, restartVersion string, deployment *appsv1.Deployment, log logr.Logger) error {
	if deployment.Spec.Paused {
		return fmt.Errorf("deployment %s can't be restarted because it is paused", deployment.Name)
	}
	log.Info(fmt.Sprintf("The deployment %s/%s restart version is set to %s", deployment.Namespace, deployment.Name, restartVersion))
	if deployment.Spec.Template.ObjectMeta.Annotations == nil {
		deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.ObjectMeta.Annotations[RestartVersionAnnotation] = restartVersion
	if err := client.Update(context.TODO(), deployment); err != nil {
		return err
	}
	return nil
}

func DoRestartStatefulSet(client client.Client, restartVersion string, statefulSet *appsv1.StatefulSet, log logr.Logger) error {
	log.Info(fmt.Sprintf("The statefulSet %s/%s restart version is set to %s", statefulSet.Namespace, statefulSet.Name, restartVersion))
	if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
		statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	statefulSet.Spec.Template.ObjectMeta.Annotations[RestartVersionAnnotation] = restartVersion
	if err := client.Update(context.TODO(), statefulSet); err != nil {
		return err
	}
	return nil
}

func DoRestartDaemonSet(client client.Client, restartVersion string, daemonSet *appsv1.DaemonSet, log logr.Logger) error {
	log.Info(fmt.Sprintf("The daemonSet %s/%s restart version is set to %s", daemonSet.Namespace, daemonSet.Name, restartVersion))
	if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
		daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	daemonSet.Spec.Template.ObjectMeta.Annotations[RestartVersionAnnotation] = restartVersion
	if err := client.Update(context.TODO(), daemonSet); err != nil {
		return err
	}
	return nil
}
