// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package operatorinit

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"os"

	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/constants"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/webhooks"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/validator"
	internalconfig "github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/netpolicy"
	"go.uber.org/zap"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// WebhookInit Webhook init container entry point
func WebhookInit(config internalconfig.OperatorConfig, log *zap.SugaredLogger) error {
	log.Debug("Creating certificates used by webhooks")

	conf, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return err
	}

	// Create the webhook certificates and secrets
	if err := certificate.CreateWebhookCertificates(log, kubeClient, config.CertDir); err != nil {
		return err
	}

	return nil
}

// StartWebhookServers Webhook startup entry point
func StartWebhookServers(config internalconfig.OperatorConfig, log *zap.SugaredLogger, scheme *runtime.Scheme) error {
	log.Debug("Creating certificates used by webhooks")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      config.MetricsAddr,
		Port:                    9443,
		LeaderElection:          config.LeaderElectionEnabled,
		LeaderElectionNamespace: constants.VerrazzanoInstallNamespace,
		LeaderElectionID:        "3ec4d295.verrazzano.io",
	})
	if err != nil {
		return fmt.Errorf("Error creating controller runtime manager: %v", err)
	}

	if err := updateWebhooks(log, mgr, config); err != nil {
		return err
	}

	installv1alpha1.SetComponentValidator(validator.ComponentValidatorImpl{})
	installv1beta1.SetComponentValidator(validator.ComponentValidatorImpl{})

	// +kubebuilder:scaffold:builder
	log.Info("Starting webhook controller-runtime manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("Failed starting webhook controller-runtime manager: %v", err)
	}
	return nil
}

// updateWebhooks Updates the webhook configurations and sets up the controllerruntime Manager for the webhook
func updateWebhooks(log *zap.SugaredLogger, mgr manager.Manager, config internalconfig.OperatorConfig) error {
	log.Infof("Start called for pod %s", os.Getenv("HOSTNAME"))
	conf, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("Failed to get kubeconfig: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return fmt.Errorf("Failed to get clientset: %v", err)
	}
	dynamicClient, err := dynamic.NewForConfig(conf)
	if err != nil {
		return fmt.Errorf("Failed to create Kubernetes dynamic client: %v", err)
	}

	if err := updateWebhookConfigurations(kubeClient, log, conf); err != nil {
		return err
	}
	if err := createOrUpdateNetworkPolicies(conf, log, kubeClient); err != nil {
		return err
	}
	if err := setupWebhooksWithManager(log, mgr, kubeClient, dynamicClient, config); err != nil {
		return err
	}
	return nil
}

// setupWebhooksWithManager Sets up the webhook with the controllerruntime Manager instance
func setupWebhooksWithManager(log *zap.SugaredLogger, mgr manager.Manager, kubeClient *kubernetes.Clientset, dynamicClient dynamic.Interface, config internalconfig.OperatorConfig) error {
	// Setup the validation webhook
	log.Debug("Setting up Verrazzano webhook with manager")
	if err := (&installv1alpha1.Verrazzano{}).SetupWebhookWithManager(mgr, log); err != nil {
		return fmt.Errorf("Failed to setup install.v1alpha1.Verrazzano webhook with manager: %v", err)
	}
	if err := (&installv1beta1.Verrazzano{}).SetupWebhookWithManager(mgr, log); err != nil {
		return fmt.Errorf("Failed to setup install.v1beta1.Verrazzano webhook with manager: %v", err)
	}

	mgr.GetWebhookServer().CertDir = config.CertDir

	// register MySQL backup job mutating webhook
	mgr.GetWebhookServer().Register(
		constants.MysqlBackupMutatingWebhookPath,
		&webhook.Admission{
			Handler: &webhooks.MySQLBackupJobWebhook{
				Client:        mgr.GetClient(),
				KubeClient:    kubeClient,
				DynamicClient: dynamicClient,
				Defaulters:    []webhooks.MySQLDefaulter{},
			},
		},
	)
	// register MySQL install values webhooks
	bomFile, err := bom.NewBom(internalconfig.GetDefaultBOMFilePath())
	if err != nil {
		return err
	}
	mgr.GetWebhookServer().Register(webhooks.MysqlInstallValuesV1beta1path, &webhook.Admission{Handler: &webhooks.MysqlValuesValidatorV1beta1{BomVersion: bomFile.GetVersion()}})
	mgr.GetWebhookServer().Register(webhooks.MysqlInstallValuesV1alpha1path, &webhook.Admission{Handler: &webhooks.MysqlValuesValidatorV1alpha1{BomVersion: bomFile.GetVersion()}})

	// Set up the validation webhook for VMC
	log.Debug("Setting up VerrazzanoManagedCluster webhook with manager")
	if err := (&clustersv1alpha1.VerrazzanoManagedCluster{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("Failed to setup webhook with manager: %v", err)
	}
	return nil
}

// updateWebhookConfigurations Creates or updates the webhook configurations as needed
func updateWebhookConfigurations(kubeClient *kubernetes.Clientset, log *zap.SugaredLogger, conf *rest.Config) error {
	log.Debug("Delete old VPO webhook configuration")
	if err := deleteValidatingWebhookConfiguration(kubeClient, certificate.OldOperatorName); err != nil {
		return fmt.Errorf("Failed to delete old webhook configuration: %v", err)
	}

	log.Debug("Updating VPO webhook configuration")

	if err := updateValidatingWebhookConfiguration(kubeClient, certificate.OperatorName); err != nil {
		return fmt.Errorf("Failed to update validation webhook configuration: %v", err)
	}

	log.Debug("Updating conversion webhook")
	apixClient, err := apiextensionsv1client.NewForConfig(conf)
	if err != nil {
		return fmt.Errorf("Failed to get apix clientset: %v", err)
	}

	if err := updateConversionWebhookConfiguration(apixClient, kubeClient); err != nil {
		return fmt.Errorf("Failed to update conversion webhook: %v", err)
	}

	if err := updateMutatingWebhookConfiguration(kubeClient, constants.MysqlBackupMutatingWebhookName); err != nil {
		return fmt.Errorf("Failed to update pod mutating webhook configuration: %v", err)
	}

	log.Debug("Updating MySQL install values webhook configuration")
	if err := updateValidatingWebhookConfiguration(kubeClient, webhooks.MysqlInstallValuesWebhook); err != nil {
		return fmt.Errorf("Failed to update validation webhook configuration: %v", err)
	}
	return nil
}

// createOrUpdateNetworkPolicies Create or update the network policies required by the operator and webhooks
func createOrUpdateNetworkPolicies(conf *rest.Config, log *zap.SugaredLogger, kubeClient *kubernetes.Clientset) error {
	c, err := client.New(conf, client.Options{})
	if err != nil {
		return errors.Wrap(err, "Failed to get controller-runtime client")
	}

	log.Debug("Creating or updating network policies")
	var netPolErrors []error
	_, netPolErrors = netpolicy.CreateOrUpdateNetworkPolicies(kubeClient, c)
	if len(netPolErrors) > 0 {
		// Seems like this could make for an unreadable set of errors; may need to revisit
		return fmt.Errorf("Failed to create or update network policies: %v", netPolErrors)
	}
	return nil
}
