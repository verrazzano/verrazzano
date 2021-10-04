package verrazzano

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// Reconciler reconciles a Verrazzano object
type ComponentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
	DryRun     bool
}

func (c *ComponentReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	log := zap.S().With("resource", fmt.Sprintf("%s:%s", req.Namespace, req.Name))

	log.Info("Reconciler called")

	var vzcomp = &vzapi.VerrazzanoComponent{}
	if err := c.Get(ctx, req.NamespacedName, vzcomp); err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error getting the verrazzano resource - don't requeue
		log.Errorf("Failed to fetch verrazzano component resource %s/%s: %v", err)
		return reconcile.Result{}, err
	}

	compContext := spi.NewContext(log, c, &vzapi.Verrazzano{Spec: vzcomp.Spec.Configuration}, c.DryRun)

	ok, comp := registry.FindComponent(vzcomp.Spec.ComponentName)
	if !ok {
		return reconcile.Result{}, fmt.Errorf("Unable to find component %s", vzcomp.Spec.ComponentName)
	}

	// Loop through all of the Verrazzano components and upgrade each one sequentially for now; will parallelize later
	//for _, comp := range registry.GetComponents() {
	if !comp.IsOperatorInstallSupported() {
		return reconcile.Result{}, nil
	}

	componentState := vzcomp.Status.State

	switch componentState {
	case vzapi.Ready:
		// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
		return reconcile.Result{}, nil
	case vzapi.Disabled:
		if !isVerrazzanoComponentEnabled(vzcomp, comp.Name()) {
			// User has disabled component in Verrazzano CR, don't install
			return reconcile.Result{}, nil
		}
		if err := c.updateComponentStatus(log, vzcomp, "PreInstall started", vzapi.PreInstall); err != nil {
			return ctrl.Result{Requeue: true}, err
		}

	case vzapi.PreInstalling:
		log.Infof("PreInstalling component %s", comp.Name())
		if !registry.ComponentDependenciesMet(comp, compContext) {
			log.Infof("Dependencies not met for %s: %v", comp.Name(), comp.GetDependencies())
			return newRequeueWithDelay(), nil
		}
		if err := comp.PreInstall(compContext); err != nil {
			return newRequeueWithDelay(), fmt.Errorf("Error calling comp.PreInstall for component %s: %v", comp.Name(), err.Error())
		}
		// If component is not installed,install it
		if err := comp.Install(compContext); err != nil {
			return newRequeueWithDelay(), fmt.Errorf("Error calling comp.Install for component %s: %v", comp.Name(), err.Error())
		}
		if err := c.updateComponentStatus(log, vzcomp, "Install started", vzapi.InstallStarted); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return newRequeueWithDelay(), nil
	case vzapi.Installing:
		// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
		// If component is enabled -- need to replicate scripts' config merging logic here
		// If component is in deployed state, continue
		if comp.IsReady(compContext) {
			if err := comp.PostInstall(compContext); err != nil {
				return newRequeueWithDelay(), err
			}
			log.Infof("Component %s successfully installed", comp.Name())
			if err := c.updateComponentStatus(log, vzcomp, "Install complete", vzapi.InstallComplete); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
		}
		// Install of this component is not done, requeue to check status
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

// IsEnabled returns true if the component spec has enabled set to true
// Enabled=true is the default
func isVerrazzanoComponentEnabled(vzcomp *vzapi.VerrazzanoComponent, componentName string) bool {
	switch componentName {
	case coherence.ComponentName:
		return coherence.IsEnabled(vzcomp.Spec.Configuration.Components.CoherenceOperator)
	case weblogic.ComponentName:
		return weblogic.IsEnabled(vzcomp.Spec.Configuration.Components.WebLogicOperator)
	}
	return true
}

func (c *ComponentReconciler) updateComponentStatus(log *zap.SugaredLogger, cr *vzapi.VerrazzanoComponent, message string, conditionType vzapi.ConditionType) error {
	t := time.Now().UTC()
	condition := vzapi.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}

	cr.Status.Conditions = appendComponentConditionIfNecessary(log, cr.Name, cr.Status.Conditions, condition)

	// Set the state of resource
	cr.Status.State = checkCondtitionType(conditionType)

	// Update the status
	err := c.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Errorf("Failed to update verrazzano resource status: %v", err)
		return err
	}
	return nil
}

func appendComponentConditionIfNecessary(log *zap.SugaredLogger, resourceName string, currentConditions []vzapi.Condition, newCondition vzapi.Condition) []vzapi.Condition {
	for _, existingCondition := range currentConditions {
		if existingCondition.Type == newCondition.Type {
			return currentConditions
		}
	}
	log.Infof("Adding %s resource newCondition: %v", resourceName, newCondition.Type)
	return append(currentConditions, newCondition)
}

// SetupWithManager creates a new controller and adds it to the manager
func (c *ComponentReconciler) SetupControllerWithManager(mgr ctrl.Manager) error {
	var err error
	c.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoComponent{}).
		// The GenerateChangedPredicate will skip update events that have no change in the object's metadata.generation
		// field.  Any updates to the status or metadata do not cause the metadata.generation to be changed and
		// therefore the reconciler will not be called.
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Build(c)
	return err
}

// reconcileVerrazzanoComponents Scaffolding to allow development of new VerrazzanoComponent impls
func (r *Reconciler) reconcileVerrazzanoComponents(ctx context.Context, log *zap.SugaredLogger, cr *vzapi.Verrazzano) (ctrl.Result, error) {

	// Loop through all of the Verrazzano components and upgrade each one sequentially for now; will parallelize later
	for _, comp := range registry.GetComponents() {
		if !comp.IsOperatorInstallSupported() {
			continue
		}

		vzCompName := cr.Name + "-" + comp.Name()
		var vzcomp = &vzapi.VerrazzanoComponent{}
		if err := r.Get(ctx, types.NamespacedName{Name: vzCompName, Namespace: cr.Namespace}, vzcomp); err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Error getting verrazzano component %s/%s, error: %v", cr.Namespace, vzCompName, err)
				continue
			}
			blockOwnerDeletion := true
			vzcomp := &vzapi.VerrazzanoComponent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vzCompName,
					Namespace: cr.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         cr.APIVersion,
							Kind:               cr.Kind,
							Name:               cr.Name,
							UID:                cr.UID,
							BlockOwnerDeletion: &blockOwnerDeletion,
						},
					},
				},
				Spec: vzapi.VerrazzanoComponentSpec{
					ComponentName: comp.Name(),
					Configuration: cr.Spec,
				},
			}
			if err := r.Create(ctx, vzcomp); err != nil {
				if !errors.IsAlreadyExists(err) {
					log.Errorf("Error creating verrazzano component %s/%s, error: %v", vzcomp.Namespace, vzcomp.Name, err)
					continue
				}
			}
		}

		compList := &vzapi.VerrazzanoComponentList{}
		if err := r.List(ctx, compList, client.InNamespace(cr.Namespace)); err != nil {
			return newRequeueWithDelay(), err
		}
		for _, comp := range compList.Items {
			cr.Status.Components[comp.Spec.ComponentName].State = comp.Status.State
			cr.Status.Components[comp.Spec.ComponentName].Conditions = comp.Status.Conditions
			cr.Status.Components[comp.Spec.ComponentName].Version = comp.Status.Version
		}
		if err := r.Update(ctx, cr); err != nil {
			return newRequeueWithDelay(), err
		}
	}
	return ctrl.Result{}, nil
}
