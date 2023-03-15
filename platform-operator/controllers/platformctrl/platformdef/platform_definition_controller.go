// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package platformdef

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/platformctrl/common"
	"k8s.io/apimachinery/pkg/types"
	"time"

	vzcontroller "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	platformapi "github.com/verrazzano/verrazzano/platform-operator/apis/platform/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PlatformDefinitionReconciler reconciles a Verrazzano PlatformDefinition object
type PlatformDefinitionReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
}

// Name of finalizer
const finalizerName = "platformdef.verrazzano.io"

// SetupWithManager creates a new controller and adds it to the manager
func (r *PlatformDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&platformapi.PlatformDefinition{}).Build(r)
	return err
}

// Reconcile the Verrazzano CR
// +kubebuilder:rbac:groups=platform.verrazzano.io,resources=platformdefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.verrazzano.io,resources=platformdefinitions/status,verbs=get;update;patch
func (r *PlatformDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// TODO: Metrics setup

	platformDef := &platformapi.PlatformDefinition{}
	if err := r.Get(ctx, req.NamespacedName, platformDef); err != nil {
		// TODO: errorCounterMetricObject.Inc()
		// If the resource is not found, that means all the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Platform resource: %v", err)
		return newRequeueWithDelay(), nil
	}

	// Get the resource logger
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           platformDef.Name,
		Namespace:      platformDef.Namespace,
		ID:             string(platformDef.UID),
		Generation:     platformDef.Generation,
		ControllerName: "platformcontroller",
	})
	if err != nil {
		// TODO: errorCounterMetricObject.Inc()
		zap.S().Errorf("Failed to create controller logger for PlatformDefinition controller: %v", err)
	}

	log.Infof("Reconciling platform definition %s/%s", platformDef.Namespace, platformDef.Name)

	// Check if resource is being deleted
	if !platformDef.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Oncef("Removing finalizer %s", finalizerName)
		platformDef.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(platformDef.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, platformDef); err != nil {
			return newRequeueWithDelay(), err
		}
		return ctrl.Result{}, nil
	}

	if !vzstring.SliceContainsString(platformDef.ObjectMeta.Finalizers, finalizerName) {
		log.Debugf("Adding finalizer %s", finalizerName)
		platformDef.ObjectMeta.Finalizers = append(platformDef.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(context.TODO(), platformDef); err != nil {
			return newRequeueWithDelay(), err
		}
	}

	if err := r.doReconcile(log, platformDef); err != nil {
		return newRequeueWithDelay(), err
	}

	// Update the platform status
	platformDef.Status.Version = platformDef.Spec.Version
	if err := r.Status().Update(context.TODO(), platformDef); err != nil {
		return newRequeueWithDelay(), err
	}

	log.Infof("Reconcile of platform definition %s/%s complete", platformDef.Namespace, platformDef.Name)
	return ctrl.Result{}, nil
}

func (r *PlatformDefinitionReconciler) doReconcile(log vzlog.VerrazzanoLogger, pd *platformapi.PlatformDefinition) error {
	moduleList := &platformapi.ModuleList{}
	if err := r.List(context.TODO(), moduleList); err != nil {
		return err
	}

	platformInstance := &platformapi.Platform{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: pd.Name, Namespace: pd.Namespace}, platformInstance); err != nil {
		log.ErrorfThrottled("Error getting platform instance %s/%s for platform definition", pd.Namespace, pd.Name)
		return err
	}
	for _, module := range moduleList.Items {
		pdModuleVersion, found := common.FindPlatformModuleVersion(log, module, pd)
		if !found {
			log.Progressf("Module %s/%s not found in PlatformDefinition %s/%s, ignoring", module.Namespace, module.Name, pd.Namespace, pd.Name)
			continue
		}
		if module.Spec.Source.Name != platformInstance.Name || module.Spec.Source.Namespace != platformInstance.Namespace {
			continue
		}
		if module.Status.Version != pdModuleVersion {
			module.Spec.Version = pdModuleVersion
			log.Progressf("Newer version %s detected for module %s/%s, upgrading", pdModuleVersion, module.Namespace, module.Name)
			updateModule := &platformapi.Module{}
			if err := r.Get(context.TODO(), types.NamespacedName{Namespace: module.Namespace, Name: module.Name}, updateModule); err != nil {
				return err
			}
			updateModule.Spec.Version = pdModuleVersion
			if err := r.Update(context.TODO(), updateModule); err != nil {
				return err
			}
		}
	}
	return nil
}

func newRequeueWithDelay() ctrl.Result {
	return vzcontroller.NewRequeueWithDelay(2, 5, time.Second)
}
