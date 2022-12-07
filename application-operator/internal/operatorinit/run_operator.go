// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"fmt"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/appconfig"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterapplicationconfiguration"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclustercomponent"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterconfigmap"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclustersecret"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/verrazzanoproject"
	"github.com/verrazzano/verrazzano/application-operator/controllers/cohworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/containerizedworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/helidonworkload"
	"github.com/verrazzano/verrazzano/application-operator/controllers/ingresstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingtrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricsbinding"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/controllers/namespace"
	"github.com/verrazzano/verrazzano/application-operator/controllers/wlsworkload"
	"github.com/verrazzano/verrazzano/application-operator/mcagent"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	vmcclient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/scheme"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func StartApplicationOperator(metricsAddr string, enableLeaderElection bool, defaultMetricsScraper string, log *zap.SugaredLogger, scheme *runtime.Scheme) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "5df248b3.verrazzano.io",
	})
	if err != nil {
		return fmt.Errorf("Failed to start manager: %v", err)
	}

	if err = (&ingresstrait.Reconciler{
		Client: mgr.GetClient(),
		Log:    log,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create IngressTrait controller: %v", err)
	}
	metricsReconciler := &metricstrait.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Scraper: defaultMetricsScraper,
	}

	if err = metricsReconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create MetricsTrait controller: %v", err)
	}

	logger, err := vzlog.BuildZapInfoLogger(0)
	if err != nil {
		return fmt.Errorf("Failed to create ApplicationConfiguration logger: %v", err)
	}
	if err = (&cohworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     logger,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create VerrazzanoCoherenceWorkload controller: %v", err)
	}
	wlsWorkloadReconciler := &wlsworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}
	if err = wlsWorkloadReconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create VerrazzanoWeblogicWorkload controller %v", err)
	}
	if err = (&helidonworkload.Reconciler{
		Client:  mgr.GetClient(),
		Log:     log,
		Scheme:  mgr.GetScheme(),
		Metrics: metricsReconciler,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create VerrazzanoHelidonWorkload controller: %v", err)
	}
	// Setup the namespace reconciler
	if _, err := namespace.NewNamespaceController(mgr, log.With("controller", "VerrazzanoNamespaceController")); err != nil {
		return fmt.Errorf("Failed to create VerrazzanoNamespaceController controller: %v", err)
	}

	// Create a buffered channel of size 10 for the multi cluster agent to receive messages
	agentChannel := make(chan clusters.StatusUpdateMessage, constants.StatusUpdateChannelBufferSize)

	// Initialize the metricsExporter
	if err := metricsexporter.StartMetricsServer(); err != nil {
		return fmt.Errorf("Failed to create metrics exporter: %v", err)
	}

	if err = (&multiclustersecret.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterSecretKind, err)
	}
	if err = (&multiclustercomponent.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterComponentKind, err)
	}
	if err = (&multiclusterconfigmap.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create %s controller %v", clustersv1alpha1.MultiClusterConfigMapKind, err)
	}
	if err = (&multiclusterapplicationconfiguration.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterAppConfigKind, err)
	}
	scheme = mgr.GetScheme()
	vmcclient.AddToScheme(scheme)
	if err = (&verrazzanoproject.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       scheme,
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create %s controller %v", clustersv1alpha1.VerrazzanoProjectKind, err)
	}
	if err = (&loggingtrait.LoggingTraitReconciler{
		Client: mgr.GetClient(),
		Log:    log,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create LoggingTrait controller: %v", err)
	}
	if err = (&appconfig.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create ApplicationConfiguration controller: %v", err)
	}
	if err = (&containerizedworkload.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create ContainerizedWorkload controller: %v", err)
	}
	// Register the metrics workload controller
	if err = (&metricsbinding.Reconciler{
		Client: mgr.GetClient(),
		Log:    logger,
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to create MetricsBinding controller: %v", err)
	}

	// +kubebuilder:scaffold:builder

	log.Debug("Starting agent for syncing multi-cluster objects")
	go mcagent.StartAgent(mgr.GetClient(), agentChannel, log)

	log.Info("Starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("Failed to run manager: %v", err)
	}
	return err
}
