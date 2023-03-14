package platformdef

import (
	"context"
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
		ControllerName: "platform",
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

	// Update the platform status
	platformDef.Status.Version = platformDef.Spec.Version
	if err := r.Status().Update(context.TODO(), platformDef); err != nil {
		return newRequeueWithDelay(), err
	}

	log.Infof("Reconcile of platform definition %s/%s complete", platformDef.Namespace, platformDef.Name)
	return ctrl.Result{}, nil
}

func newRequeueWithDelay() ctrl.Result {
	return vzcontroller.NewRequeueWithDelay(2, 5, time.Second)
}
