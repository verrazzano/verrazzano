// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"github.com/verrazzano/verrazzano/application-operator/controllers/appconfig"
	"github.com/verrazzano/verrazzano/application-operator/controllers/cohworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/containerizedworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/helidonworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/ingresstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingtrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricsbinding"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/namespace"
	"github.com/verrazzano/verrazzano/application-operator/controllers/wlsworkload"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	vzlog2 "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func StartApplicationOperator(metricsAddr string, enableLeaderElection bool, defaultMetricsScraper string, log *zap.SugaredLogger, scheme *runtime.Scheme) error {
	ingressNGINXNamespace, err := nginxutil.DetermineNamespaceForIngressNGINX(vzlog2.DefaultLogger())
	if err != nil {
		return err
	}
	nginxutil.SetIngressNGINXNamespace(ingressNGINXNamespace)

	mgr, err := ctrl.NewManager(k8sutil.GetConfigOrDieFromController(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "5df248b3.verrazzano.io",
	})
	if err != nil {
		log.Errorf("Failed to start manager: %v", err)
		return err
	}

	log.Info("Starting application reconcilers")
	if err := setupAppReconcilers(mgr, defaultMetricsScraper, log); err != nil {
		return err
	}

	// Initialize the metricsExporter
	if err := metricsexporter.StartMetricsServer(); err != nil {
		log.Errorf("Failed to create metrics exporter: %v", err)
		return err
	}

	// +kubebuilder:scaffold:builder

	log.Info("Starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Errorf("Failed to run manager: %v", err)
		return err
	}
	return err
}

func setupAppReconcilers(mgr manager.Manager, defaultMetricsScraper string, log *zap.SugaredLogger) error {
	logger, err := vzlog.BuildZapInfoLogger(0)
	if err != nil {
		return err
	}
	if err := (&ingresstrait.Reconciler{
		Client: mgr.GetClient(),
		Log:    log,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create IngressTrait controller: %v", err)
		return err
	}
	metricsReconciler := &metricstrait.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Scraper: defaultMetricsScraper,
	}

	if err = metricsReconciler.SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create MetricsTrait controller: %v", err)
		return err
	}

	if err != nil {
		log.Errorf("Failed to create ApplicationConfiguration logger: %v", err)
		return err
	}
	if err = (&cohworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     logger,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create VerrazzanoCoherenceWorkload controller: %v", err)
		return err
	}
	wlsWorkloadReconciler := &wlsworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}
	if err = wlsWorkloadReconciler.SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create VerrazzanoWeblogicWorkload controller %v", err)
		return err
	}
	if err = (&helidonworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create VerrazzanoHelidonWorkload controller: %v", err)
		return err
	}
	// Setup the namespace reconciler
	if _, err := namespace.NewNamespaceController(mgr, log.With("controller", "VerrazzanoNamespaceController")); err != nil {
		log.Errorf("Failed to create VerrazzanoNamespaceController controller: %v", err)
		return err
	}

	if err = (&loggingtrait.LoggingTraitReconciler{
		Client: mgr.GetClient(),
		Log:    log,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create LoggingTrait controller: %v", err)
		return err
	}

	if err = (&appconfig.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create ApplicationConfiguration controller: %v", err)
		return err
	}
	if err = (&containerizedworkload.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create ContainerizedWorkload controller: %v", err)
		return err
	}
	// Register the metrics workload controller
	if err = (&metricsbinding.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create MetricsBinding controller: %v", err)
		return err
	}
	return nil
}
