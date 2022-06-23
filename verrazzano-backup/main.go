// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"fmt"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	log "github.com/verrazzano/verrazzano/verrazzano-backup/lib/klog"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/opensearch"
	model "github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/utils"
	"go.uber.org/zap"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"strings"
	"time"
)

var (
	VeleroBackupName string
	Component        string
	Operation        string
	Profile          string
)

func main() {
	flag.StringVar(&VeleroBackupName, "velero-backup-name", "", "The Velero-backup-name associated with this operation.")
	flag.StringVar(&Component, "", "", "The Verrazzano component to be backed up or restored (Default = opensearch).")
	flag.StringVar(&Operation, "operation", "", "The operation to be performed - backup/restore.")
	flag.StringVar(&Profile, "profile", "default", "Object store credentials profile")

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)

	// Flag validations
	if Operation == "" {
		fmt.Printf("Operation cannot be empty . It has to be 'backup/restore\n")
		os.Exit(1)
	}
	if Operation != constants.BackupOperation && Operation != constants.RestoreOperation {
		fmt.Printf("Operation has to be 'backup/restore\n")
		os.Exit(1)
	}
	if VeleroBackupName == "" {
		fmt.Printf("VeleroBackupName cannot be empty . It has to be set to an existing velero backup.\n")
		os.Exit(1)
	}

	// Auto detect component based on injection
	componentFound, err := utils.GetComponent(constants.ComponentPath)
	if err != nil {
		fmt.Printf("Component detection failure %v", err)
		os.Exit(1)
	}
	Component = componentFound

	// Initialize the zap log
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("verrazzano-%s-hook-*.log", strings.ToLower(Operation)))
	if err != nil {
		fmt.Printf("Unable to create temp file")
		os.Exit(1)
	}
	defer file.Close()
	log, err := log.Logger(file.Name())
	if err != nil {
		fmt.Printf("Unable to fetch logger")
		os.Exit(1)
	}
	log.Info("Verrazzano backup and restore helper invoked.")

	// Gathering k8s clients
	done := false
	retryCount := 0
	k8sContextReady := true
	var config *rest.Config
	var kubeClient *kubernetes.Clientset
	var clientk client.Client
	var dynClient dynamic.Interface
	//var search opensearch.Opensearch
	var cData model.ConnectionData
	cData.Timeout = utils.GetEnvWithDefault(constants.OpenSearchHealthCheckTimeoutKey, constants.OpenSearchHealthCheckTimeoutDefaultValue)
	timeParse, err := time.ParseDuration(cData.Timeout)
	if err != nil {
		log.Errorf("Unable to parse time duration ", zap.Error(err))
		os.Exit(1)
	}
	// Check OpenSearch health before proceeding with backup or restore
	search := opensearch.New(constants.OpenSearchURL, timeParse, http.DefaultClient)
	if Component == constants.OSComponent {
		err = search.EnsureOpenSearchIsHealthy(&cData, log)
		if err != nil {
			log.Errorf("Operation cannot be performed as OpenSearch is not healthy")
			os.Exit(1)
		}
	}

	// Feedback loop to gather k8s context
	for !done {
		config, err = ctrl.GetConfig()
		if err != nil {
			log.Errorf("Failed to get kubeconfig: %v", err)
			k8sContextReady = false
		}
		kubeClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Errorf("Failed to get clientset: %v", err)
			k8sContextReady = false
		}
		clientk, err = client.New(config, client.Options{})
		if err != nil {
			log.Errorf("Failed to get controller-runtime client: %v", err)
			k8sContextReady = false
		}
		dynClient, err = dynamic.NewForConfig(config)
		if err != nil {
			log.Errorf("Failed to get dynamic client: %v", err)
			k8sContextReady = false
		}

		if !k8sContextReady {
			if retryCount <= constants.RetryCount {
				message := "Unable to get context"
				_, err := utils.WaitRandom(message, cData.Timeout, log)
				if err != nil {
					log.Panic(err)
				}
				retryCount = retryCount + 1
			}
		} else {
			done = true
			log.Info("kubecontext retrieval successful")
		}

	}

	// Get S3 access details from Velero Backup Storage location associated with Backup given as input
	// Ensure the Backup Storage Location is NOT default
	k8s := utils.K8s(&utils.K8sImpl{})
	conData, err := k8s.PopulateConnData(dynClient, clientk, constants.VeleroNameSpace, VeleroBackupName, log)
	if err != nil {
		log.Errorf("Unable to fetch secret: %v", err)
		os.Exit(1)
	}

	// Update OpenSearch keystores
	_, err = search.UpdateKeystore(kubeClient, config, conData, log)
	if err != nil {
		log.Errorf("Unable to update keystore")
		os.Exit(1)
	}
	err = search.ReloadOpensearchSecureSettings(log)
	if err != nil {
		log.Errorf("Unable to reload security settings")
		os.Exit(1)
	}

	// Control flow based on component value
	switch strings.ToLower(Component) {
	case constants.OSComponent:
		// OpenSearch backup handling
		if strings.ToLower(Operation) == constants.BackupOperation {
			log.Info("Commencing opensearch backup ..")
			err = search.Backup(conData, log)
			if err != nil {
				log.Errorf("Operation '%s' unsuccessfull due to %v", Operation, zap.Error(err))
				os.Exit(1)
			}
			log.Infof("%s backup was successfull", strings.ToTitle(Component))
		}

		// OpenSearch restore handling
		if strings.ToLower(Operation) == constants.RestoreOperation {
			log.Infof("Commencing OpenSearch restore ..")
			err = k8s.ScaleDeployment(clientk, kubeClient, constants.VMOLabelSelector, constants.VerrazzanoNameSpaceName, constants.VMODeploymentName, int32(0), log)
			if err != nil {
				log.Errorf("Unable to scale deployment '%s' due to %v", constants.VMODeploymentName, zap.Error(err))
				os.Exit(1)
			}
			err = k8s.ScaleDeployment(clientk, kubeClient, constants.IngestLabelSelector, constants.VerrazzanoNameSpaceName, constants.IngestDeploymentName, int32(0), log)
			if err != nil {
				log.Errorf("Unable to scale deployment '%s' due to %v", constants.IngestDeploymentName, zap.Error(err))
				os.Exit(1)
			}
			err = search.Restore(conData, log)
			if err != nil {
				log.Errorf("Operation '%s' unsuccessfull due to %v", Operation, zap.Error(err))
				os.Exit(1)
			}

			ok, err := k8s.CheckDeployment(kubeClient, constants.KibanaDeploymentLabelSelector, constants.VerrazzanoNameSpaceName, log)
			if err != nil {
				log.Errorf("Unable to detect Kibana deployment '%s' due to %v", constants.KibanaDeploymentLabelSelector, zap.Error(err))
				os.Exit(1)
			}
			// If kibana is deployed then scale it down
			if ok {
				err = k8s.ScaleDeployment(clientk, kubeClient, constants.KibanaLabelSelector, constants.VerrazzanoNameSpaceName, constants.KibanaDeploymentName, int32(0), log)
				if err != nil {
					log.Errorf("Unable to scale deployment '%s' due to %v", constants.IngestDeploymentName, zap.Error(err))
				}
			}
			err = k8s.ScaleDeployment(clientk, kubeClient, constants.VMOLabelSelector, constants.VerrazzanoNameSpaceName, constants.VMODeploymentName, int32(1), log)
			if err != nil {
				log.Errorf("Unable to scale deployment '%s' due to %v", constants.VMODeploymentName, zap.Error(err))
			}

			err = k8s.CheckAllPodsAfterRestore(kubeClient, log)
			if err != nil {
				log.Errorf("Unable to check deployments after restoring Verrazzano Monitoring Operator %v", zap.Error(err))
			}

			log.Infof("%s restore was successfull", strings.ToTitle(Component))
		}
	}

}
