// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/utils"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strconv"
	"time"
)

//EnsureOpenSearchIsReachable is used determine whether opensearch cluster is reachable
func (o *OpensearchImpl) EnsureOpenSearchIsReachable(url string, log *zap.SugaredLogger) error {
	log.Infof("Checking if cluster is reachable")
	var osinfo types.OpenSearchClusterInfo
	done := false
	retryCount := 0

	for !done {
		err := utils.HTTPHelper("GET", url, nil, &osinfo, log)
		if err != nil {
			if retryCount <= constants.RetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Cluster is not reachable. Retry after '%v' seconds", duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
			} else {
				log.Errorf("Cluster not reachable after '%v' retries", retryCount)
				return err
			}
		} else {
			done = true
		}
	}

	log.Infof("Cluster '%s' is reachable", osinfo.ClusterName)

	return nil
}

//EnsureOpenSearchIsHealthy ensures opensearch cluster is healthy
// Verifies if cluster is reachable
// Verifies if health url is reachable
// Verifies health status is green
func (o *OpensearchImpl) EnsureOpenSearchIsHealthy(url string, log *zap.SugaredLogger) error {
	log.Infof("Checking if cluster is healthy")
	var clusterHealth types.OpenSearchHealthResponse
	err := o.EnsureOpenSearchIsReachable(url, log)
	if err != nil {
		return err
	}

	healthURL := fmt.Sprintf("%s/_cluster/health", url)
	healthReachable := false
	retryCount := 0

	for !healthReachable {
		err = utils.HTTPHelper("GET", healthURL, nil, &clusterHealth, log)
		if err != nil {
			if retryCount <= constants.RetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Cluster health endpoint is not reachable. Retry after '%v' seconds", duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
			}
		} else {
			log.Errorf("Cluster health endpoint is reachable now")
			healthReachable = true
		}
	}

	healthGreen := false
	retryCount = 0

	for !healthGreen {
		err = utils.HTTPHelper("GET", healthURL, nil, &clusterHealth, log)
		if err != nil {
			if retryCount <= constants.RetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Json unmarshalling error. Retry after '%v' seconds", duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
				continue
			} else {
				log.Errorf("Json unmarshalling error while checking cluster health %v. Retry count exceeded", err)
				return err
			}
		}
		if clusterHealth.Status != "green" {
			if retryCount <= constants.RetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Cluster health is '%s'. Retry after '%v' seconds", clusterHealth.Status, duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
			}
		} else {
			healthGreen = true
		}
	}

	if healthReachable && healthGreen {
		log.Infof("Cluster is reachable and healthy with status as '%s'", clusterHealth.Status)
		return nil
	}

	return err
}

//UpdateKeystore Update Opensearch keystore with object store creds
func (o *OpensearchImpl) UpdateKeystore(client kubernetes.Interface, cfg *rest.Config, connData *types.ConnectionData, log *zap.SugaredLogger) (bool, error) {

	var accessKeyCmd, secretKeyCmd []string
	accessKeyCmd = append(accessKeyCmd, "/bin/sh", "-c", fmt.Sprintf("echo %s | %s", strconv.Quote(connData.Secret.ObjectAccessKey), constants.OpensearchKeystoreAccessKeyCmd))
	secretKeyCmd = append(secretKeyCmd, "/bin/sh", "-c", fmt.Sprintf("echo %s | %s", strconv.Quote(connData.Secret.ObjectSecretKey), constants.OpensearchKeystoreSecretAccessKeyCmd))

	// Updating keystore in other masters
	for i := 0; i < 3; i++ {

		podName := fmt.Sprintf("vmi-system-es-master-%v", i)
		log.Infof("Updating keystore in pod '%s'", podName)
		pod, err := client.CoreV1().Pods(constants.VerrazzanoNameSpaceName).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		_, _, err = k8sutil.ExecPod(client, cfg, pod, constants.OpenSearchMasterPodContainerName, accessKeyCmd)
		if err != nil {
			log.Errorf("Unable to exec into pod %s due to %v", pod.Name, err)
			return false, err
		}
		_, _, err = k8sutil.ExecPod(client, cfg, pod, constants.OpenSearchMasterPodContainerName, secretKeyCmd)
		if err != nil {
			log.Errorf("Unable to exec into pod %s due to %v", pod.Name, err)
			return false, err
		}

	}

	for i := 0; i < 3; i++ {
		labelkv := fmt.Sprintf("app=%s,index=%v", constants.OpenSearchDataPodPrefix, i)
		listOptions := metav1.ListOptions{LabelSelector: labelkv}
		podItems, err := client.CoreV1().Pods(constants.VerrazzanoNameSpaceName).List(context.TODO(), listOptions)
		if err != nil {
			return false, err
		}

		log.Infof("Updating keystore in pod '%s'", podItems.Items[0].GetName())
		_, _, err = k8sutil.ExecPod(client, cfg, &podItems.Items[0], constants.OpenSearchDataPodContainerName, accessKeyCmd)
		if err != nil {
			log.Errorf("Unable to exec into pod %s due to %v", podItems.Items[0].GetName(), err)
			return false, err
		}
		_, _, err = k8sutil.ExecPod(client, cfg, &podItems.Items[0], constants.OpenSearchDataPodContainerName, secretKeyCmd)
		if err != nil {
			log.Errorf("Unable to exec into pod %s due to %v", podItems.Items[0].GetName(), err)
			return false, err
		}

	}
	return true, nil

}

//ReloadOpensearchSecureSettings used to reload secure settings once object store keys are updated
func (o *OpensearchImpl) ReloadOpensearchSecureSettings(log *zap.SugaredLogger) error {
	var secureSettings types.OpenSearchSecureSettingsReloadStatus
	url := fmt.Sprintf("%s/_nodes/reload_secure_settings", constants.OpenSearchURL)
	err := utils.HTTPHelper("POST", url, nil, &secureSettings, log)
	if err != nil {
		return err
	}
	if secureSettings.ClusterNodes.Failed == 0 && secureSettings.ClusterNodes.Total == secureSettings.ClusterNodes.Successful {
		log.Infof("Secure settings reloaded sucessfully across all '%v' nodes of the cluster", secureSettings.ClusterNodes.Total)
		return nil
	}
	return fmt.Errorf("Not all nodes were updated successfully. Total = '%v', Failed = '%v' , Successful = '%v'", secureSettings.ClusterNodes.Total, secureSettings.ClusterNodes.Failed, secureSettings.ClusterNodes.Successful)
}

//RegisterSnapshotRepository Register an opbject store with Opensearch using the s3-plugin
func (o *OpensearchImpl) RegisterSnapshotRepository(secretData *types.ConnectionData, log *zap.SugaredLogger) error {
	log.Infof("Registering s3 backend repository '%s'", constants.OpeSearchSnapShotRepoName)
	var snapshotPayload types.OpenSearchSnapshotRequestPayload
	var registerResponse types.OpenSearchOperationResponse
	snapshotPayload.Type = "s3"
	snapshotPayload.Settings.Bucket = secretData.BucketName
	snapshotPayload.Settings.Region = secretData.RegionName
	snapshotPayload.Settings.Client = "default"
	snapshotPayload.Settings.Endpoint = secretData.Endpoint
	snapshotPayload.Settings.PathStyleAccess = true

	postBody, err := json.Marshal(snapshotPayload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/_snapshot/%s", constants.OpenSearchURL, constants.OpeSearchSnapShotRepoName)
	err = utils.HTTPHelper("POST", url, bytes.NewBuffer(postBody), &registerResponse, log)
	if err != nil {
		return err
	}

	if registerResponse.Acknowledged {
		log.Infof("Snapshot registered successfully !")
		return nil
	}
	return fmt.Errorf("Snapshot registration unsuccessful. Response = %v", registerResponse)
}

//TriggerSnapshot this triggers a snapshot/backup of all the data streams/indices
func (o *OpensearchImpl) TriggerSnapshot(backupName string, log *zap.SugaredLogger) error {
	log.Infof("Triggering snapshot with name '%s'", backupName)
	var snapshotResponse types.OpenSearchSnapshotResponse
	snapShotURL := fmt.Sprintf("%s/_snapshot/%s/%s", constants.OpenSearchURL, constants.OpeSearchSnapShotRepoName, backupName)
	err := utils.HTTPHelper("POST", snapShotURL, nil, &snapshotResponse, log)
	if err != nil {
		return err
	}

	if !snapshotResponse.Accepted {
		return fmt.Errorf("Snapshot registration failure. Response = %v ", snapshotResponse)
	}
	log.Infof("Snapshot registered successfully !")
	return nil
}

//CheckSnapshotProgress checks the data backup progress
func (o *OpensearchImpl) CheckSnapshotProgress(backupName string, log *zap.SugaredLogger) error {
	log.Infof("Checking snapshot progress with name '%s'", backupName)
	snapShotURL := fmt.Sprintf("%s/_snapshot/%s/%s", constants.OpenSearchURL, constants.OpeSearchSnapShotRepoName, backupName)
	var snapshotInfo types.OpenSearchSnapshotStatus

	done := false
	retryCount := 0
	for !done {
		err := utils.HTTPHelper("GET", snapShotURL, nil, &snapshotInfo, log)
		if err != nil {
			return err
		}
		switch snapshotInfo.Snapshots[0].State {
		case constants.OpenSearchSnapShotInProgress:
			if retryCount <= constants.RetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Snapshot '%s' is in progress. Check again after '%v' seconds", backupName, duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
			} else {
				return fmt.Errorf("Retry count exceeded. Snapshot '%s' state is still IN_PROGRESS", backupName)
			}
		case constants.OpenSearchSnapShotSucess:
			log.Infof("Snapshot '%s' complete", backupName)
			done = true
		default:
			return fmt.Errorf("Snapshot '%s' state is invalid '%s'", backupName, snapshotInfo.Snapshots[0].State)
		}
	}

	log.Infof("Number of shards backed up = %v", snapshotInfo.Snapshots[0].Shards.Total)
	log.Infof("Number of successfull shards backed up = %v", snapshotInfo.Snapshots[0].Shards.Total)
	log.Infof("Indices backed up = %v", snapshotInfo.Snapshots[0].Indices)
	log.Infof("Data streams backed up = %v", snapshotInfo.Snapshots[0].DataStreams)

	return nil
}

//DeleteData used to delete data streams before restore.
// This requires that ingest be turned off
func (o *OpensearchImpl) DeleteData(log *zap.SugaredLogger) error {
	log.Infof("Deleting data streams followed by index ..")
	dataStreamURL := fmt.Sprintf("%s/_data_stream/*", constants.OpenSearchURL)
	dataIndexURL := fmt.Sprintf("%s/*", constants.OpenSearchURL)
	var deleteResponse types.OpenSearchOperationResponse

	err := utils.HTTPHelper("DELETE", dataStreamURL, nil, &deleteResponse, log)
	if err != nil {
		return err
	}

	if !deleteResponse.Acknowledged {
		return fmt.Errorf("Data streams deletion failure. Response = %v ", deleteResponse)
	}

	err = utils.HTTPHelper("DELETE", dataIndexURL, nil, &deleteResponse, log)
	if err != nil {
		return err
	}

	if !deleteResponse.Acknowledged {
		return fmt.Errorf("Data index deletion failure. Response = %v ", deleteResponse)
	}

	log.Infof("Data streams and data indexes deleted successfully !")
	return nil
}

//TriggerRestore Triggers a restore from a specified snapshot
func (o *OpensearchImpl) TriggerRestore(backupName string, log *zap.SugaredLogger) error {
	log.Infof("Triggering restore with name '%s'", backupName)
	restoreURL := fmt.Sprintf("%s/_snapshot/%s/%s/_restore", constants.OpenSearchURL, constants.OpeSearchSnapShotRepoName, backupName)
	var restoreResponse types.OpenSearchSnapshotResponse

	err := utils.HTTPHelper("POST", restoreURL, nil, &restoreResponse, log)
	if err != nil {
		return err
	}

	if !restoreResponse.Accepted {
		return fmt.Errorf("Snapshot registration failure. Response = %v ", restoreResponse)
	}
	log.Infof("Snapshot registered successfully !")
	return nil
}

//CheckRestoreProgress checks progress of restore process, by monitoring all the data streams
func (o *OpensearchImpl) CheckRestoreProgress(backupName string, log *zap.SugaredLogger) error {
	log.Infof("Checking restore progress with name '%s'", backupName)
	dsURL := fmt.Sprintf("%s/_data_stream", constants.OpenSearchURL)
	var snapshotInfo types.OpenSearchDataStreams
	done := false
	notGreen := false
	retryCount := 0
	for !done {
		err := utils.HTTPHelper("GET", dsURL, nil, &snapshotInfo, log)
		if err != nil {
			return err
		}
		for _, ds := range snapshotInfo.DataStreams {
			switch ds.Status {
			case constants.DataStreamGreen:
				log.Infof("Data stream '%s' restore complete", ds.Name)
			default:
				notGreen = true
			}
		}

		if notGreen {
			if retryCount <= constants.RetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Restore is in progress. Check again after '%v' seconds", duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
				notGreen = false
			} else {
				return fmt.Errorf("Retry count exceeded. Snapshot '%s' state is still IN_PROGRESS", backupName)
			}
		} else {
			// This section is hit when all data streams are green
			// exit feedback loop
			done = true
		}

	}

	log.Infof("All streams are healthy")
	return nil
}

//Backup - Toplevel method to invoke Opensearch backup
func (o *OpensearchImpl) Backup(secretData *types.ConnectionData, backupName string, log *zap.SugaredLogger) error {
	log.Info("Start backup steps ....")
	err := o.RegisterSnapshotRepository(secretData, log)
	if err != nil {
		return err
	}

	err = o.TriggerSnapshot(backupName, log)
	if err != nil {
		return err
	}

	err = o.CheckSnapshotProgress(backupName, log)
	if err != nil {
		return err
	}

	log.Infof("Opensearch snapshot taken successfully. ")
	return nil
}

//Restore - Top level method to invoke opensearch restore
func (o *OpensearchImpl) Restore(secretData *types.ConnectionData, backupName string, log *zap.SugaredLogger) error {
	log.Info("Start restore steps ....")
	err := o.RegisterSnapshotRepository(secretData, log)
	if err != nil {
		return err
	}

	err = o.DeleteData(log)
	if err != nil {
		return err
	}

	err = o.TriggerRestore(backupName, log)
	if err != nil {
		return err
	}

	err = o.CheckRestoreProgress(backupName, log)
	if err != nil {
		return err
	}

	log.Infof("Opensearch snapshot taken successfully. ")
	return nil
}
