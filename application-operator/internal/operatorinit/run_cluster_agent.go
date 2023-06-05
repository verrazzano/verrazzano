// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterapplicationconfiguration"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclustercomponent"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclusterconfigmap"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/multiclustersecret"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters/verrazzanoproject"
	"github.com/verrazzano/verrazzano/application-operator/mcagent"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	vmcclient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/scheme"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func StartClusterAgent(metricsAddr string, enableLeaderElection bool, log *zap.SugaredLogger, scheme *runtime.Scheme) error {
	mgr, err := ctrl.NewManager(k8sutil.GetConfigOrDieFromController(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "5df248b5.verrazzano.io",
	})
	if err != nil {
		log.Errorf("Failed to start manager: %v", err)
		return err
	}
	log.Info("Starting agent reconciler for syncing multi-cluster objects")
	agentChannel, err := setupClusterAgentReconciler(mgr, log)
	if err != nil {
		return err
	}
	log.Info("Starting multicluster reconcilers")
	if err := setupMulticlusterReconcilers(mgr, agentChannel, log); err != nil {
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

func setupClusterAgentReconciler(mgr manager.Manager, log *zap.SugaredLogger) (chan clusters.StatusUpdateMessage, error) {
	// Create a buffered channel of size 10 for the multi cluster agent to receive messages
	agentChannel := make(chan clusters.StatusUpdateMessage, constants.StatusUpdateChannelBufferSize)

	if err := (&mcagent.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log.With(vzlog.FieldAgent, "multi-cluster"),
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create managed cluster agent controller: %v", err)
		return nil, err
	}
	return agentChannel, nil
}

func setupMulticlusterReconcilers(mgr manager.Manager, agentChannel chan clusters.StatusUpdateMessage, log *zap.SugaredLogger) error {
	if err := (&multiclustersecret.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterSecretKind, err)
		return err
	}
	if err := (&multiclustercomponent.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterComponentKind, err)
		return err
	}
	if err := (&multiclusterconfigmap.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller %v", clustersv1alpha1.MultiClusterConfigMapKind, err)
		return err
	}
	if err := (&multiclusterapplicationconfiguration.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       mgr.GetScheme(),
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller: %v", clustersv1alpha1.MultiClusterAppConfigKind, err)
		return err
	}
	scheme := mgr.GetScheme()
	vmcclient.AddToScheme(scheme)
	if err := (&verrazzanoproject.Reconciler{
		Client:       mgr.GetClient(),
		Log:          log,
		Scheme:       scheme,
		AgentChannel: agentChannel,
	}).SetupWithManager(mgr); err != nil {
		log.Errorf("Failed to create %s controller %v", clustersv1alpha1.VerrazzanoProjectKind, err)
		return err
	}
	return nil
}
