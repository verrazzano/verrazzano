// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
	RestartVersionAnnotation = "verrazzano.io/restart-version"
)

var containerAnnotationsFields = []string{"metadata", "annotations"}

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
	if !ok || len(restartVersion) == 0 {
		log.Info("No restart version annotation found, nothing to do")
		return reconcile.Result{}, nil
	}

	// restart all components in the appconfig
	log.Info(fmt.Sprintf("Reconciling application with restart-version %s", restartVersion))
	for index := range appConfig.Spec.Components {
		componentName := appConfig.Spec.Components[index].ComponentName
		componentNamespace := appConfig.Namespace
		log.Info(fmt.Sprintf("Marking component %s in namespace %s with restart-version %s", componentName, componentNamespace, restartVersion))
		err := r.restartComponent(ctx, componentName, componentNamespace, restartVersion, log)
		if err != nil {
			log.Error(err, fmt.Sprintf("Enountered error marking component %s in namespace %swith restart-version %s", componentName, componentNamespace, restartVersion))
			return reconcile.Result{}, err
		}
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
		// passs "verrazzano.io/restart-version" to VerrazzanoCoherenceWorkload
		err = addAnnotation(workload, restartVersion)
		if err != nil {
			return err
		}
	case "VerrazzanoWebLogicWorkload":
		// passs "verrazzano.io/restart-version" to VerrazzanoWebLogicWorkload
		err = addAnnotation(workload, restartVersion)
		if err != nil {
			return err
		}
	case "VerrazzanoHelidonWorkload":
		// passs "verrazzano.io/restart-version" to VerrazzanoHelidonWorkload
		err = addAnnotation(workload, restartVersion)
		if err != nil {
			return err
		}
	case "ContainerizedWorkload":
		// passs "verrazzano.io/restart-version" to ContainerizedWorkload
		err = addAnnotation(workload, restartVersion)
		if err != nil {
			return err
		}
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
		log.Info(fmt.Sprintf("Skip marking restart-version for %s of kind %s in namespace %s", workload.GetName(), workload.GetKind(), componentNamespace))
	}

	return nil
}

func addAnnotation(workload *unstructured.Unstructured, restartVersion string) error {
	annotations, found, _ := unstructured.NestedStringMap(workload.Object, containerAnnotationsFields...)
	if !found {
		annotations = map[string]string{}
	}
	annotations[RestartVersionAnnotation] = restartVersion
	return unstructured.SetNestedStringMap(workload.Object, annotations, containerAnnotationsFields...)
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
	log.Info(fmt.Sprintf("Marking deployment %s in namespace %s with restart-version %s", name, namespace, restartVersion))
	return DoRestartDeployment(ctx, r.Client, restartVersion, &deployment, log)
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
	log.Info(fmt.Sprintf("Marking statefulSet %s in namespace %s with restart-version %s", name, namespace, restartVersion))
	return DoRestartStatefulSet(ctx, r.Client, restartVersion, &statefulSet, log)
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
	log.Info(fmt.Sprintf("Marking daemonSet %s in namespace %s with restart-version %s", name, namespace, restartVersion))
	return DoRestartDaemonSet(ctx, r.Client, restartVersion, &daemonSet, log)
}

func DoRestartDeployment(ctx context.Context, client client.Client, restartVersion string, deployment *appsv1.Deployment, log logr.Logger) error {
	if deployment.Spec.Paused {
		return fmt.Errorf("deployment %s can't be restarted because it is paused", deployment.Name)
	}
	log.Info(fmt.Sprintf("The deployment %s/%s restart version is set to %s", deployment.Namespace, deployment.Name, restartVersion))
	_, err := controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		if len(restartVersion) > 0 {
			if deployment.Spec.Template.ObjectMeta.Annotations == nil {
				deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.ObjectMeta.Annotations[RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	if err != nil {
		log.Error(err, fmt.Sprintf("Error updating deployment %s/%s", deployment.Namespace, deployment.Name))
		return err
	}
	return nil
}

func DoRestartStatefulSet(ctx context.Context, client client.Client, restartVersion string, statefulSet *appsv1.StatefulSet, log logr.Logger) error {
	log.Info(fmt.Sprintf("The statefulSet %s/%s restart version is set to %s", statefulSet.Namespace, statefulSet.Name, restartVersion))
	_, err := controllerutil.CreateOrUpdate(ctx, client, statefulSet, func() error {
		if len(restartVersion) > 0 {
			if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
				statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			statefulSet.Spec.Template.ObjectMeta.Annotations[RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	if err != nil {
		log.Error(err, fmt.Sprintf("Error updating statefulSet %s/%s", statefulSet.Namespace, statefulSet.Name))
		return err
	}
	return nil
}

func DoRestartDaemonSet(ctx context.Context, client client.Client, restartVersion string, daemonSet *appsv1.DaemonSet, log logr.Logger) error {
	log.Info(fmt.Sprintf("The daemonSet %s/%s restart version is set to %s", daemonSet.Namespace, daemonSet.Name, restartVersion))
	_, err := controllerutil.CreateOrUpdate(ctx, client, daemonSet, func() error {
		if len(restartVersion) > 0 {
			if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
				daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			daemonSet.Spec.Template.ObjectMeta.Annotations[RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	if err != nil {
		log.Error(err, fmt.Sprintf("Error updating daemonSet %s/%s", daemonSet.Namespace, daemonSet.Name))
		return err
	}
	return nil
}
