// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps/components"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/configmaps/overrides"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/secrets"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/mysqlcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/reconcile"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
	"sync"
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
