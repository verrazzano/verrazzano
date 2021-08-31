package verrazzano

import (
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a Verrazzano object
type ComponentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
	DryRun     bool
}

func (c *ComponentReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

// SetupWithManager creates a new controller and adds it to the manager
func (c *ComponentReconciler) SetupControllerWithManager(mgr ctrl.Manager) error {
	var err error
	c.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.VerrazzanoComponent{}).
		// The GenerateChangedPredicate will skip update events that have no change in the object's metadata.generation
		// field.  Any updates to the status or metadata do not cause the metadata.generation to be changed and
		// therefore the reconciler will not be called.
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Build(c)
	return err
}
