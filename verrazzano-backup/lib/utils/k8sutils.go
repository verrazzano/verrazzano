// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	model "github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
	"go.uber.org/zap"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

type K8s interface {
	PopulateConnData(dclient dynamic.Interface, client client.Client, veleroNamespace, backupName string, log *zap.SugaredLogger) (*model.ConnectionData, error)
	GetObjectStoreCreds(client client.Client, secretName, namespace, secretKey string, log *zap.SugaredLogger) (*model.ObjectStoreSecret, error)
	GetBackup(client dynamic.Interface, veleroNamespace, backupName string, log *zap.SugaredLogger) (*model.VeleroBackup, error)
	GetBackupStorageLocation(client dynamic.Interface, veleroNamespace, bslName string, log *zap.SugaredLogger) (*model.VeleroBackupStorageLocation, error)
	ScaleDeployment(clientk client.Client, k8sclient *kubernetes.Clientset, labelSelector, namespace, deploymentName string, replicaCount int32, log *zap.SugaredLogger) error
	CheckDeployment(k8sclient kubernetes.Interface, labelSelector, namespace string, log *zap.SugaredLogger) (bool, error)
	CheckPodStatus(k8sclient kubernetes.Interface, podName, namespace, checkFlag string, timeout string, log *zap.SugaredLogger, wg *sync.WaitGroup) error
	CheckAllPodsAfterRestore(k8sclient kubernetes.Interface, log *zap.SugaredLogger) error
	IsPodReady(pod *v1.Pod, log *zap.SugaredLogger) (bool, error)
}

type K8sImpl struct {
}

//PopulateConnData creates the connection object thats used to communicate to object store
func (k *K8sImpl) PopulateConnData(dclient dynamic.Interface, client client.Client, veleroNamespace, backupName string, log *zap.SugaredLogger) (*model.ConnectionData, error) {
	log.Infof("Populating connection data from backup '%v' in namespace '%s'", backupName, veleroNamespace)

	backup, err := k.GetBackup(dclient, veleroNamespace, backupName, log)
	if err != nil {
		return nil, err
	}

	if backup.Spec.StorageLocation == "default" {
		log.Infof("Default creds not supported. Custom credentaisl needs to be created before creating backup storage location")
		return nil, err
	}

	log.Infof("Detected velero backup storage location '%s' in namespace '%s' used by backup '%s'", backup.Spec.StorageLocation, veleroNamespace, backupName)
	bsl, err := k.GetBackupStorageLocation(dclient, veleroNamespace, backup.Spec.StorageLocation, log)
	if err != nil {
		return nil, err
	}

	secretData, err := k.GetObjectStoreCreds(client, bsl.Spec.Credential.Name, bsl.Metadata.Namespace, bsl.Spec.Credential.Key, log)
	if err != nil {
		return nil, err
	}

	var conData model.ConnectionData
	conData.Secret = *secretData
	conData.RegionName = bsl.Spec.Config.Region
	conData.Endpoint = bsl.Spec.Config.S3URL
	conData.BucketName = bsl.Spec.ObjectStorage.Bucket
	conData.BackupName = backupName
	// For now, we will look at the first POST hook in the first Hook in Backup
	conData.Timeout = backup.Spec.Hooks.Resources[0].Post[0].Exec.Timeout

	return &conData, nil

}

//GetObjectStoreCreds - Fetches credentials from Velero Backup object store location.
//This object will be pre-created before the execution of this hook
func (k *K8sImpl) GetObjectStoreCreds(client client.Client, secretName, namespace, secretKey string, log *zap.SugaredLogger) (*model.ObjectStoreSecret, error) {
	secret := v1.Secret{}
	if err := client.Get(context.TODO(), crtclient.ObjectKey{Name: secretName, Namespace: namespace}, &secret); err != nil {
		log.Errorf("Failed to retrieve secret '%s' due to : %v", secretName, err)
		return nil, err
	}

	file, err := CreateTempFileWithData(secret.Data[secretKey])
	if err != nil {
		return nil, err
	}
	defer os.Remove(file)

	accessKey, secretAccessKey, err := ReadTempCredsFile(file)
	if err != nil {
		log.Error("Error while reading creds from file ", zap.Error(err))
		return nil, err
	}

	var secretData model.ObjectStoreSecret
	secretData.SecretName = secretName
	secretData.SecretKey = secretKey
	secretData.ObjectAccessKey = accessKey
	secretData.ObjectSecretKey = secretAccessKey
	return &secretData, nil
}

//GetBackupStorageLocation - Retrieves the backup storage location from the backup storage location
func (k *K8sImpl) GetBackupStorageLocation(client dynamic.Interface, veleroNamespace, bslName string, log *zap.SugaredLogger) (*model.VeleroBackupStorageLocation, error) {
	log.Infof("Fetching velero backup storage location '%s' in namespace '%s'", bslName, veleroNamespace)
	gvr := schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backupstoragelocations",
	}
	bslRecievd, err := client.Resource(gvr).Namespace(veleroNamespace).Get(context.Background(), bslName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if bslRecievd == nil {
		log.Infof("No velero backup storage location in namespace '%s' was detected", veleroNamespace)
		return nil, err
	}

	var bsl model.VeleroBackupStorageLocation
	bdata, err := json.Marshal(bslRecievd)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bdata, &bsl)
	if err != nil {
		return nil, err
	}
	return &bsl, nil
}

//GetBackup - Retrives velero backups in the cluster
func (k *K8sImpl) GetBackup(client dynamic.Interface, veleroNamespace, backupName string, log *zap.SugaredLogger) (*model.VeleroBackup, error) {
	log.Infof("Fetching velero backup '%s' in namespace '%s'", backupName, veleroNamespace)
	gvr := schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backups",
	}
	backupFetched, err := client.Resource(gvr).Namespace(veleroNamespace).Get(context.Background(), backupName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if backupFetched == nil {
		log.Infof("No velero backup in namespace '%s' was detected", veleroNamespace)
		return nil, err
	}

	var backup model.VeleroBackup
	bdata, err := json.Marshal(backupFetched)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bdata, &backup)
	if err != nil {
		return nil, err
	}
	return &backup, nil
}

// ScaleDeployment is used to scale deployment to specific replica count
// labelselectors,namespace, deploymentName are used to identify deployments and specific pods associated with them
func (k *K8sImpl) ScaleDeployment(clientk client.Client, k8sclient *kubernetes.Clientset, labelSelector, namespace, deploymentName string, replicaCount int32, log *zap.SugaredLogger) error {
	log.Infof("Scale deployment '%s' in namespace '%s", deploymentName, namespace)
	var wg sync.WaitGroup
	depPatch := apps.Deployment{}
	if err := clientk.Get(context.TODO(), types.NamespacedName{Name: deploymentName, Namespace: namespace}, &depPatch); err != nil {
		return err
	}
	currentValue := *depPatch.Spec.Replicas
	desiredValue := replicaCount

	if desiredValue == currentValue {
		log.Infof("Deployment scaling skipped as desired replicas is same as current replicas")
		return nil
	}

	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	pods, err := k8sclient.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	wg.Add(len(pods.Items))

	mergeFromDep := client.MergeFrom(depPatch.DeepCopy())
	depPatch.Spec.Replicas = &replicaCount
	if err := clientk.Patch(context.TODO(), &depPatch, mergeFromDep); err != nil {
		log.Error("Unable to patch !!")
		return err
	}

	timeout := GetEnvWithDefault(constants.OpenSearchHealthCheckTimeoutKey, constants.OpenSearchHealthCheckTimeoutDefaultValue)

	if desiredValue > currentValue {
		//log.Info("Scaling up pods ...")
		message := "Wait for pods to come up"
		_, err := WaitRandom(message, timeout, log)
		if err != nil {
			return err
		}

		for _, item := range pods.Items {
			log.Debugf("Firing go routine to check on pod '%s'", item.Name)
			go k.CheckPodStatus(k8sclient, item.Name, namespace, "up", timeout, log, &wg)
		}
	}

	if desiredValue < currentValue {
		log.Info("Scaling down pods ...")
		for _, item := range pods.Items {
			log.Debugf("Firing go routine to check on pod '%s'", item.Name)
			go k.CheckPodStatus(k8sclient, item.Name, namespace, "down", timeout, log, &wg)
		}
	}

	wg.Wait()
	log.Infof("Successfully scaled deployment '%s' in namespace '%s' from '%v' to '%v' replicas ", deploymentName, namespace, currentValue, replicaCount)
	return nil

}

// CheckDeployment checks the existence of a deployment
func (k *K8sImpl) CheckDeployment(k8sclient kubernetes.Interface, labelSelector, namespace string, log *zap.SugaredLogger) (bool, error) {
	log.Infof("Checking deployment with labelselector '%v' exists in namespace '%s", labelSelector, namespace)
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	deployment, err := k8sclient.AppsV1().Deployments(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return false, err
	}

	// There should be one deployment of kibana
	if len(deployment.Items) == 1 {
		return true, nil
	}
	return false, nil
}

// IsPodReady checks whether pod is Ready
func (k *K8sImpl) IsPodReady(pod *v1.Pod, log *zap.SugaredLogger) (bool, error) {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			log.Infof("Pod '%s' in namespace '%s' is now in '%s' state", pod.Name, pod.Namespace, condition.Type)
			return true, nil
		}
	}
	log.Infof("Pod '%s' in namespace '%s' is still not Ready", pod.Name, pod.Namespace)
	return false, nil
}

// CheckPodStatus checks the state of the pod depending on checkFlag
func (k *K8sImpl) CheckPodStatus(k8sclient kubernetes.Interface, podName, namespace, checkFlag string, timeout string, log *zap.SugaredLogger, wg *sync.WaitGroup) error {
	log.Infof("Checking Pod '%s' status in namespace '%s", podName, namespace)
	var timeSeconds float64
	defer wg.Done()
	timeParse, err := time.ParseDuration(timeout)
	if err != nil {
		log.Errorf("Unable to parse time duration ", zap.Error(err))
		return err
	}
	totalSeconds := timeParse.Seconds()
	done := false
	wait := false

	for !done {
		pod, err := k8sclient.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if pod == nil && checkFlag == "down" {
			// break loop when scaling down condition is met
			log.Infof("Pod '%s' has scaled down successfully", pod.Name)
			done = true
		}

		// If pod is found
		if pod != nil {
			switch checkFlag {
			case "up":
				pod.Status.Conditions[0].Type = "Ready"
				// Check status and apply retry logic
				if pod.Status.Phase != "Running" {
					wait = true
				} else {
					// break loop when scaling up condition is met
					log.Infof("Pod '%s' is in 'Running' state", pod.Name)
					ok, err := k.IsPodReady(pod, log)
					if err != nil {
						return err
					}
					if ok {
						// break loop pod is Running and pod is in Ready !!
						done = true
					}

				}
			case "down":
				wait = true
			}

			if wait {
				fmt.Printf("timeSeconds = %v, totalSeconds = %v ", timeSeconds, totalSeconds)
				if timeSeconds < totalSeconds {
					message := fmt.Sprintf("Pod '%s' is in '%s' state", pod.Name, pod.Status.Phase)
					duration, err := WaitRandom(message, timeout, log)
					if err != nil {
						return err
					}
					timeSeconds = timeSeconds + float64(duration)

				} else {
					return fmt.Errorf("Timeout '%s' exceeded. Pod '%s' is still not in running state", timeout, pod.Name)
				}
				// change wait to false after each wait
				wait = false
			}
		}
	}
	return nil
}

// CheckDeployment checks the existence of a deployment
func (k *K8sImpl) CheckAllPodsAfterRestore(k8sclient kubernetes.Interface, log *zap.SugaredLogger) error {
	timeout := GetEnvWithDefault(constants.OpenSearchHealthCheckTimeoutKey, constants.OpenSearchHealthCheckTimeoutDefaultValue)

	message := "Waiting for Verrazzano Monitoring Operator to come up"
	_, err := WaitRandom(message, timeout, log)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	log.Infof("Checking pods with labelselector '%v' in namespace '%s", constants.IngestLabelSelector, constants.VerrazzanoNameSpaceName)
	listOptions := metav1.ListOptions{LabelSelector: constants.IngestLabelSelector}
	ingestPods, err := k8sclient.CoreV1().Pods(constants.VerrazzanoNameSpaceName).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	wg.Add(len(ingestPods.Items))
	for _, pod := range ingestPods.Items {
		log.Debugf("Firing go routine to check on pod '%s'", pod.Name)
		go k.CheckPodStatus(k8sclient, pod.Name, constants.VerrazzanoNameSpaceName, "up", timeout, log, &wg)
	}

	log.Infof("Checking pods with labelselector '%v' in namespace '%s", constants.KibanaLabelSelector, constants.VerrazzanoNameSpaceName)
	listOptions = metav1.ListOptions{LabelSelector: constants.KibanaLabelSelector}
	kibanaPods, err := k8sclient.CoreV1().Pods(constants.VerrazzanoNameSpaceName).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	wg.Add(len(kibanaPods.Items))
	for _, pod := range kibanaPods.Items {
		log.Debugf("Firing go routine to check on pod '%s'", pod.Name)
		go k.CheckPodStatus(k8sclient, pod.Name, constants.VerrazzanoNameSpaceName, "up", timeout, log, &wg)
	}

	wg.Wait()
	return nil
}
