// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package module

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"time"

	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	vpoclient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

	defaultSourceURI  = "http://localhost:9080/vz/stable"
	defaultSourceName = "vz-stable"
)

var (
	trueValue = true
)

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoModuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&v1beta2.Module{}).
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

	moduleInstance := &v1beta2.Module{}
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

func (r *VerrazzanoModuleReconciler) doReconcile(log vzlog.VerrazzanoLogger, moduleInstance *v1beta2.Module) (ctrl.Result, error) {
	log.Infof("Reconciling Verrazzano module instance %s/%s", moduleInstance.Namespace, moduleInstance.Name)

	//platformSource := moduleInstance.Spec.Source
	//platformInstance, _ := r.getPlatormInstance(log, platformSource)
	//platformDefinition, err := r.getPlatformDefinition(log, platformInstance, platformSource)
	//if err != nil {
	//	return newRequeueWithDelay(), err
	//}
	//sourceName, sourceURI := r.lookupModuleSource(platformInstance, moduleInstance.Spec.Source)

	// Find the desired module version
	//targetModuleVersion, err := r.lookupModuleVersion(log, moduleInstance, platformDefinition, sourceName, sourceURI)
	//if err != nil {
	//	return vzcontroller.NewRequeueWithDelay(5, 10, time.Second), err
	//}
	//
	//chartName := r.lookupChartName(moduleInstance)

	// FIXME: we only need the chart type if we can't assume the module we're reconciling is not a CRD or operator chart
	// Pull module type from chart
	//moduleChartType, err := helm.LookupChartType(log, sourceName, sourceURI, chartName, targetModuleVersion)
	//if err != nil {
	//	return vzcontroller.NewRequeueWithDelay(30, 300, time.Second), err
	//}
	//
	//// Load the ModuleDefinitions if necessary
	//if err := helm.ApplyModuleDefinitions(log, r.Client, chartName, targetModuleVersion, sourceURI); err != nil {
	//	return newRequeueWithDelay(), err
	//}
	//
	//namespace := r.lookupChartNamespace(moduleInstance, platformSource)
	//
	//// Load the module dependencies
	//crdDeps, opDeps := r.getModuleDependencies(log, moduleInstance, moduleChartType, sourceURI, err)
	//
	//// Apply CRD Dependencies
	//if result, err := r.applyDependencies(log, moduleInstance, crdDeps, namespace); err != nil || !result.IsZero() {
	//	return result, err
	//}
	//
	//// Apply Operator dependencies
	//if result, err := r.applyDependencies(log, moduleInstance, opDeps, namespace); err != nil || !result.IsZero() {
	//	return result, err
	//}
	//
	//// Apply Module dependencies
	//if result, err := r.applyDependencies(log, moduleInstance, opDeps, namespace); err != nil || !result.IsZero() {
	//	return result, err
	//}
	//
	//if _, err := r.reconcileModule(log, moduleInstance, targetModuleVersion, namespace, sourceURI); err != nil {
	//	return newRequeueWithDelay(), err
	//}
	//if moduleInstance.Status.State != v1beta2.ModuleStateReady {
	//	// Not in a ready state yet, requeue and re-check
	//	log.Progressf("Module %s/%s reconciling, requeue", moduleInstance.Namespace, moduleInstance.Name)
	//	return newRequeueWithDelay(), nil
	//}
	//log.Infof("Module %s/%s reconcile complete", moduleInstance.Namespace, moduleInstance.Name)
	return ctrl.Result{}, nil
}

func (r *VerrazzanoModuleReconciler) getModuleDependencies(log vzlog.VerrazzanoLogger, moduleInstance *v1beta2.Module, moduleChartType v1beta2.ChartType, sourceURI string, err error) ([]v1beta2.ChartDependency, []v1beta2.ChartDependency) {
	var crdDeps []v1beta2.ChartDependency
	var opDeps []v1beta2.ChartDependency
	//// Look up definition in cluster
	//clientset, err := getVPOClientset()
	//if err != nil {
	//	return nil, nil
	//}
	//switch moduleChartType {
	//case v1beta2.ModuleChartType:
	//	// Look up definition in cluster
	//	moduleDef, err := clientset.VerrazzanoV1beta2().ModuleDefinitions().Get(context.TODO(), moduleInstance.Name, metav1.GetOptions{})
	//	if err != nil {
	//		return nil, nil
	//	}
	//	// FIXME: controllerruntime cache is interfering with these lookups
	//	//moduleDef := &v1beta2.ModuleDefinition{}
	//	//if err := r.Get(context.TODO(), types.NamespacedName{Name: moduleInstance.Name}, moduleDef); err != nil {
	//	//	return ctrl.Result{}, err
	//	//}
	//	crdDeps = moduleDef.Spec.CRDDependencies
	//	opDeps = moduleDef.Spec.OperatorDependencies
	//case v1beta2.OperatorChartType:
	//	operatorDef, err := clientset.VerrazzanoV1beta2().OperatorDefinitions().Get(context.TODO(), moduleInstance.Name, metav1.GetOptions{})
	//	if err != nil {
	//		return nil, nil
	//	}
	//	//operatorDef := &v1beta2.OperatorDefinition{}
	//	//if err := r.Get(context.TODO(), types.NamespacedName{Name: moduleInstance.Name, Namespace: namespace}, operatorDef); err != nil {
	//	//	return ctrl.Result{}, err
	//	//}
	//	crdDeps = operatorDef.Spec.CRDDependencies
	//	opDeps = operatorDef.Spec.OperatorDependencies
	//}
	return crdDeps, opDeps
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

func (r *VerrazzanoModuleReconciler) reconcileModule(log vzlog.VerrazzanoLogger, moduleInstance *v1beta2.Module, desiredModuleVersion string, namespace string, sourceURI string) (*v1beta2.ModuleLifecycle, error) {
	lifecycleResource, err := r.createLifecycleResource(sourceURI, moduleInstance.Name, namespace, desiredModuleVersion,
		v1beta2.Overrides{}, createOwnerRef(moduleInstance))
	if err != nil {
		return nil, err
	}
	if err := r.updateModuleInstanceState(moduleInstance, lifecycleResource); err != nil {
		return nil, err
	}
	return lifecycleResource, err
}

func (r *VerrazzanoModuleReconciler) applyDependencies(log vzlog.VerrazzanoLogger, moduleInstance *v1beta2.Module, opDeps []v1beta2.ChartDependency, moduleNamespace string) (ctrl.Result, error) {
	// Fan-out to V2 modules that will be independently reconciled to apply dependencies,
	// and roll up their status via the installers
	var installers []*v1beta2.Module
	for _, operatorDependency := range opDeps {
		dependentModule, err := r.createDependentModule(operatorDependency, moduleNamespace, moduleInstance)
		if err != nil {
			return newRequeueWithDelay(), err
		}
		installers = append(installers, dependentModule)
	}
	// Watch for completion
	allDependenciesMet := r.checkInstallerDependencies(log, installers)
	if !allDependenciesMet {
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

func (r *VerrazzanoModuleReconciler) createDependentModule(operatorDependency v1beta2.ChartDependency, moduleNamespace string, moduleInstance *v1beta2.Module) (*v1beta2.Module, error) {
	// Create or update the dependency resources
	var modDep *v1beta2.Module
	//modDep = &v1beta2.Module{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      operatorDependency.Name,
	//		Namespace: moduleNamespace,
	//		OwnerReferences: []metav1.OwnerReference{
	//			*createOwnerRef(moduleInstance),
	//		},
	//	},
	//}
	//depVersion := operatorDependency.Version
	//_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, modDep, func() error {
	//	modDep.Spec.ChartName = operatorDependency.Name
	//	modDep.Spec.Version = depVersion
	//	modDep.Spec = v1beta2.ModuleSpec{
	//		ChartName:       operatorDependency.Name,
	//		Source:          moduleInstance.Spec.Source,
	//		Enabled:         moduleInstance.Spec.Enabled,
	//		Version:         operatorDependency.Version,
	//		TargetNamespace: moduleInstance.Spec.TargetNamespace,
	//		Reconcile:       moduleInstance.Spec.Reconcile,
	//	}
	//	return nil
	//})
	//if err != nil {
	//	return nil, err
	//}
	return modDep, nil
}

//func (r *VerrazzanoModuleReconciler) applyModuleDependencies(log vzlog.VerrazzanoLogger, moduleInstance *v1beta2.Module, def *v1beta2.ModuleDefinition, sourceURI string, moduleNamespace string) (ctrl.Result, error) {
//	return ctrl.Result{}, nil
//}

func (r *VerrazzanoModuleReconciler) checkInstallerDependencies(log vzlog.VerrazzanoLogger, installers []*v1beta2.Module) bool {
	allDependenciesMet := true
	for _, installer := range installers {
		installerState := installer.Status.State
		if installerState != v1beta2.ModuleStateReady {
			log.Progressf("CRD dependency %s/%s not ready, state: %s", installer.Namespace, installer.Name, installerState)
			allDependenciesMet = false
		}
	}
	return allDependenciesMet
}

func (r *VerrazzanoModuleReconciler) createLifecycleResource(sourceURI string, chartName string, chartNamespace string, chartVersion string, overrides v1beta2.Overrides, ownerRef *metav1.OwnerReference) (*v1beta2.ModuleLifecycle, error) {

	// Create a CR to manage the module installation
	moduleInstaller := &v1beta2.ModuleLifecycle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chartName,
			Namespace: chartNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, moduleInstaller, func() error {
		moduleInstaller.Spec = v1beta2.ModuleLifecycleSpec{
			Installer: v1beta2.ModuleInstaller{
				HelmRelease: &v1beta2.HelmRelease{
					Name:      chartName,
					Namespace: chartNamespace,
					Repository: v1beta2.HelmChartRepository{
						URI: sourceURI,
					},
					ChartInfo: v1beta2.HelmChart{
						Version: chartVersion,
					},
					Overrides: []v1beta2.Overrides{overrides},
				},
			},
		}
		//if ownerRef != nil {
		//	if !ownerRefExists(moduleInstaller, ownerRef) {
		//		moduleInstaller.OwnerReferences = append(moduleInstaller.OwnerReferences, *ownerRef)
		//	}
		//}
		return nil
	})
	return moduleInstaller, err
}

func ownerRefExists(moduleInstaller *v1beta2.Module, ownerRef *metav1.OwnerReference) bool {
	for _, ref := range moduleInstaller.OwnerReferences {
		if ref.UID == ownerRef.UID {
			return true
		}
	}
	return false
}

//func (r *VerrazzanoModuleReconciler) getPlatormInstance(log vzlog.VerrazzanoLogger, platformSource *v1beta2.PlatformSource) (*v1beta2.Platform, error) {
//	if platformSource == nil {
//		return nil, nil
//	}
//	platformInstance := v1beta2.Platform{}
//	err := r.Get(context.TODO(), types.NamespacedName{Namespace: platformSource.Namespace, Name: platformSource.Name}, &platformInstance)
//	if err != nil {
//		log.ErrorfThrottledNewErr("Platform instance %s not found for module")
//		return nil, err
//	}
//	return &platformInstance, nil
//}

//func (r *VerrazzanoModuleReconciler) lookupModuleSource(platform *v1beta2.Platform, declaredSource *v1beta2.PlatformSource) (sourceName, sourceURI string) {
//	if platform == nil || declaredSource == nil {
//		return defaultSourceName, defaultSourceURI
//	}
//	for _, source := range platform.Spec.Sources {
//		if source.Name == declaredSource.Source {
//			return source.Name, source.URL
//		}
//	}
//	return defaultSourceName, defaultSourceURI
//}

//func (r *VerrazzanoModuleReconciler) lookupChartNamespace(moduleInstance *v1beta2.Module, platformSource *v1beta2.PlatformSource) string {
//	namespace := moduleInstance.Namespace
//	if len(namespace) == 0 && platformSource != nil && len(platformSource.Namespace) > 0 {
//		namespace = platformSource.Namespace
//	}
//	// TODO: target namespaces mess up owner references, unless we use CrossNamespaceObjectReferences, so disable honoring
//	// targetNamespace for now
//	//if moduleInstance.Spec.TargetNamespace != nil && len(*moduleInstance.Spec.TargetNamespace) > 0 {
//	//	namespace = *moduleInstance.Spec.TargetNamespace
//	//}
//	return namespace
//}

//func (r *VerrazzanoModuleReconciler) lookupChartName(moduleInstance *v1beta2.Module) string {
//	chartName := moduleInstance.Name
//	if len(moduleInstance.Spec.ChartName) > 0 {
//		chartName = moduleInstance.Spec.ChartName
//	}
//	return chartName
//}

func (r *VerrazzanoModuleReconciler) lookupModuleVersion(log vzlog.VerrazzanoLogger, moduleInstance *v1beta2.Module, pd *v1beta2.PlatformDefinition, repoName string, repoURI string) (string, error) {
	// Find target module version
	// - declared in the Module instance
	var modVersion string
	// Look up the explicitly declared module version
	if len(moduleInstance.Spec.Version) > 0 {
		modVersion = moduleInstance.Spec.Version
	}
	// - default version in the Platform definition, if it exists there
	defaultVersion, vzVersionConstraints, found := r.getModuleVersionInfoFromPlatform(pd, moduleInstance)
	if found {
		if len(modVersion) == 0 {
			modVersion = defaultVersion
		}
		matches, err := semver.MatchesConstraint(modVersion, vzVersionConstraints)
		if err != nil {
			return "", log.ErrorfThrottledNewErr("Module %s/%s version %s failed to meet ModuleDefinition constraints: %s",
				moduleInstance.Namespace, moduleInstance.Name, modVersion, vzVersionConstraints)
		}
		if matches {
			return modVersion, nil
		}
	}
	// - find the most recent version in the repo compatible based on the Chart annotations
	modVersion, err := helm.FindNearestSupportingChartVersion(log, moduleInstance.Name, repoName, repoURI, pd.Spec.Version)
	if err != nil {
		return "", err
	}

	return modVersion, nil
}

// getModuleVersionInfoFromPlatform Obtains the module version information declared in the Platform definition
func (r *VerrazzanoModuleReconciler) getModuleVersionInfoFromPlatform(pd *v1beta2.PlatformDefinition, moduleInstance *v1beta2.Module) (defaultModuleVersion, vzVersionConstraints string, found bool) {
	for _, modDef := range pd.Spec.OperatorVersions {
		if modDef.Name == moduleInstance.Name {
			return modDef.DefaultVersion, modDef.SupportedVersions, true
		}
	}
	for _, modDef := range pd.Spec.ModuleVersions {
		if modDef.Name == moduleInstance.Name {
			return modDef.DefaultVersion, modDef.SupportedVersions, true
		}
	}
	for _, modDef := range pd.Spec.CRDVersions {
		if modDef.Name == moduleInstance.Name {
			return modDef.DefaultVersion, modDef.SupportedVersions, true
		}
	}
	return "", "", false
}

//func (r *VerrazzanoModuleReconciler) getPlatformDefinition(log vzlog.VerrazzanoLogger, instance *v1beta2.Platform, source *v1beta2.PlatformSource) (*v1beta2.PlatformDefinition, error) {
//	if source == nil || len(source.Namespace) == 0 || len(source.Name) == 0 {
//		return nil, log.ErrorfThrottledNewErr("Source not defined for module %s/%s", instance.Namespace, instance.Name)
//	}
//	pd := &v1beta2.PlatformDefinition{}
//	// TODO: Need to figure out relationship between PD and platform instance; might need to be a configmap
//	//   - perhaps the focus of the platform controller is downloading/creating the platform definition based on the Platform instance version?
//	err := r.Get(context.TODO(), types.NamespacedName{Name: source.Name, Namespace: source.Namespace}, pd)
//	if err != nil {
//		return nil, err
//	}
//	return pd, nil
//}

func (r *VerrazzanoModuleReconciler) updateModuleInstanceState(instance *v1beta2.Module, lifecycleResource *v1beta2.ModuleLifecycle) error {
	//instance.Status.State = v1beta2.ModuleStateUnknown
	//if lifecycleResource != nil && lifecycleResource.Status.State != nil {
	//	installerState := *lifecycleResource.Status.State
	//	switch installerState {
	//	case v1beta2.StateReady:
	//		instance.Status.State = v1beta2.ModuleStateReady
	//		installerChart := lifecycleResource.Spec.Installer.HelmChart
	//		if installerChart != nil {
	//			instance.Status.Version = installerChart.Version
	//		}
	//	default:
	//		instance.Status.State = v1beta2.ModuleStateReconciling
	//	}
	//}
	return r.Status().Update(context.TODO(), instance)
}

//func (r *VerrazzanoModuleReconciler) loadModuleDefinitions(log vzlog.VerrazzanoLogger, instance *v1beta2.Module, modVersion string, sourceName string, sourceURI string, platformVersion string) error {
//	return helm.ApplyModuleDefinitions(
//		log, r.Client, instance.Spec.ChartName, modVersion, sourceURI, platformVersion,
//	)
//}

func createOwnerRef(owner *v1beta2.Module) *metav1.OwnerReference {
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
