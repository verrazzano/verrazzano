// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package containerizedworkload

import (
	"context"
	errors "errors"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/verrazzano/verrazzano/application-operator/controllers/appconfig"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	vzlog2 "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
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

const controllerName = "containerizedworkload"

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&oamv1.ContainerizedWorkload{}).
		Complete(r)
}

// Reconcile checks restart version annotations on an ContainerizedWorkload and
// restarts as needed.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		return ctrl.Result{}, errors.New("context cannot be nil")
	}

	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	if req.Namespace == vzconst.KubeSystem {
		log := zap.S().With(vzlogInit.FieldResourceNamespace, req.Namespace, vzlogInit.FieldResourceName, req.Name, vzlogInit.FieldController, controllerName)
		log.Infof("Containerized workload resource %v should not be reconciled in kube-system namespace, ignoring", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	var workload oamv1.ContainerizedWorkload
	if err := r.Client.Get(ctx, req.NamespacedName, &workload); err != nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}
	log, err := clusters.GetResourceLogger("containerizedworkload", req.NamespacedName, &workload)
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for containerized workload resource: %v", err)
		return clusters.NewRequeueWithDelay(), nil
	}
	log.Oncef("Reconciling containerized workload resource %v, generation %v", req.NamespacedName, workload.Generation)

	res, err := r.doReconcile(ctx, workload, log)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		return clusters.NewRequeueWithDelay(), nil
	}

	log.Oncef("Finished reconciling containerized workload %v", req.NamespacedName)

	return ctrl.Result{}, nil
}

// doReconcile performs the reconciliation operations for the ContainerizedWorkload
func (r *Reconciler) doReconcile(ctx context.Context, workload oamv1.ContainerizedWorkload, log vzlog2.VerrazzanoLogger) (ctrl.Result, error) {
	// Label the service with the OAM app and component, if the service exists. Errors will be logged in the method, we still
	// need to process restart annotation even if there's an error
	r.updateServiceLabels(ctx, workload, log)

	// get the user-specified restart version - if it's missing then there's nothing to do here
	restartVersion, ok := workload.Annotations[vzconst.RestartVersionAnnotation]
	if !ok || len(restartVersion) == 0 {
		log.Debug("No restart version annotation found, nothing to do")
		return reconcile.Result{}, nil
	}

	if err := r.restartWorkload(ctx, restartVersion, &workload, log); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) restartWorkload(ctx context.Context, restartVersion string, workload *oamv1.ContainerizedWorkload, log vzlog2.VerrazzanoLogger) error {
	log.Debugf("Marking container %s with restart-version %s", workload.Name, restartVersion)
	var deploymentList appsv1.DeploymentList
	componentNameReq, _ := labels.NewRequirement(oam.LabelAppComponent, selection.Equals, []string{workload.ObjectMeta.Labels[oam.LabelAppComponent]})
	appNameReq, _ := labels.NewRequirement(oam.LabelAppName, selection.Equals, []string{workload.ObjectMeta.Labels[oam.LabelAppName]})
	selector := labels.NewSelector()
	selector = selector.Add(*componentNameReq, *appNameReq)
	err := r.Client.List(ctx, &deploymentList, &client.ListOptions{Namespace: workload.Namespace, LabelSelector: selector})
	if err != nil {
		return err
	}
	for index := range deploymentList.Items {
		deployment := &deploymentList.Items[index]
		if err := appconfig.DoRestartDeployment(ctx, r.Client, restartVersion, deployment, log); err != nil {
			return err
		}
	}
	return nil
}

// updateServiceLabels looks up the Service associated with the workload, and updates its OAM
// app and component labels if needed. Any errors will be logged and not returned since we don't want
// this to fail anything.
func (r *Reconciler) updateServiceLabels(ctx context.Context, workload oamv1.ContainerizedWorkload, log vzlog2.VerrazzanoLogger) {
	svc, err := r.getWorkloadService(ctx, workload, log)
	if err != nil {
		return
	}
	if svc == nil {
		return
	}
	if svc.Labels == nil {
		svc.Labels = map[string]string{}
	}
	if svc.Labels[oam.LabelAppName] == workload.Labels[oam.LabelAppName] &&
		svc.Labels[oam.LabelAppComponent] == workload.Labels[oam.LabelAppComponent] {
		// nothing to do, return
		return
	}
	log.Infof("Updating service OAM app and component labels for %s/%s", svc.Namespace, svc.Name)
	svc.Labels[oam.LabelAppName] = workload.Labels[oam.LabelAppName]
	svc.Labels[oam.LabelAppComponent] = workload.Labels[oam.LabelAppComponent]
	err = r.Update(ctx, svc)
	if err != nil {
		log.Errorf("Failed to update Service %s for ContainerizedWorkload %s/%s: %v", svc.Name, workload.Namespace, workload.Name, err)
	}
}

// getWorkloadService retrieves the Service associated with the workload
func (r *Reconciler) getWorkloadService(ctx context.Context, workload oamv1.ContainerizedWorkload, log vzlog2.VerrazzanoLogger) (*corev1.Service, error) {
	service := corev1.Service{}
	svcName := ""
	for _, res := range workload.Status.Resources {
		if res.Kind == service.Kind && res.APIVersion == service.APIVersion {
			svcName = res.Name
		}
	}
	if svcName == "" {
		log.Errorf("Service does not exist in status of ContainerizedWorkload %s/%s", workload.Namespace, workload.Name)
		return nil, nil
	}
	svc := corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: workload.Namespace}, &svc); err != nil {
		log.Errorf("Failed to retrieve Service %s for ContainerizedWorkload %s/%s: %v", svcName, workload.Namespace, workload.Name, err)
		return nil, err
	}
	return &svc, nil
}
