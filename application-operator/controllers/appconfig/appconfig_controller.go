// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"
	"fmt"
	"time"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	vzlog2 "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler struct {
	client.Client
	Log    *zap.SugaredLogger
	Scheme *runtime.Scheme
}

const (
	finalizerName  = "appconfig.finalizers.verrazzano.io"
	controllerName = "appconfig"
)

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&oamv1.ApplicationConfiguration{}).
		Complete(r)
}

// Reconcile checks restart version annotations on an ApplicationConfiguration and
// restarts applications as needed. When applications are restarted, the previous restart
// version annotation value is updated.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	counterMetricObject, errorCounterMetricObject, reconcileDurationMetricObject, zapLogForMetrics, err := metricsexporter.ExposeControllerMetrics(controllerName, metricsexporter.AppconfigReconcileCounter, metricsexporter.AppconfigReconcileError, metricsexporter.AppconfigReconcileDuration)
	if err != nil {
		return ctrl.Result{}, err
	}
	reconcileDurationMetricObject.TimerStart()
	defer reconcileDurationMetricObject.TimerStop()

	if req.Namespace == constants.KubeSystem {
		log := zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldController, controllerName)
		log.Infof("Application configuration resource %v should not be reconciled in kube-system namespace, ignoring", req.NamespacedName)
		return reconcile.Result{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var appConfig oamv1.ApplicationConfiguration
	if err := r.Client.Get(ctx, req.NamespacedName, &appConfig); err != nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}
	log, err := clusters.GetResourceLogger("applicationconfiguration", req.NamespacedName, &appConfig)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		log.Errorf("Failed to create controller logger for application configuration resource: %v", err)
		return clusters.NewRequeueWithDelay(), nil
	}
	log.Oncef("Reconciling application configuration resource %v, generation %v", req.NamespacedName, appConfig.Generation)

	res, err := r.doReconcile(ctx, &appConfig, log)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return clusters.NewRequeueWithDelay(), nil
	}

	// The Verrazzano resource has been reconciled.
	log.Oncef("Finished reconciling application configuration %v", req.NamespacedName)
	counterMetricObject.Inc(zapLogForMetrics, err)
	return ctrl.Result{}, nil
}

// doReconcile performs the reconciliation operations for the application configuration
func (r *Reconciler) doReconcile(ctx context.Context, appConfig *oamv1.ApplicationConfiguration, log vzlog2.VerrazzanoLogger) (ctrl.Result, error) {
	// the logic to delete cert/secret is moved to the ingress trait finalizer
	// but there could be apps deployed by older version of Verrazzano that are stuck being deleted, with finalizer
	// remove the finalizer
	if isAppConfigBeingDeleted(appConfig) {
		log.Debugf("Deleting application configuration %v", appConfig)
		if err := r.removeFinalizerIfRequired(ctx, appConfig, log); err != nil {
			return vzctrl.NewRequeueWithDelay(2, 3, time.Second), nil
		}
		return reconcile.Result{}, nil
	}

	// get the user-specified restart version - if it's missing then there's nothing to do here
	restartVersion, ok := appConfig.Annotations[constants.RestartVersionAnnotation]
	if !ok || len(restartVersion) == 0 {
		log.Debug("No restart version annotation found, nothing to do")
		return reconcile.Result{}, nil
	}

	// restart all workloads in the appconfig
	log.Debugf("Setting restart version %s for workloads in application %s", restartVersion, appConfig.Name)
	for _, wlStatus := range appConfig.Status.Workloads {
		if err := r.restartComponent(ctx, appConfig.Namespace, wlStatus, restartVersion, log); err != nil {
			return vzctrl.NewRequeueWithDelay(2, 3, time.Second), nil
		}
	}
	log.Debug("Successfully reconciled ApplicationConfiguration")
	return reconcile.Result{}, nil
}

// removeFinalizerIfRequired removes the finalizer from the application configuration if required
// The finalizer is only removed if the application configuration is being deleted and the finalizer had been added
func (r *Reconciler) removeFinalizerIfRequired(ctx context.Context, appConfig *oamv1.ApplicationConfiguration, log vzlog2.VerrazzanoLogger) error {
	if !appConfig.DeletionTimestamp.IsZero() && vzstring.SliceContainsString(appConfig.Finalizers, finalizerName) {
		appName := vznav.GetNamespacedNameFromObjectMeta(appConfig.ObjectMeta)
		log.Debugf("Removing finalizer from application configuration %s", appName)
		appConfig.Finalizers = vzstring.RemoveStringFromSlice(appConfig.Finalizers, finalizerName)
		err := r.Update(ctx, appConfig)
		return vzlog.ConflictWithLog(fmt.Sprintf("Failed to remove finalizer from application configuration %s", appName), err, zap.S())
	}
	return nil
}

func (r *Reconciler) restartComponent(ctx context.Context, wlNamespace string, wlStatus oamv1.WorkloadStatus, restartVersion string, log vzlog2.VerrazzanoLogger) error {
	// Get the workload as an unstructured object
	var wlName = wlStatus.Reference.Name
	var workload unstructured.Unstructured
	workload.SetAPIVersion(wlStatus.Reference.APIVersion)
	workload.SetKind(wlStatus.Reference.Kind)
	err := r.Client.Get(ctx, types.NamespacedName{Name: wlName, Namespace: wlNamespace}, &workload)
	if err != nil {
		log.Errorf("Failed getting workload component %s in namespace %s with restart-version %s: %v", wlName, wlNamespace, restartVersion, err)
		return err
	}
	// Set the annotation based on the workload kind
	switch workload.GetKind() {
	case constants.VerrazzanoCoherenceWorkloadKind:
		log.Debugf("Setting Coherence workload %s restart-version", wlName)
		return updateRestartVersion(ctx, r.Client, &workload, restartVersion, log)
	case constants.VerrazzanoWebLogicWorkloadKind:
		log.Debugf("Setting WebLogic workload %s restart-version", wlName)
		return updateRestartVersion(ctx, r.Client, &workload, restartVersion, log)
	case constants.VerrazzanoHelidonWorkloadKind:
		log.Debugf("Setting Helidon workload %s restart-version", wlName)
		return updateRestartVersion(ctx, r.Client, &workload, restartVersion, log)
	case constants.ContainerizedWorkloadKind:
		log.Debugf("Setting Containerized workload %s restart-version", wlName)
		return updateRestartVersion(ctx, r.Client, &workload, restartVersion, log)
	case constants.DeploymentWorkloadKind:
		log.Debugf("Setting Deployment workload %s restart-version", wlName)
		return r.restartDeployment(ctx, restartVersion, wlName, wlNamespace, log)
	case constants.StatefulSetWorkloadKind:
		log.Debugf("Setting StatefulSet workload %s restart-version", wlName)
		return r.restartStatefulSet(ctx, restartVersion, wlName, wlNamespace, log)
	case constants.DaemonSetWorkloadKind:
		log.Debugf("Setting DaemonSet workload %s restart-version", wlName)
		return r.restartDaemonSet(ctx, restartVersion, wlName, wlNamespace, log)
	default:
		log.Debugf("Skip marking restart-version for %s of kind %s in namespace %s", workload.GetName(), workload.GetKind(), wlNamespace)
	}
	return nil
}

func (r *Reconciler) restartDeployment(ctx context.Context, restartVersion, name, namespace string, log vzlog2.VerrazzanoLogger) error {
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

func (r *Reconciler) restartStatefulSet(ctx context.Context, restartVersion, name, namespace string, log vzlog2.VerrazzanoLogger) error {
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

func (r *Reconciler) restartDaemonSet(ctx context.Context, restartVersion, name, namespace string, log vzlog2.VerrazzanoLogger) error {
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

func DoRestartDeployment(ctx context.Context, client client.Client, restartVersion string, deployment *appsv1.Deployment, log vzlog2.VerrazzanoLogger) error {
	if deployment.Spec.Paused {
		return fmt.Errorf("deployment %s can't be restarted because it is paused", deployment.Name)
	}
	log.Debugf("The deployment %s/%s restart version is set to %s", deployment.Namespace, deployment.Name, restartVersion)
	_, err := controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		if len(restartVersion) > 0 {
			if deployment.Spec.Template.ObjectMeta.Annotations == nil {
				deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.ObjectMeta.Annotations[constants.RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	return vzlog.ConflictWithLog(fmt.Sprintf("Failed updating deployment %s/%s", deployment.Namespace, deployment.Name), err, zap.S())
}

func DoRestartStatefulSet(ctx context.Context, client client.Client, restartVersion string, statefulSet *appsv1.StatefulSet, log vzlog2.VerrazzanoLogger) error {
	log.Debugf("The statefulSet %s/%s restart version is set to %s", statefulSet.Namespace, statefulSet.Name, restartVersion)
	_, err := controllerutil.CreateOrUpdate(ctx, client, statefulSet, func() error {
		if len(restartVersion) > 0 {
			if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
				statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			statefulSet.Spec.Template.ObjectMeta.Annotations[constants.RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	return vzlog.ConflictWithLog(fmt.Sprintf("Conflict updating statefulSet %s/%s:", statefulSet.Namespace, statefulSet.Name), err, zap.S())
}

func DoRestartDaemonSet(ctx context.Context, client client.Client, restartVersion string, daemonSet *appsv1.DaemonSet, log vzlog2.VerrazzanoLogger) error {
	log.Debugf("The daemonSet %s/%s restart version is set to %s", daemonSet.Namespace, daemonSet.Name, restartVersion)
	_, err := controllerutil.CreateOrUpdate(ctx, client, daemonSet, func() error {
		if len(restartVersion) > 0 {
			if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
				daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			daemonSet.Spec.Template.ObjectMeta.Annotations[constants.RestartVersionAnnotation] = restartVersion
		}
		return nil
	})
	return vzlog.ConflictWithLog(fmt.Sprintf("Conflict updating daemonSet %s/%s:", daemonSet.Namespace, daemonSet.Name), err, zap.S())
}

// Update the workload annotation with the restart version. This will cause the workload to be restarted if the version changed
func updateRestartVersion(ctx context.Context, client client.Client, u *unstructured.Unstructured, restartVersion string, log vzlog2.VerrazzanoLogger) error {
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
		annotations[constants.RestartVersionAnnotation] = restartVersion
		err = unstructured.SetNestedStringMap(u.Object, annotations, metaAnnotationFields...)
		if err != nil {
			log.Errorf("Failed setting NestedStringMap for workload %s: %v", u.GetName(), err)
			return err
		}
		return nil
	})
	err = vzlog.ConflictWithLog(fmt.Sprintf("Failed to update restart version for workload %s/%s", u.GetNamespace(), u.GetName()), err, zap.S())
	return err
}

// isAppConfigBeingDeleted determines if the app config is in the process of being deleted.
// This is done checking for a non-nil deletion timestamp.
func isAppConfigBeingDeleted(appConfig *oamv1.ApplicationConfiguration) bool {
	return appConfig != nil && appConfig.GetDeletionTimestamp() != nil
}
