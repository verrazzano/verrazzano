// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"strings"
	"text/template"
	"time"
)

// CreateVeleroBackupLocationObject creates velero backup object location
func CreateVeleroBackupLocationObject(backupStorageName, backupSecretName string, log *zap.SugaredLogger) error {
	var b bytes.Buffer
	template, _ := template.New("velero-backup-location").Parse(VeleroBackupLocation)

	data := VeleroBackupLocationObjectData{
		VeleroBackupStorageName:          backupStorageName,
		VeleroNamespaceName:              VeleroNameSpace,
		VeleroObjectStoreBucketName:      OciBucketName,
		VeleroSecretName:                 backupSecretName,
		VeleroObjectStorageNamespaceName: OciNamespaceName,
		VeleroBackupRegion:               BackupRegion,
	}
	template.Execute(&b, data)
	err := pkg.CreateOrUpdateResourceFromBytes(b.Bytes())
	if err != nil {
		log.Errorf("Error creating velero backup location ", zap.Error(err))
		return err
	}
	return nil
}

// GetRancherBackupFileName gets the filename backed up to object store
func GetRancherBackupFileName(backupName string, log *zap.SugaredLogger) (string, error) {

	log.Infof("Fetching uploaded filename from backup '%s'", backupName)
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return "", err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return "", err
	}

	gvr := schema.GroupVersionResource{
		Group:    "resources.cattle.io",
		Version:  "v1",
		Resource: "backups",
	}

	backupFetched, err := dclient.Resource(gvr).Get(context.TODO(), backupName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to fetch Rancher backup '%s' due to '%v'", backupName, zap.Error(err))
		return "", err
	}

	if backupFetched == nil {
		log.Infof("No Rancher backup with name '%s'' was detected", backupName)
	}

	var backup RancherBackupModel
	bdata, err := json.Marshal(backupFetched)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return "", err
	}
	err = json.Unmarshal(bdata, &backup)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return "", err
	}
	return backup.Status.Filename, nil
}

// GetVeleroBackup Retrieves Velero backup object from the cluster
func GetVeleroBackup(namespace, backupName string, log *zap.SugaredLogger) (*VeleroBackupModel, error) {

	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return nil, err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return nil, err
	}

	log.Infof("Fetching Velero backup '%s' in namespace '%s'", backupName, namespace)
	gvr := schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backups",
	}

	backupFetched, err := dclient.Resource(gvr).Namespace(namespace).Get(context.TODO(), backupName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to fetch Velero backup '%s' due to '%v'", backupName, zap.Error(err))
		return nil, err
	}

	if backupFetched == nil {
		log.Infof("No Velero backup with name '%s' in namespace '%s' was detected", backupName, namespace)
	}

	var backup VeleroBackupModel
	bdata, err := json.Marshal(backupFetched)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(bdata, &backup)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return nil, err
	}

	return &backup, nil
}

// GetVeleroRestore Retrieves Velero backup object from the cluster
func GetVeleroRestore(namespace, restoreName string, log *zap.SugaredLogger) (*VeleroRestoreModel, error) {

	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return nil, err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return nil, err
	}

	log.Infof("Fetching Velero restore '%s' in namespace '%s'", restoreName, namespace)
	gvr := schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "restores",
	}

	backupFetched, err := dclient.Resource(gvr).Namespace(namespace).Get(context.TODO(), restoreName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to fetch Velero restore '%s' due to '%v'", restoreName, zap.Error(err))
		return nil, err
	}

	if backupFetched == nil {
		log.Infof("No Velero restore with name '%s' in namespace '%s' was detected", restoreName, namespace)
	}

	var restore VeleroRestoreModel
	bdata, err := json.Marshal(backupFetched)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(bdata, &restore)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return nil, err
	}

	return &restore, nil
}

func GetRancherBackup(backupName string, log *zap.SugaredLogger) (*RancherBackupModel, error) {

	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return nil, err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return nil, err
	}

	log.Infof("Fetching Rancher backup '%s'", backupName)
	gvr := schema.GroupVersionResource{
		Group:    "resources.cattle.io",
		Version:  "v1",
		Resource: "backups",
	}

	backupFetched, err := dclient.Resource(gvr).Get(context.TODO(), backupName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to fetch Rancher backup '%s' due to '%v'", backupName, zap.Error(err))
		return nil, err
	}

	if backupFetched == nil {
		log.Infof("No Rancher backup with name '%s'' was detected", backupName)
	}

	var backup RancherBackupModel
	bdata, err := json.Marshal(backupFetched)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(bdata, &backup)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return nil, err
	}

	return &backup, nil
}

func GetRancherRestore(restoreName string, log *zap.SugaredLogger) (*RancherRestoreModel, error) {

	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return nil, err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return nil, err
	}

	log.Infof("Fetching Rancher restore '%s'", restoreName)
	gvr := schema.GroupVersionResource{
		Group:    "resources.cattle.io",
		Version:  "v1",
		Resource: "restores",
	}

	backupFetched, err := dclient.Resource(gvr).Get(context.TODO(), restoreName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to fetch Rancher restore '%s' due to '%v'", restoreName, zap.Error(err))
		return nil, err
	}

	if backupFetched == nil {
		log.Infof("No Rancher restore with name '%s'' was detected", restoreName)
	}

	var restore RancherRestoreModel
	bdata, err := json.Marshal(backupFetched)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(bdata, &restore)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return nil, err
	}

	return &restore, nil
}

func TrackOperationProgress(retryLimit int, operator, operation, objectName, namespace string, log *zap.SugaredLogger) error {
	var response string
	retryCount := 0
	for {
		if retryCount > retryLimit {
			return fmt.Errorf("retry count execeeded while checking progress for %s '%s'", operation, objectName)
		}

		switch operator {
		case "velero":
			switch operation {
			case "backups":
				backupInfo, err := GetVeleroBackup(namespace, objectName, log)
				if err != nil {
					log.Errorf("Unable to fetch backup '%s' due to '%v'", objectName, zap.Error(err))
				}
				response = backupInfo.Status.Phase

			case "restores":
				restoreInfo, err := GetVeleroRestore(namespace, objectName, log)
				if err != nil {
					log.Errorf("Unable to fetch restore '%s' due to '%v'", objectName, zap.Error(err))
				}
				response = restoreInfo.Status.Phase
			default:
				log.Errorf("Invalid operation specified for Velero = %s'", operation)
				response = "NAN"
			}

		case "rancher":
			switch operation {
			case "backups":
				backupInfo, err := GetRancherBackup(objectName, log)
				if err != nil {
					log.Errorf("Unable to fetch backup '%s' due to '%v'", objectName, zap.Error(err))
				}

				if backupInfo.Status.Conditions != nil {
					for _, cond := range backupInfo.Status.Conditions {
						if cond.Type == "Ready" {
							response = cond.Message
						} else {
							log.Infof("Rancher backup status : Type = %v, Status = %v", cond.Type, cond.Status)
						}
					}
				}

			case "restores":
				restoreInfo, err := GetRancherRestore(objectName, log)
				if err != nil {
					log.Errorf("Unable to fetch restore '%s' due to '%v'", objectName, zap.Error(err))
				}
				if restoreInfo.Status.Conditions != nil {
					for _, cond := range restoreInfo.Status.Conditions {
						if cond.Type == "Ready" {
							response = cond.Message
						} else {
							log.Infof("Rancher restore status : Type = %v, Status = %v", cond.Type, cond.Status)
						}
					}
				}

			default:
				log.Errorf("Invalid operation specified for Rancher = %s'", operation)
				response = "NAN"
			}

		default:
			log.Errorf("Invalid operator specified = %s'", operator)
			response = "NAN"

		}

		switch response {
		case "InProgress", "":
			log.Infof("%s '%s' is in progress. Check back after 60 seconds. (Retry count left = %v).", strings.ToTitle(operation), objectName, retryLimit-retryCount)
			time.Sleep(60 * time.Second)
		case "Completed":
			log.Infof("%s '%s' completed successfully", strings.ToTitle(operation), objectName)
			return nil
		default:
			return fmt.Errorf("%s failed. State = '%s'", strings.ToTitle(operation), response)
		}
		retryCount = retryCount + 1
	}

}

func CrdPruner(group, version, resource, resourceName, nameSpaceName string, log *zap.SugaredLogger) error {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return err
	}

	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	if strings.Contains(group, "velero") {
		err = dclient.Resource(gvr).Namespace(nameSpaceName).Delete(context.TODO(), resourceName, metav1.DeleteOptions{})
		if !k8serror.IsNotFound(err) {
			log.Errorf("Unable to delete resource '%s' from namespace '%s' due to '%v'", resourceName, nameSpaceName, zap.Error(err))
			return err
		}
	}

	if strings.Contains(group, "cattle") {
		err = dclient.Resource(gvr).Delete(context.TODO(), resourceName, metav1.DeleteOptions{})
		if !k8serror.IsNotFound(err) {
			log.Errorf("Unable to delete resource '%s' due to '%v'", resourceName, zap.Error(err))
			return err
		}
	}
	return nil
}
