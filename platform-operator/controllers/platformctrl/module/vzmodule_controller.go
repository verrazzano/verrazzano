package module

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	platformapi "github.com/verrazzano/verrazzano/platform-operator/apis/platform/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoclient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
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
	platformInstance, _ := r.getPlatormInstance(log, platformSource)
	sourceURI := r.lookupModuleSourceURI(platformInstance, moduleInstance.Spec.Source)

	namespace := r.lookupChartNamespace(moduleInstance, platformSource)

	// Pull module type from chart
	// FIXME: we only need the chart type if we can't assume the module we're reconciling is not a CRD or operator chart
	var crdDeps []platformapi.ChartDependency
	var opDeps []platformapi.ChartDependency
	moduleChartType := helm.LookupChartType(log, "vz-stable", sourceURI, moduleInstance.Name)
	// Look up definition in cluster
	clientset, err := getVPOClientset()
	if err != nil {
		return ctrl.Result{}, err
	}
	switch moduleChartType {
	case platformapi.ModuleChartType:
		// Look up definition in cluster
		moduleDef, err := clientset.PlatformV1alpha1().ModuleDefinitions(namespace).Get(context.TODO(), moduleInstance.Name, metav1.GetOptions{})
		if err != nil {
			return ctrl.Result{}, err
		}
		//moduleDef := &platformapi.ModuleDefinition{}
		//if err := r.Get(context.TODO(), types.NamespacedName{Name: moduleInstance.Name, Namespace: namespace}, moduleDef); err != nil {
		//	return ctrl.Result{}, err
		//}
		crdDeps = moduleDef.Spec.CRDDependencies
		opDeps = moduleDef.Spec.OperatorDependencies
	case platformapi.OperatorChartType:
		operatorDef, err := clientset.PlatformV1alpha1().OperatorDefinitions(namespace).Get(context.TODO(), moduleInstance.Name, metav1.GetOptions{})
		if err != nil {
			return ctrl.Result{}, err
		}
		//operatorDef := &platformapi.OperatorDefinition{}
		//if err := r.Get(context.TODO(), types.NamespacedName{Name: moduleInstance.Name, Namespace: namespace}, operatorDef); err != nil {
		//	return ctrl.Result{}, err
		//}
		crdDeps = operatorDef.Spec.CRDDependencies
		opDeps = operatorDef.Spec.OperatorDependencies
	}

	// Apply CRD depedencies
	// FIXME: CRDs are cluster-scoped, but charts are namespace-scoped; we need to figure out how to
	//    manage that, either likely by guarding the CRD resources in the charts or ignoring errors
	//    - Do we list APIs in the operator definition, and detect their presence?
	//    - CRDDefinition may be in order, i.e., allow charts to list published APIs
	//    - and, we may want to distinguish Module instances by their type (operator, crd), or collapse the notions
	//      and just have general ModuleDefinitions that support publishing APIs with module dependencies
	if result, err := r.applyCRDDependencies(log, moduleInstance, crdDeps, sourceURI, namespace); err != nil || !result.IsZero() {
		return result, err
	}
	// Apply Operator dependencies
	// FIXME: Need to indicate if an operator is global or scoped to the namespace of the referencing Module
	//   also, what if the user wants to leverage an existing operator that they installed?
	//   - might have to break up the definition between
	if result, err := r.applyOperatorDependencies(log, moduleInstance, opDeps, sourceURI, namespace); err != nil || !result.IsZero() {
		return result, err
	}
	// Apply Module dependencies
	// FIXME: Are module dependencies owned by the module being resolved?  Seems like it might lead to complicated nesting
	//if result, err := r.applyModuleDependencies(log, moduleInstance, crdDeps, opDeps, sourceURI, namespace); err != nil || !result.IsZero() {
	//	return result, err
	//}

	if _, err := r.reconcileModule(moduleInstance, namespace, sourceURI); err != nil {
		return newRequeueWithDelay(), err
	}
	return ctrl.Result{}, nil
}

func getVPOClientset() (*vpoclient.Clientset, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	vpoclientset, err := vpoclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return vpoclientset, nil
}

func (r *VerrazzanoModuleReconciler) reconcileModule(moduleInstance *platformapi.Module, namespace string, sourceURI string) (*modulesv1alpha1.Module, error) {
	resource, err := r.createLifecycleResource(sourceURI, moduleInstance.Name, namespace, r.lookupModuleVersion(moduleInstance),
		vzapi.InstallOverrides{}, createOwnerRef(moduleInstance))
	if err != nil {
		return nil, err
	}
	return resource, err
}

func (r *VerrazzanoModuleReconciler) applyCRDDependencies(log vzlog.VerrazzanoLogger, moduleInstance *platformapi.Module, crdDeps []platformapi.ChartDependency, sourceURI string, moduleNamespace string) (ctrl.Result, error) {
	// FIXME: should probably fan-out to create other V2 modules that will be independently reconciled?
	//   - can roll up their status via the installers
	// FIXME: how do we determine if the CRD dependency is already installed?
	var installers []*modulesv1alpha1.Module
	for _, crdChartDependency := range crdDeps {
		// Create or update the dependency resources
		installer, err := r.createLifecycleResource(sourceURI, crdChartDependency.Name, moduleNamespace, crdChartDependency.Version, vzapi.InstallOverrides{}, createOwnerRef(moduleInstance))
		if err != nil {
			return newRequeueWithDelay(), err
		}
		installers = append(installers, installer)
	}
	// FIXME: this is probably too simplistic, but what the hell, good enough for a POC
	// Watch for completion
	allDependenciesMet := r.checkInstallerDependencies(log, installers)
	if !allDependenciesMet {
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

func (r *VerrazzanoModuleReconciler) applyOperatorDependencies(log vzlog.VerrazzanoLogger, moduleInstance *platformapi.Module, opDeps []platformapi.ChartDependency, sourceURI string, moduleNamespace string) (ctrl.Result, error) {
	// FIXME: should probably fan-out to create other V2 modules that will be independently reconciled?
	//   - can roll up their status via the installers
	var installers []*modulesv1alpha1.Module
	for _, operatorDependency := range opDeps {
		// Create or update the dependency resources
		installer, err := r.createLifecycleResource(sourceURI, operatorDependency.Name, moduleNamespace, operatorDependency.Version, vzapi.InstallOverrides{}, createOwnerRef(moduleInstance))
		if err != nil {
			return newRequeueWithDelay(), err
		}
		installers = append(installers, installer)
	}
	// FIXME: this is probably too simplistic, but what the hell, good enough for a POC
	// Watch for completion
	allDependenciesMet := r.checkInstallerDependencies(log, installers)
	if !allDependenciesMet {
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

func (r *VerrazzanoModuleReconciler) applyModuleDependencies(log vzlog.VerrazzanoLogger, moduleInstance *platformapi.Module, def *platformapi.ModuleDefinition, sourceURI string, moduleNamespace string) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *VerrazzanoModuleReconciler) checkInstallerDependencies(log vzlog.VerrazzanoLogger, installers []*modulesv1alpha1.Module) bool {
	allDependenciesMet := true
	for _, installer := range installers {
		installerState := installer.Status.State
		if installerState != nil && *installerState != modulesv1alpha1.StateReady {
			log.Progressf("CRD dependency %s/%s not ready, state: %s", installer.Namespace, installer.Name, *installerState)
			allDependenciesMet = false
		}
	}
	return allDependenciesMet
}

func (r *VerrazzanoModuleReconciler) createLifecycleResource(sourceURI string, chartName string, chartNamespace string, chartVersion string, overrides vzapi.InstallOverrides, ownerRef *metav1.OwnerReference) (*modulesv1alpha1.Module, error) {

	// Create a CR to manage the module installation
	moduleInstaller := &modulesv1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chartName,
			Namespace: chartNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, moduleInstaller, func() error {
		moduleInstaller.Spec = modulesv1alpha1.ModuleSpec{
			Installer: modulesv1alpha1.ModuleInstaller{
				HelmChart: &modulesv1alpha1.HelmChart{
					Name:      chartName,
					Namespace: chartNamespace,
					Repository: modulesv1alpha1.HelmRepository{
						URI: sourceURI,
					},
					Version: chartVersion,
					// TODO: provide install overrides
					InstallOverrides: overrides,
				},
			},
		}
		if ownerRef != nil {
			moduleInstaller.OwnerReferences = append(moduleInstaller.OwnerReferences, *ownerRef)
		}
		return nil
	})
	return moduleInstaller, err
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

func createOwnerRef(owner *platformapi.Module) *metav1.OwnerReference {
	return &metav1.OwnerReference{
		APIVersion:         owner.APIVersion,
		Kind:               owner.Kind,
		Name:               owner.Name,
		UID:                owner.UID,
		Controller:         &trueValue,
		BlockOwnerDeletion: &trueValue,
	}
}

func newRequeueWithDelay() ctrl.Result {
	return vzcontroller.NewRequeueWithDelay(2, 5, time.Second)
}
