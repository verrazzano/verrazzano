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
	"net/http"
	"strconv"
	"time"
)

//EnsureOpenSearchIsReachable is used determine whether opensearch cluster is reachable
func (o *OpensearchImpl) EnsureOpenSearchIsReachable(url string, log *zap.SugaredLogger) bool {
	response, err := http.Get(url) //#nosec G204
	if err != nil {
		return false
	}
	if response.StatusCode == 200 {
		log.Infof("OpenSearch is reachable.")
		return true
	}
	log.Infof("Unexpected response from opensearch at %v", response.StatusCode)
	return false
}

//EnsureOpenSearchIsHealthy ensures opensearch cluster is healthy
// Verifies if cluster is reachable
// Verifies if health url is reachable
// Verifies health status is green
func (o *OpensearchImpl) EnsureOpenSearchIsHealthy(url string, log *zap.SugaredLogger) bool {

	done := false
	retryCount := 0

	for !done {
		if o.EnsureOpenSearchIsReachable(url, log) {
			done = true
		} else {
			if retryCount <= constants.SnapshotRetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Cluster is not reachable. Retry after '%v' seconds", duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
			} else {
				log.Errorf("Cluster not reachable after '%v' retries", retryCount)
				return false
			}
		}
	}

	healthURL := fmt.Sprintf("%s/_cluster/health", url)
	healthReachable := false
	retryCount = 0

	for !healthReachable {
		response, err := http.Get(healthURL) //#nosec G204
		if err != nil {
			log.Error("HTTP GET failure ", zap.Error(err))
			return false
		}
		if response.StatusCode != 200 {
			if retryCount <= constants.SnapshotRetryCount {
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
	var clusterHealth types.OpenSearchHealthResponse
	for !healthGreen {
		bdata, err := utils.HTTPHelper("GET", healthURL, nil, log)
		if err != nil {
			return false
		}

		err = json.Unmarshal(bdata, &clusterHealth)
		if err != nil {
			if retryCount <= constants.SnapshotRetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Json unmarshalling error. Retry after '%v' seconds", duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
				continue
			} else {
				log.Errorf("Json unmarshalling error while checking cluster health %v. Retry count exceeded", err)
				return false
			}
		}

		if clusterHealth.Status != "green" {
			if retryCount <= constants.SnapshotRetryCount {
				duration := utils.GenerateRandom()
				log.Infof("Cluster health is '%s'. Retry after '%v' seconds", clusterHealth.Status, duration)
				time.Sleep(time.Second * time.Duration(duration))
				retryCount = retryCount + 1
			}
		} else {
			healthGreen = true
		}
	}

	if done && healthReachable && healthGreen {
		log.Infof("Cluster is reachable and healthy with status as '%s'", clusterHealth.Status)
		return true
	}

	return false
}

//UpdateKeystore Update Opensearch keystore with object store creds
func (o *OpensearchImpl) UpdateKeystore(client kubernetes.Interface, cfg *rest.Config, connData *types.ConnectionData, log *zap.SugaredLogger) (bool, error) {

	var accessKeyCmd, secretKeyCmd []string
	accessKeyCmd = append(accessKeyCmd, "/bin/sh", "-c", fmt.Sprintf("echo %s | %s", strconv.Quote(connData.Secret.ObjectAccessKey), constants.OpensearchKeystoreAccessKeyCmd))
	secretKeyCmd = append(secretKeyCmd, "/bin/sh", "-c", fmt.Sprintf("echo %s | %s", strconv.Quote(connData.Secret.ObjectSecretKey), constants.OpensearchkeystoreSecretAccessKeyCmd))

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
	url := fmt.Sprintf("%s/_nodes/reload_secure_settings", constants.EsUrl)
	nullBody := make(map[string]interface{})
	postBody, err := json.Marshal(nullBody)
	if err != nil {
		return err
	}
	response, err := http.Post(url, constants.HTTPContentType, bytes.NewBuffer(postBody))
	if err != nil {
		return err
	}
	if response.StatusCode == 200 {
		log.Infof("Secure settings reloaded")
		return nil
	}

	return fmt.Errorf("Error during reloading secure settings")

}

//RegisterSnapshotRepository Register an opbject store with Opensearch using the s3-plugin
func (o *OpensearchImpl) RegisterSnapshotRepository(secretData *types.ConnectionData, log *zap.SugaredLogger) error {
	log.Infof("Registering s3 backend repository '%s'", constants.OpeSearchSnapShotRepoName)
	var snapshotPayload types.OpenSearchSnapshotRequestPayload
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

	url := fmt.Sprintf("%s/_snapshot/%s", constants.EsUrl, constants.OpeSearchSnapShotRepoName)
	urlinfo := fmt.Sprintf("_snapshot/%s", constants.OpeSearchSnapShotRepoName)
	log.Infof("POST on registry url => '%s'", urlinfo)

	bdata, err := utils.HTTPHelper("POST", url, bytes.NewBuffer(postBody), log)
	if err != nil {
		return err
	}
	var registerResponse types.OpenSearchOperationResponse
	err = json.Unmarshal(bdata, &registerResponse)
	if err != nil {
		log.Errorf("json unmarshalling error %v", err)
		return err
	}
	//&& response.StatusCode == 200
	if registerResponse.Acknowledged {
		log.Infof("Snapshot registered successfully !")
		return nil
	}
	return fmt.Errorf("Snapshot registration unsuccessful. Response = %v", string(bdata))
}

//TriggerSnapshot this triggers a snapshot/backup of all the data streams/indices
func (o *OpensearchImpl) TriggerSnapshot(backupName string, log *zap.SugaredLogger) error {
	log.Infof("Triggering snapshot with name '%s'", backupName)
	snapShotURL := fmt.Sprintf("%s/_snapshot/%s/%s", constants.EsUrl, constants.OpeSearchSnapShotRepoName, backupName)
	nullBody := make(map[string]interface{})
	postBody, err := json.Marshal(nullBody)
	if err != nil {
		return err
	}
	bdata, err := utils.HTTPHelper("POST", snapShotURL, bytes.NewBuffer(postBody), log)
	if err != nil {
		return err
	}

	var snapshotResponse types.OpenSearchSnapshotResponse
	err = json.Unmarshal(bdata, &snapshotResponse)
	if err != nil {
		log.Errorf("json unmarshalling error %v", err)
		return err
	}
	if !snapshotResponse.Accepted {
		return fmt.Errorf("Snapshot registration failure. Response = %v ", string(bdata))
	}
	log.Infof("Snapshot registered successfully !")
	return nil
}

//CheckSnapshotProgress checks the data backup progress
func (o *OpensearchImpl) CheckSnapshotProgress(backupName string, log *zap.SugaredLogger) error {
	log.Infof("Checking snapshot progress with name '%s'", backupName)
	snapShotURL := fmt.Sprintf("%s/_snapshot/%s/%s", constants.EsUrl, constants.OpeSearchSnapShotRepoName, backupName)
	urlInfo := fmt.Sprintf("_snapshot/%s/%s", constants.OpeSearchSnapShotRepoName, backupName)
	log.Infof("GET on snapshot progress => '%s'", urlInfo)
	var snapshotInfo types.OpenSearchSnapshotStatus
	done := false
	retryCount := 0
	for !done {
		bdata, err := utils.HTTPHelper("GET", snapShotURL, nil, log)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bdata, &snapshotInfo)
		if err != nil {
			log.Errorf("json unmarshalling error %v", err)
			return err
		}
		switch snapshotInfo.Snapshots[0].State {
		case constants.OpenSearchSnapShotInProgress:
			if retryCount <= constants.SnapshotRetryCount {
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

//DeleteDataStreams used to delete data streams before restore.
// This requires that ingest be turned off
func (o *OpensearchImpl) DeleteDataStreams(log *zap.SugaredLogger) error {
	log.Infof("Delete existing data streams ..")
	snapShotURL := fmt.Sprintf("%s/_data_stream/*", constants.EsUrl)
	nullBody := make(map[string]interface{})
	postBody, err := json.Marshal(nullBody)
	if err != nil {
		return err
	}

	bdata, err := utils.HTTPHelper("DELETE", snapShotURL, bytes.NewBuffer(postBody), log)
	if err != nil {
		return err
	}

	var deleteResponse types.OpenSearchOperationResponse
	err = json.Unmarshal(bdata, &deleteResponse)
	if err != nil {
		log.Errorf("json unmarshalling error %v", err)
		return err
	}

	if !deleteResponse.Acknowledged {
		return fmt.Errorf("Data streams deletion failure. Response = %v ", string(bdata))
	}
	log.Infof("Data streams deleted successfully !")
	return nil
}

//DeleteDataIndexes used to delete data indexes before restore.
// This requires that ingest be turned off
func (o *OpensearchImpl) DeleteDataIndexes(log *zap.SugaredLogger) error {
	log.Infof("Delete existing data indices ..")
	snapShotURL := fmt.Sprintf("%s/*", constants.EsUrl)
	nullBody := make(map[string]interface{})
	postBody, err := json.Marshal(nullBody)
	if err != nil {
		return err
	}

	bdata, err := utils.HTTPHelper("DELETE", snapShotURL, bytes.NewBuffer(postBody), log)
	if err != nil {
		return err
	}

	var deleteResponse types.OpenSearchOperationResponse
	err = json.Unmarshal(bdata, &deleteResponse)
	if err != nil {
		log.Errorf("json unmarshalling error %v", err)
		return err
	}
	if deleteResponse.Acknowledged != true {
		return fmt.Errorf("Data indices deletion failure. Response = %v ", string(bdata))
	}
	log.Infof("Data indices deleted successfully !")
	return nil
}

//TriggerRestore Triggers a restore from a specified snapshot
func (o *OpensearchImpl) TriggerRestore(backupName string, log *zap.SugaredLogger) error {
	log.Infof("Triggering restore with name '%s'", backupName)
	restoreURL := fmt.Sprintf("%s/_snapshot/%s/%s/_restore", constants.EsUrl, constants.OpeSearchSnapShotRepoName, backupName)
	nullBody := make(map[string]interface{})
	postBody, err := json.Marshal(nullBody)
	if err != nil {
		return err
	}

	bdata, err := utils.HTTPHelper("POST", restoreURL, bytes.NewBuffer(postBody), log)
	if err != nil {
		return err
	}

	var restoreResponse types.OpenSearchSnapshotResponse
	err = json.Unmarshal(bdata, &restoreResponse)
	if err != nil {
		log.Errorf("json unmarshalling error %v", err)
		return err
	}
	if !restoreResponse.Accepted {
		return fmt.Errorf("Snapshot registration failure. Response = %v ", string(bdata))
	}
	log.Infof("Snapshot registered successfully !")
	return nil
}

//CheckRestoreProgress checks progress of restore process, by monitoring all the data streams
func (o *OpensearchImpl) CheckRestoreProgress(backupName string, log *zap.SugaredLogger) error {
	log.Infof("Checking restore progress with name '%s'", backupName)
	dsURL := fmt.Sprintf("%s/_data_stream", constants.EsUrl)
	var snapshotInfo types.OpenSearchDataStreams
	done := false
	notGreen := false
	retryCount := 0
	for !done {

		bdata, err := utils.HTTPHelper("GET", dsURL, nil, log)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bdata, &snapshotInfo)
		if err != nil {
			log.Errorf("json unmarshalling error %v", err)
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
			if retryCount <= constants.SnapshotRetryCount {
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

	err = o.DeleteDataStreams(log)
	if err != nil {
		return err
	}

	err = o.DeleteDataIndexes(log)
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
