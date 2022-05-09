// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils

import (
	"context"
	"encoding/json"
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
	"time"
)

type K8s interface {
	PopulateConnData(dclient dynamic.Interface, client client.Client, veleroNamespace, backupName string, log *zap.SugaredLogger) (*model.ConnectionData, error)
	GetObjectStoreCreds(client client.Client, secretName, namespace, secretKey string, log *zap.SugaredLogger) (*model.ObjectStoreSecret, error)
	GetBackup(client dynamic.Interface, veleroNamespace, backupName string, log *zap.SugaredLogger) (*model.VeleroBackup, error)
	GetBackupStorageLocation(client dynamic.Interface, veleroNamespace, bslName string, log *zap.SugaredLogger) (*model.VeleroBackupStorageLocation, error)
	ScaleDeployment(clientk client.Client, k8sclient *kubernetes.Clientset, labelSelector, namespace, deploymentName string, replicaCount int32, log *zap.SugaredLogger) error
}

type K8sImpl struct {
}

//PopulateConnData crestes the connection object thats used to communicate to object store
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

//ScaleDeployment is used to scale deployment to specific replica count
// labelselectors,namespace, deploymentName are used to identify deployments and specific pods associated with them
func (k *K8sImpl) ScaleDeployment(clientk client.Client, k8sclient *kubernetes.Clientset, labelSelector, namespace, deploymentName string, replicaCount int32, log *zap.SugaredLogger) error {
	log.Infof("Scale deployment '%s' in namespace '%s", deploymentName, namespace)
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

	done := false
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	var podStateCondition []bool
	var podNames []string
	pods, err := k8sclient.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	mergeFromDep := client.MergeFrom(depPatch.DeepCopy())
	depPatch.Spec.Replicas = &replicaCount
	if err := clientk.Patch(context.TODO(), &depPatch, mergeFromDep); err != nil {
		log.Error("Unable to patch !!")
		return err
	}

	for !done {
		//pods, err := k8sclient.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
		//if err != nil {
		//	return err
		//}

		//Scale up
		if desiredValue > currentValue {
			log.Info("Scaling up pods ...")
			// There could be multiple pods in a deployment
			for _, item := range pods.Items {
				if item.Status.Phase == "Running" {
					podStateCondition = append(podStateCondition, true)
				}
				podNames = append(podNames, item.Name)
			}

			if int32(len(pods.Items)) == desiredValue && int32(len(podStateCondition)) == desiredValue {
				// when all running pods is equal to desired input
				// exit the check loop
				log.Infof("Actual pod count = '%v', Desired pod count = '%v'", len(pods.Items), desiredValue)
				done = true
			} else {
				// otherwise retry and keep monitoring the pod status
				duration := GenerateRandom()
				log.Infof("Waiting for '%v' seconds for following '%v' pods to come up.", duration, podNames)
				time.Sleep(time.Second * time.Duration(duration))
			}
		}

		// scale down
		if desiredValue < currentValue {
			for _, item := range pods.Items {
				// populate podNames for display
				podNames = append(podNames, item.Name)
			}

			log.Info("Scaling down pods ..")
			if int32(len(pods.Items)) != desiredValue {
				duration := GenerateRandom()
				log.Infof("Waiting for '%v' seconds for following '%v' pods  pods to go down.", duration, podNames)
				time.Sleep(time.Second * time.Duration(duration))
			} else {
				// when all running pods is equal to desired input
				// exit the check loop
				log.Infof("Actual pod count = '%v', Desired pod count = '%v'", len(pods.Items), desiredValue)
				done = true
			}
		}

		pods, err = k8sclient.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return err
		}

	}

	log.Infof("Successfully scaled deployment '%s' in namespace '%s' from '%v' to '%v' replicas ", deploymentName, namespace, currentValue, replicaCount)
	return nil

}
