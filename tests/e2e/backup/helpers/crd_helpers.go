// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
)

// getUnstructuredData common utility to fetch unstructured data
func getUnstructuredData(group, version, resource, resourceName, nameSpaceName, component string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	var dataFetched *unstructured.Unstructured
	var err error
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

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	if nameSpaceName != "" {
		log.Infof("Fetching '%s' '%s' '%s' in namespace '%s'", component, resource, resourceName, nameSpaceName)
		dataFetched, err = dclient.Resource(gvr).Namespace(nameSpaceName).Get(context.TODO(), resourceName, metav1.GetOptions{})
	} else {
		log.Infof("Fetching '%s' '%s' '%s'", component, resource, resourceName)
		dataFetched, err = dclient.Resource(gvr).Get(context.TODO(), resourceName, metav1.GetOptions{})
	}
	if err != nil {
		log.Errorf("Unable to fetch %s %s %s due to '%v'", component, resource, resourceName, zap.Error(err))
		return nil, err
	}
	return dataFetched, nil
}

// getUnstructuredData common utility to fetch list of unstructured data
func getUnstructuredDataList(group, version, resource, nameSpaceName, component string, log *zap.SugaredLogger) (*unstructured.UnstructuredList, error) {
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

	log.Infof("Fetching %s %s", component, resource)
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	dataFetched, err := dclient.Resource(gvr).Namespace(nameSpaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Unable to fetch %s %s due to '%v'", component, resource, zap.Error(err))
		return nil, err
	}
	return dataFetched, nil
}

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
	backupFetched, err := getUnstructuredData("resources.cattle.io", "v1", "backups", backupName, "", "rancher", log)
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
	backupFetched, err := getUnstructuredData("velero.io", "v1", "backups", backupName, namespace, "velero", log)
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

// GetVeleroRestore Retrieves Velero restore object from the cluster
func GetVeleroRestore(namespace, restoreName string, log *zap.SugaredLogger) (*VeleroRestoreModel, error) {

	restoreFetched, err := getUnstructuredData("velero.io", "v1", "restores", restoreName, namespace, "velero", log)
	if err != nil {
		log.Errorf("Unable to fetch velero restore '%s' due to '%v'", restoreName, zap.Error(err))
		return nil, err
	}

	if restoreFetched == nil {
		log.Infof("No Velero restore with name '%s' in namespace '%s' was detected", restoreName, namespace)
	}

	var restore VeleroRestoreModel
	bdata, err := json.Marshal(restoreFetched)
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

// GetPodVolumeBackups Retrieves Velero pod volume backups object from the cluster
func GetPodVolumeBackups(namespace string, log *zap.SugaredLogger) error {

	podVolumeBackupsFetched, err := getUnstructuredDataList("velero.io", "v1", BackupPodVolumeResource, namespace, "velero", log)
	if err != nil {
		log.Errorf("Unable to fetch velero podvolumebackups due to '%v'", zap.Error(err))
		return err
	}

	if podVolumeBackupsFetched == nil {
		log.Infof("No Velero podvolumebackups in namespace '%s' was detected ", namespace)
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tStatus\tNamespace\tPod\tVolume\tRepo\tStorage")
	for _, item := range podVolumeBackupsFetched.Items {
		var podVolumeBackup VeleroPodVolumeBackups
		bdata, err := json.Marshal(item.Object)
		if err != nil {
			log.Errorf("Json marshalling error %v", zap.Error(err))
			return err
		}
		err = json.Unmarshal(bdata, &podVolumeBackup)
		if err != nil {
			log.Errorf("Json unmarshall error %v", zap.Error(err))
			return err
		}
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v", podVolumeBackup.Metadata.Name, podVolumeBackup.Status.Phase,
			podVolumeBackup.Metadata.Namespace, podVolumeBackup.Spec.Pod.Name,
			podVolumeBackup.Spec.Volume, podVolumeBackup.Spec.RepoIdentifier, podVolumeBackup.Spec.BackupStorageLocation))
	}
	writer.Flush()
	return nil
}

// GetPodVolumeRestores Retrieves Velero pod volume restores object from the cluster
func GetPodVolumeRestores(namespace string, log *zap.SugaredLogger) error {

	restoreFetched, err := getUnstructuredDataList("velero.io", "v1", RestorePodVolumeResource, namespace, "velero", log)
	if err != nil {
		log.Errorf("Unable to fetch velero podvolumebackups due to '%v'", zap.Error(err))
		return err
	}

	if restoreFetched == nil {
		log.Infof("No Velero podvolumebackups in namespace '%s' was detected ", namespace)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tnamespace\tPod\tVolume\tStatus\tTotalBytes\tBytesDone")
	for _, item := range restoreFetched.Items {
		var podVolumeRestore VeleroPodVolumeRestores
		bdata, err := json.Marshal(item.Object)
		if err != nil {
			log.Errorf("Json marshalling error %v", zap.Error(err))
			return err
		}
		err = json.Unmarshal(bdata, &podVolumeRestore)
		if err != nil {
			log.Errorf("Json unmarshall error %v", zap.Error(err))
			return err
		}
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v", podVolumeRestore.Metadata.Name,
			podVolumeRestore.Metadata.Namespace, podVolumeRestore.Spec.Pod.Name, podVolumeRestore.Spec.Volume,
			podVolumeRestore.Status.Phase, podVolumeRestore.Status.Progress.TotalBytes, podVolumeRestore.Status.Progress.BytesDone))
	}
	writer.Flush()
	return nil
}

// GetRancherBackup Retrieves rancher backup object from the cluster
func GetRancherBackup(backupName string, log *zap.SugaredLogger) (*RancherBackupModel, error) {

	backupFetched, err := getUnstructuredData("resources.cattle.io", "v1", "backups", backupName, "", "rancher", log)
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

// GetRancherRestore Retrieves rancher restore object from the cluster
func GetRancherRestore(restoreName string, log *zap.SugaredLogger) (*RancherRestoreModel, error) {

	restoreFetched, err := getUnstructuredData("resources.cattle.io", "v1", "restores", restoreName, "", "rancher", log)
	if err != nil {
		log.Errorf("Unable to fetch Rancher restore '%s' due to '%v'", restoreName, zap.Error(err))
		return nil, err
	}

	if restoreFetched == nil {
		log.Infof("No Rancher restore with name '%s'' was detected", restoreName)
	}

	var restore RancherRestoreModel
	bdata, err := json.Marshal(restoreFetched)
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

func GetMySQLBackup(namespace, backupName string, log *zap.SugaredLogger) (*MySQLBackupModel, error) {
	backupFetched, err := getUnstructuredData("mysql.oracle.com", "v2", "mysqlbackups", backupName, namespace, "MySQL", log)
	if err != nil {
		log.Errorf("Unable to fetch MySQL backup '%s' due to '%v'", backupName, zap.Error(err))
		return nil, err
	}

	if backupFetched == nil {
		log.Infof("No MySQL backup with name '%s' in namespace '%s' was detected", backupName, namespace)
	}

	var backup MySQLBackupModel
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

func GetMySQLInnoDBStatus(namespace, backupName string, log *zap.SugaredLogger) (*InnoDBIcsModel, error) {
	innodbFetched, err := getUnstructuredData("mysql.oracle.com", "v2", "innodbclusters", backupName, namespace, "MySQL", log)
	if err != nil {
		log.Errorf("Unable to fetch MySQL backup '%s' due to '%v'", backupName, zap.Error(err))
		return nil, err
	}

	if innodbFetched == nil {
		log.Infof("No MySQL backup with name '%s' in namespace '%s' was detected", backupName, namespace)
	}

	var ics InnoDBIcsModel
	bdata, err := json.Marshal(innodbFetched)
	if err != nil {
		log.Errorf("Json marshalling error %v", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(bdata, &ics)
	if err != nil {
		log.Errorf("Json unmarshall error %v", zap.Error(err))
		return nil, err
	}

	//Debug
	var cmd BashCommand
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "ics")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, "keycloak")
	cmdArgs = append(cmdArgs, backupName)
	cmd.CommandArgs = cmdArgs

	response := Runner(&cmd, log)
	log.Infof("Debug Cmd Output =  '%v'", response.StandardOut.String())

	return &ics, nil
}

// TrackOperationProgress used to track operation status for a given gvr
func TrackOperationProgress(operator, operation, objectName, namespace string, log *zap.SugaredLogger) error {
	var response string
	switch operator {
	case "mysql":
		switch operation {
		case "backups":
			backupInfo, err := GetMySQLBackup(namespace, objectName, log)
			if err != nil {
				log.Errorf("Unable to fetch backup '%s' due to '%v'", objectName, zap.Error(err))
			}
			if backupInfo == nil {
				response = "Nil"
			} else {
				response = backupInfo.Status.Status
			}

		case "restores":
			restoreInfo, err := GetMySQLInnoDBStatus(namespace, objectName, log)
			if err != nil {
				log.Errorf("Unable to fetch restore '%s' due to '%v'", objectName, zap.Error(err))
			}
			if restoreInfo == nil {
				response = "Nil"
			} else {
				response = restoreInfo.Status.Cluster.Status
			}
		default:
			log.Errorf("Invalid operation specified for Velero = %s'", operation)
			response = "NAN"
		}

	case "velero":
		switch operation {
		case "backups":
			backupInfo, err := GetVeleroBackup(namespace, objectName, log)
			if err != nil {
				log.Errorf("Unable to fetch backup '%s' due to '%v'", objectName, zap.Error(err))
			}
			if backupInfo == nil {
				response = "Nil"
			} else {
				response = backupInfo.Status.Phase
			}

		case "restores":
			restoreInfo, err := GetVeleroRestore(namespace, objectName, log)
			if err != nil {
				log.Errorf("Unable to fetch restore '%s' due to '%v'", objectName, zap.Error(err))
			}
			if restoreInfo == nil {
				response = "Nil"
			} else {
				response = restoreInfo.Status.Phase
			}
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
			if backupInfo == nil {
				response = "Nil"
			} else {
				if backupInfo.Status.Conditions != nil {
					for _, cond := range backupInfo.Status.Conditions {
						if cond.Type == "Ready" {
							response = cond.Message
						} else {
							log.Infof("Rancher backup status : Type = %v, Status = %v", cond.Type, cond.Status)
						}
					}
				}
			}

		case "restores":
			restoreInfo, err := GetRancherRestore(objectName, log)
			if err != nil {
				log.Errorf("Unable to fetch restore '%s' due to '%v'", objectName, zap.Error(err))
			}
			if restoreInfo == nil {
				response = "Nil"
			} else {
				if restoreInfo.Status.Conditions != nil {
					for _, cond := range restoreInfo.Status.Conditions {
						if cond.Type == "Ready" {
							response = cond.Message
						} else {
							log.Infof("Rancher restore status : Type = %v, Status = %v", cond.Type, cond.Status)
						}
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
	case "InProgress", "", "PENDING", "INITIALIZING":
		log.Infof("%s '%s' is in progress. Status = %s", strings.ToTitle(operation), objectName, response)
		return fmt.Errorf("%s '%s' is in progress", strings.ToTitle(operation), objectName)
	case "Completed", "ONLINE":
		log.Infof("%s '%s' completed successfully", strings.ToTitle(operation), objectName)
		return nil
	default:
		return fmt.Errorf("%s failed. State = '%s'", strings.ToTitle(operation), response)
	}
}

// CrdPruner is a gvr based pruner
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

	if strings.Contains(group, "velero") || strings.Contains(group, "mysql") {
		err = dclient.Resource(gvr).Namespace(nameSpaceName).Delete(context.TODO(), resourceName, metav1.DeleteOptions{})
		if err != nil {
			if !k8serror.IsNotFound(err) {
				log.Errorf("Unable to delete resource '%s' from namespace '%s' due to '%v'", resourceName, nameSpaceName, zap.Error(err))
				return err
			}
		}
	}

	if strings.Contains(group, "cattle") {
		err = dclient.Resource(gvr).Delete(context.TODO(), resourceName, metav1.DeleteOptions{})
		if err != nil {
			if !k8serror.IsNotFound(err) {
				log.Errorf("Unable to delete resource '%s' due to '%v'", resourceName, zap.Error(err))
				return err
			}
		}
	}
	return nil
}
