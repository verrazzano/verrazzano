// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"sync"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/mysqlcheck"

	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/secrets"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/reconcile"
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
	statusUpdater := healthcheck.NewStatusUpdater(mgr.GetClient())
	healthCheck := healthcheck.NewHealthChecker(statusUpdater, mgr.GetClient(), time.Duration(config.HealthCheckPeriodSeconds)*time.Second)
	reconciler := reconcile.Reconciler{
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

	// Setup MySQL checker
	mysqlCheck, err := mysqlcheck.NewMySQLChecker(mgr.GetClient(), time.Duration(config.HealthCheckPeriodSeconds)*time.Second)
	if err != nil {
		return errors.Wrap(err, "Failed starting MySQLChecker")
	}
	mysqlCheck.Start()

	// +kubebuilder:scaffold:builder
	log.Info("Starting controller-runtime manager")
	if err := mgr.Start(controllerruntime.SetupSignalHandler()); err != nil {
		return errors.Wrap(err, "Failed starting controller-runtime manager: %v")
	}

	return nil
}
