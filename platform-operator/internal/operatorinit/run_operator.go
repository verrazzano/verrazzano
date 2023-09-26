// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	"github.com/pkg/errors"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps/components"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps/overrides"
	opensearchcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/integration/opensearch"
	modulehandlerfactory "github.com/verrazzano/verrazzano/platform-operator/controllers/module/component-handler/factory"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/obsolete/namespacewatch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/obsolete/reconcile"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/secrets"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	verrazzancontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/controller"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/mysqlcheck"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"sync"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
	"time"
)

const vpoHelmChartConfigMapName = "vpo-helm-chart"

// StartPlatformOperator Platform operator execution entry point
func StartPlatformOperator(vzconfig config.OperatorConfig, log *zap.SugaredLogger, scheme *runtime.Scheme) error {
	// Determine NGINX namespace before initializing components
	ingressNGINXNamespace, err := nginxutil.DetermineNamespaceForIngressNGINX(vzlog.DefaultLogger())
	if err != nil {
		return err
	}
	nginxutil.SetIngressNGINXNamespace(ingressNGINXNamespace)

	if err := CreateVerrazzanoVersionsConfigMap(context.Background()); err != nil {
		return err
	}

	registry.InitRegistry()
	metricsexporter.Init()

	chartDir := config.GetHelmVPOChartsDir()
	files, err := os.ReadDir(chartDir)
	if err != nil {
		return errors.Wrap(err, "Failed to read the verrazzano-platform-operator helm chart directory")
	}
	vpoHelmChartConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vpoHelmChartConfigMapName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
	}
	err = generateConfigMapFromHelmChartFiles(chartDir, "", files, vpoHelmChartConfigMap)
	if err != nil {
		return errors.Wrap(err, "Failed to generate config map containing the verrazzano-platform-operator helm chart")
	}
	kubeClient, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	err = createVPOHelmChartConfigMap(kubeClient, vpoHelmChartConfigMap)
	if err != nil {
		return errors.Wrap(err, "Failed to create/update config map containing the verrazzano-platform-operator helm chart")
	}

	mgr, err := controllerruntime.NewManager(k8sutil.GetConfigOrDieFromController(), controllerruntime.Options{
		Scheme:             scheme,
		MetricsBindAddress: vzconfig.MetricsAddr,
		Port:               8080,
		LeaderElection:     vzconfig.LeaderElectionEnabled,
		LeaderElectionID:   "3ec4d290.verrazzano.io",
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create a controller-runtime manager")
	}

	metricsexporter.StartMetricsServer(log)

	// Set up the reconciler
	statusUpdater := healthcheck.NewStatusUpdater(mgr.GetClient())

	// There are 2 verrazzano CR controllers, the new module-based controller and the original (legacy) one.
	// Use the new module-base controllers if module integration is enabled otherwise use the legacy verrazzano one.
	if vzconfig.ModuleIntegration {
		if err := initModuleControllers(log, mgr); err != nil {
			log.Errorf("Failed to start all module controllers", err)
			return errors.Wrap(err, "Failed to initialize modules controllers for the components")
		}

		// init verrazzano module controller
		if err := verrazzancontroller.InitController(mgr); err != nil {
			log.Errorf("Failed to start module-based Verrazzano controller", err)
			return errors.Wrap(err, "Failed to initialize controller for module-based Verrazzano controller")
		}
	} else {
		// init legacy verrazzano controller
		reconciler := reconcile.Reconciler{
			Client:            mgr.GetClient(),
			Scheme:            mgr.GetScheme(),
			DryRun:            vzconfig.DryRun,
			WatchedComponents: map[string]bool{},
			WatchMutex:        &sync.RWMutex{},
			StatusUpdater:     statusUpdater,
		}
		if err = reconciler.SetupWithManager(mgr); err != nil {
			return errors.Wrap(err, "Failed to setup controller")
		}
	}

	if err := opensearchcontroller.InitController(mgr); err != nil {
		log.Errorf("Failed to start module-based opensearch controller", err)
		return errors.Wrap(err, "Failed to initialize controller for module-based opensearch controller")
	}

	// Setup health checker
	healthCheck := healthcheck.NewHealthChecker(statusUpdater, mgr.GetClient(), time.Duration(vzconfig.HealthCheckPeriodSeconds)*time.Second)
	if vzconfig.HealthCheckPeriodSeconds > 0 {
		healthCheck.Start()
	}

	dynamicClient, err := k8sutil.GetDynamicClient()
	if err != nil {
		return errors.Wrapf(err, "Failed to get Dynamic Client")
	}
	// Setup secrets reconciler
	if err = (&secrets.VerrazzanoSecretsReconciler{
		Client:        mgr.GetClient(),
		DynamicClient: dynamicClient,
		Scheme:        mgr.GetScheme(),
		StatusUpdater: statusUpdater,
	}).SetupWithManager(mgr); err != nil {
		return errors.Wrapf(err, "Failed to setup controller VerrazzanoSecrets")
	}

	// Setup configMaps reconciler
	if err = (&overrides.OverridesConfigMapsReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		StatusUpdater: statusUpdater,
	}).SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "Failed to setup controller VerrazzanoConfigMaps")
	}

	// Setup MySQL checker
	mysqlCheck, err := mysqlcheck.NewMySQLChecker(mgr.GetClient(), time.Duration(vzconfig.MySQLCheckPeriodSeconds)*time.Second, time.Duration(vzconfig.MySQLRepairTimeoutSeconds)*time.Second)
	if err != nil {
		return errors.Wrap(err, "Failed starting MySQLChecker")
	}
	mysqlCheck.Start()

	// Setup Namespaces watcher
	watchNamespace := namespacewatch.NewNamespaceWatcher(mgr.GetClient(), time.Duration(vzconfig.NamespacePeriodSeconds)*time.Second)
	if vzconfig.NamespacePeriodSeconds > 0 {
		watchNamespace.Start()
	}

	// Setup stacks reconciler
	if err = (&components.ComponentConfigMapReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		DryRun: vzconfig.DryRun,
	}).SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "Failed to setup controller for Verrazzano Stacks")
	}

	if vzconfig.ModuleIntegration {
		log.Infof("Module Integration enabled")
	}

	// +kubebuilder:scaffold:builder
	log.Info("Starting controller-runtime manager")
	if err := mgr.Start(controllerruntime.SetupSignalHandler()); err != nil {
		return errors.Wrap(err, "Failed starting controller-runtime manager: %v")
	}

	return nil
}

// generateConfigMapFromHelmChartFiles generates a config map with the files from a given helm chart directory
func generateConfigMapFromHelmChartFiles(dir string, key string, files []os.DirEntry, configMap *corev1.ConfigMap) error {
	for _, file := range files {
		newKey := file.Name()
		if len(key) != 0 {
			newKey = key + "/" + file.Name()
		}
		if file.IsDir() {
			files2, err := os.ReadDir(dir + "/" + newKey)
			if err != nil {
				return err
			}
			err = generateConfigMapFromHelmChartFiles(dir, newKey, files2, configMap)
			if err != nil {
				return err
			}
		} else {
			err := addKeyForFileToConfigMap(dir, newKey, configMap)
			if err != nil {
				return nil
			}
		}
	}

	return nil
}

// addKeyForFileToConfigMap adds a key for a file to a config map
func addKeyForFileToConfigMap(dir string, key string, configMap *corev1.ConfigMap) error {
	data, err := os.ReadFile(dir + "/" + key)
	if err != nil {
		return err
	}
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	// Use "..." as a path separator since "/" is an invalid character for a config map data key
	keyName := strings.ReplaceAll(key, "/", "...")
	configMap.Data[keyName] = string(data)

	return nil
}

// createVPOHelmChartConfigMap creates/updates a config map containing the VPO helm chart
func createVPOHelmChartConfigMap(kubeClient kubernetes.Interface, configMap *corev1.ConfigMap) error {
	_, err := kubeClient.CoreV1().ConfigMaps(constants.VerrazzanoInstallNamespace).Get(context.TODO(), configMap.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			_, err = kubeClient.CoreV1().ConfigMaps(constants.VerrazzanoInstallNamespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		}
	} else {
		_, err = kubeClient.CoreV1().ConfigMaps(constants.VerrazzanoInstallNamespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	}

	return err
}

// initModuleControllers creates a controller for every module
// The controller uses the module name (i.e. component name) as a predicate,
// so that each controller only processes Module CRs for the respective component.
func initModuleControllers(log *zap.SugaredLogger, mgr controllerruntime.Manager) error {
	// Run 50 controllers max concurrently.  Even though there is a controller for each
	// module, the controller-runtime serializes reconciles if the MaxConcurrentReconciles is 1.
	// This is because it is the same resource type being reconcile, but with a
	// different predicate per controller.  Also, with the MaxConcurrentReconciles > 1, there
	// is a chance that the controller-runtime will call the same controller (e.g. istio) to
	// reconcile, while it is already reconciling a given CR.  There is a check in the
	// module-operator package to catch this and requeue, thereby never allowing more
	// than 1 instance of the same module from having more than 1 outstanding reconcile happening.
	// See https://github.com/verrazzano/verrazzano-modules/blob/main/module-operator/controllers/module/reconciler.go#L225
	// for details.
	const maxReconciles = 50

	// Temp hack to prevent module controller from looking up helm info
	module.IgnoreHelmInfo()

	for _, comp := range registry.GetComponents() {
		if !comp.ShouldUseModule() {
			continue
		}
		// Create and initialize the module controller.  Note that the implementation of the module controller
		// is in the module-operator package (in the module-operator repo).  This is the exact
		// same controller that is used by the module-operator used by OCNE.  The only difference
		// is the set of handlers.  This code uses component shim handlers in the VPO code, whereas
		// the OCNE module-operator uses OCNE handlers in the module-operator controller.  See
		// ModuleHandlerInfo param below.  Also see the module-operator where the OCNE handlers are used
		// for comparison: https://github.com/verrazzano/verrazzano-modules/blob/main/module-operator/main.go
		if err := module.InitController(module.ModuleControllerConfig{
			ControllerManager: mgr,
			ModuleHandlerInfo: modulehandlerfactory.NewModuleHandlerInfo(),
			ModuleClass:       moduleapi.ModuleClassType(comp.Name()),
			WatchDescriptors:  comp.GetWatchDescriptors(),
			ControllerOptions: &controller.Options{MaxConcurrentReconciles: maxReconciles},
		}); err != nil {
			log.Errorf("Failed to start the controller for module %s:%v", comp.Name(), err)
			return err
		}
	}
	return nil
}
