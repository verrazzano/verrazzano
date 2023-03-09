package vzmodule

import (
	"context"

	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	platformapi "github.com/verrazzano/verrazzano/platform-operator/apis/platform/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	vzcontroller "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// VerrazzanoModuleReconciler reconciles a Verrazzano Platform object
type VerrazzanoModuleReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
}

// Name of finalizer
const (
	finalizerName = "vzmodule.verrazzano.io"

	defaultSourceURI = "http://localhost:9080/vz/stable"
)

var (
	trueValue = true
)

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoModuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&platformapi.Module{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
		}).
		Build(r)
	return err
}

// Reconcile the Module CR
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanomodules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanomodules/status,verbs=get;update;patch
func (r *VerrazzanoModuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// TODO: Metrics setup

	moduleInstance := &platformapi.Module{}
	if err := r.Get(ctx, req.NamespacedName, moduleInstance); err != nil {
		// TODO: errorCounterMetricObject.Inc()
		// If the resource is not found, that means all the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Module resource: %v", err)
		return newRequeueWithDelay(), nil
	}

	// Get the resource logger
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           moduleInstance.Name,
		Namespace:      moduleInstance.Namespace,
		ID:             string(moduleInstance.UID),
		Generation:     moduleInstance.Generation,
		ControllerName: "vzmodule",
	})
	if err != nil {
		// TODO: errorCounterMetricObject.Inc()
		zap.S().Errorf("Failed to create controller logger for Module controller: %v", err)
	}

	// Check if resource is being deleted
	if !moduleInstance.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Oncef("Removing finalizer %s", finalizerName)
		moduleInstance.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(moduleInstance.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, moduleInstance); err != nil {
			return newRequeueWithDelay(), err
		}
		return ctrl.Result{}, nil
	}

	if !vzstring.SliceContainsString(moduleInstance.ObjectMeta.Finalizers, finalizerName) {
		log.Debugf("Adding finalizer %s", finalizerName)
		moduleInstance.ObjectMeta.Finalizers = append(moduleInstance.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(context.TODO(), moduleInstance); err != nil {
			return newRequeueWithDelay(), err
		}
	}

	return r.doReconcile(log, moduleInstance)
}

func (r *VerrazzanoModuleReconciler) doReconcile(log vzlog.VerrazzanoLogger, moduleInstance *platformapi.Module) (ctrl.Result, error) {
	log.Infof("Reconciling Verrazzano module instance %s/%s", moduleInstance.Namespace, moduleInstance.Name)

	platformSource := moduleInstance.Spec.Source
	//if platformSource == nil {
	//	return newRequeueWithDelay(), fmt.Errorf("missing platform source for module %s/%s", moduleInstance.Namespace, moduleInstance.Name)
	//}

	platformInstance, _ := r.getPlatormInstance(log, platformSource)
	//if err != nil {
	//	return vzcontroller.NewRequeueWithDelay(5, 10, time.Second), err
	//}

	namespace := r.lookupChartNamespace(moduleInstance, platformSource)

	// Create a CR to manage the module installation
	moduleInstaller := &modulesv1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      moduleInstance.Name,
			Namespace: namespace,
		},
	}

	controllerutil.CreateOrUpdate(context.TODO(), r.Client, moduleInstaller, func() error {
		moduleInstaller.Spec = modulesv1alpha1.ModuleSpec{
			Installer: modulesv1alpha1.ModuleInstaller{
				HelmChart: &modulesv1alpha1.HelmChart{
					Name:      r.lookupChartName(moduleInstance),
					Namespace: namespace,
					Repository: modulesv1alpha1.HelmRepository{
						URI: r.lookupModuleSourceURI(platformInstance, moduleInstance.Spec.Source),
					},
					Version: r.lookupModuleVersion(moduleInstance),
					// TODO: provide install overrides
					InstallOverrides: vzapi.InstallOverrides{},
				},
			},
		}
		moduleInstaller.OwnerReferences = addOwnerRef(moduleInstaller.OwnerReferences, moduleInstance)
		return nil
	})
	return ctrl.Result{}, nil
}

func (r *VerrazzanoModuleReconciler) getPlatormInstance(log vzlog.VerrazzanoLogger, platformSource *platformapi.PlatformSource) (*platformapi.Platform, error) {
	if platformSource == nil {
		return nil, nil
	}
	platformInstance := platformapi.Platform{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: platformSource.Namespace, Name: platformSource.Name}, &platformInstance)
	if err != nil {
		log.ErrorfThrottledNewErr("Platform instance %s not found for module")
		return nil, err
	}
	return &platformInstance, nil
}

func (r *VerrazzanoModuleReconciler) lookupModuleSourceURI(platform *platformapi.Platform, declaredSource *platformapi.PlatformSource) string {
	if platform == nil || declaredSource == nil {
		return defaultSourceURI
	}
	for _, source := range platform.Spec.Sources {
		if source.Name == declaredSource.Name {
			return source.URL
		}
	}
	return defaultSourceURI
}

func (r *VerrazzanoModuleReconciler) lookupChartNamespace(moduleInstance *platformapi.Module, platformSource *platformapi.PlatformSource) string {
	namespace := moduleInstance.Namespace
	if platformSource != nil && len(platformSource.Namespace) > 0 {
		namespace = platformSource.Namespace
	}
	// TODO: target namespaces mess up owner references, unless we use CrossNamespaceObjectReferences, so disable honoring
	// targetNamespace for now
	//if moduleInstance.Spec.TargetNamespace != nil && len(*moduleInstance.Spec.TargetNamespace) > 0 {
	//	namespace = *moduleInstance.Spec.TargetNamespace
	//}
	return namespace
}

func (r *VerrazzanoModuleReconciler) lookupChartName(moduleInstance *platformapi.Module) string {
	chartName := moduleInstance.Name
	if moduleInstance.Spec.ChartName != nil && len(*moduleInstance.Spec.ChartName) > 0 {
		chartName = *moduleInstance.Spec.ChartName
	}
	return chartName
}

func (r *VerrazzanoModuleReconciler) lookupModuleVersion(moduleInstance *platformapi.Module) string {
	// TODO: Look up module default version based on PlatformDefinition
	if moduleInstance.Spec.Version != nil && len(*moduleInstance.Spec.Version) > 0 {
		return *moduleInstance.Spec.Version
	}
	// TODO: Lookup default version in platform definition
	return ""
}

func addOwnerRef(references []metav1.OwnerReference, owner *platformapi.Module) []metav1.OwnerReference {
	return append(references, metav1.OwnerReference{
		APIVersion:         owner.APIVersion,
		Kind:               owner.Kind,
		Name:               owner.Name,
		UID:                owner.UID,
		Controller:         &trueValue,
		BlockOwnerDeletion: &trueValue,
	})
}

func newRequeueWithDelay() ctrl.Result {
	return vzcontroller.NewRequeueWithDelay(2, 5, time.Second)
}
