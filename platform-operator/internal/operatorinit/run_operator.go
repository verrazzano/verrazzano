package operatorinit

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/secrets"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// StartPlatformOperator Platform operator execution entry point
func StartPlatformOperator(config config.OperatorConfig, log *zap.SugaredLogger, scheme *runtime.Scheme) error {
	mgr, err := controllerruntime.NewManager(controllerruntime.GetConfigOrDie(), controllerruntime.Options{
		Scheme:             scheme,
		MetricsBindAddress: config.MetricsAddr,
		Port:               8080,
		LeaderElection:     config.LeaderElectionEnabled,
		LeaderElectionID:   "3ec4d290.verrazzano.io",
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create a controller-runtime manager")
	}

	metricsexporter.StartMetricsServer(log)

	// Set up the reconciler
	statusUpdater := status.NewStatusUpdater(mgr.GetClient())
	healthCheck := status.NewHealthChecker(statusUpdater, mgr.GetClient(), time.Duration(config.HealthCheckPeriodSeconds)*time.Second)
	reconciler := verrazzano.Reconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		DryRun:            config.DryRun,
		WatchedComponents: map[string]bool{},
		WatchMutex:        &sync.RWMutex{},
		StatusUpdater:     statusUpdater,
	}
	if err = reconciler.SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "Failed to setup controller")
	}
	if config.HealthCheckPeriodSeconds > 0 {
		healthCheck.Start()
	}

	// Set up the reconciler for VerrazzanoManagedCluster objects
	if err = (&clusters.VerrazzanoManagedClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "Failed to setup controller VerrazzanoManagedCluster")
	}

	// Start the goroutine to sync Rancher clusters and VerrazzanoManagedCluster objects
	rancherClusterSyncer := &clusters.RancherClusterSyncer{
		Client: mgr.GetClient(),
		Log:    log,
	}
	go rancherClusterSyncer.StartSyncing()

	// Setup secrets reconciler
	if err = (&secrets.VerrazzanoSecretsReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		StatusUpdater: statusUpdater,
	}).SetupWithManager(mgr); err != nil {
		return errors.Wrapf(err, "Failed to setup controller VerrazzanoSecrets")
	}

	// Setup configMaps reconciler
	if err = (&configmaps.VerrazzanoConfigMapsReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		StatusUpdater: statusUpdater,
	}).SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "Failed to setup controller VerrazzanoConfigMaps")
	}

	// +kubebuilder:scaffold:builder
	log.Info("Starting controller-runtime manager")
	if err := mgr.Start(controllerruntime.SetupSignalHandler()); err != nil {
		return errors.Wrap(err, "Failed starting controller-runtime manager: %v")
	}
	return nil
}
