// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/pkg/ocne"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps/components"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps/overrides"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/secrets"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/mysqlcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/reconcile"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// StartPlatformOperator Platform operator execution entry point
func StartPlatformOperator(vzconfig config.OperatorConfig, log *zap.SugaredLogger, scheme *runtime.Scheme) error {
	// Determine NGINX namespace before initializing components
	ingressNGINXNamespace, err := nginxutil.DetermineNamespaceForIngressNGINX(vzlog.DefaultLogger())
	if err != nil {
		return err
	}
	nginxutil.SetIngressNGINXNamespace(ingressNGINXNamespace)
	if err = ocne.CreateOCNEMetadataConfigMap(context.Background(), config.GetKubernetesVersionsFile()); err != nil {
		return err
	}

	registry.InitRegistry()
	metricsexporter.Init()

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
	healthCheck := healthcheck.NewHealthChecker(statusUpdater, mgr.GetClient(), time.Duration(vzconfig.HealthCheckPeriodSeconds)*time.Second)
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
	if vzconfig.HealthCheckPeriodSeconds > 0 {
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

	// Setup stacks reconciler
	if err = (&components.ComponentConfigMapReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		DryRun: vzconfig.DryRun,
	}).SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "Failed to setup controller for Verrazzano Stacks")
	}

	if vzconfig.ExperimentalModules {
		log.Infof("Experimental Modules API enabled")
	}

	// +kubebuilder:scaffold:builder
	log.Info("Starting controller-runtime manager")
	if err := mgr.Start(controllerruntime.SetupSignalHandler()); err != nil {
		return errors.Wrap(err, "Failed starting controller-runtime manager: %v")
	}

	return nil
}
