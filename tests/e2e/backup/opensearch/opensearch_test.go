// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortWaitTimeout                    = 10 * time.Minute
	shortPollingInterval                = 10 * time.Second
	waitTimeout                         = 15 * time.Minute
	pollingInterval                     = 30 * time.Second
	vmoDeploymentName                   = "verrazzano-monitoring-operator"
	osStsName                           = "vmi-system-es-master"
	osStsPvcPrefix                      = "elasticsearch-master-vmi-system-es-master"
	osDataDepPrefix                     = "vmi-system-es-data"
	osIngestDeployment                  = "vmi-system-es-ingest"
	osDepPvcPrefix                      = "vmi-system-es-data"
	objectStoreCredsAccessKeyName       = "aws_access_key_id"     //nolint:gosec //#gosec G101 //#gosec G204
	objectStoreCredsSecretAccessKeyName = "aws_secret_access_key" //nolint:gosec //#gosec G101 //#gosec G204
)

var (
	VeleroNameSpace, VeleroSecretName                                                                    string
	RancherSecretName                                                                                    string
	OciBucketID, OciBucketName, OciOsAccessKey, OciOsAccessSecretKey, OciCompartmentID, OciNamespaceName string
	BackupName, RestoreName, BackupResourceName, BackupOpensearchName, BackupRancherName                 string
	RestoreOpensearchName, RestoreRancherName                                                            string
	BackupRegion, BackupStorageName                                                                      string
	BackupID                                                                                             string
)

func gatherInfo() {
	VeleroNameSpace = os.Getenv("VELERO_NAMESPACE")
	VeleroSecretName = os.Getenv("VELERO_SECRET_NAME")
	RancherSecretName = os.Getenv("RANCHER_SECRET_NAME")
	OciBucketID = os.Getenv("OCI_OS_BUCKET_ID")
	OciBucketName = os.Getenv("OCI_OS_BUCKET_NAME")
	OciOsAccessKey = os.Getenv("OCI_OS_ACCESS_KEY")
	OciOsAccessSecretKey = os.Getenv("OCI_OS_ACCESS_SECRET_KEY")
	OciCompartmentID = os.Getenv("OCI_OS_COMPARTMENT_ID")
	OciNamespaceName = os.Getenv("OCI_OS_NAMESPACE")
	BackupName = os.Getenv("BACKUP_NAME")
	RestoreName = os.Getenv("RESTORE_NAME")
	BackupResourceName = os.Getenv("BACKUP_RESOURCE")
	BackupOpensearchName = os.Getenv("BACKUP_OPENSEARCH")
	BackupRancherName = os.Getenv("BACKUP_RANCHER")
	RestoreOpensearchName = os.Getenv("RESTORE_OPENSEARCH")
	RestoreRancherName = os.Getenv("RESTORE_RANCHER")
	BackupStorageName = os.Getenv("BACKUP_STORAGE")
	BackupRegion = os.Getenv("BACKUP_REGION")
}

const secretsData = //nolint:gosec //#gosec G101 //#gosec G204
`[default]
{{ .AccessName }}={{ .ObjectStoreAccessValue }}
{{ .ScrtName }}={{ .ObjectStoreScrt }}
`

const veleroBackupLocation = `
    apiVersion: velero.io/v1
    kind: BackupStorageLocation
    metadata:
      name: {{ .VeleroBackupStorageName }}
      namespace: {{ .VeleroNamespaceName }}
    spec:
      provider: aws
      objectStorage:
        bucket: {{ .VeleroObjectStoreBucketName }}
        prefix: opensearch
      credential:
        name: {{ .VeleroSecretName }}
        key: cloud
      config:
        region: {{ .VeleroBackupRegion }}
        s3ForcePathStyle: "true"
        s3Url: https://{{ .VeleroObjectStorageNamespaceName }}.compat.objectstorage.{{ .VeleroBackupRegion }}.oraclecloud.com`

const veleroBackup = `
---
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: {{ .VeleroBackupName }}
  namespace: {{ .VeleroNamespaceName }}
spec:
  includedNamespaces:
    - verrazzano-system
  labelSelector:
    matchLabels:
      verrazzano-component: opensearch
  defaultVolumesToRestic: false
  storageLocation: {{ .VeleroBackupStorageName }}
  hooks:
    resources:
      - 
        name: {{ .VeleroOpensearchHookResourceName }}
        includedNamespaces:
          - verrazzano-system
        labelSelector:
          matchLabels:
            statefulset.kubernetes.io/pod-name: vmi-system-es-master-0
        post:
          - 
            exec:
              container: es-master
              command:
                - /usr/share/opensearch/bin/verrazzano-backup-hook
                - -operation
                - backup
                - -velero-backup-name
                - {{ .VeleroBackupName }}
              onError: Fail
              timeout: 10m`

const veleroRestore = `
---
apiVersion: velero.io/v1
kind: Restore
metadata:
  name: {{ .VeleroRestore }}
  namespace: {{ .VeleroNamespaceName }}
spec:
  backupName: {{ .VeleroBackupName }}
  includedNamespaces:
    - verrazzano-system
  labelSelector:
    matchLabels:
      verrazzano-component: opensearch
  restorePVs: false
  hooks:
    resources:
      - name: {{ .VeleroOpensearchHookResourceName }}
        includedNamespaces:
          - verrazzano-system
        labelSelector:
          matchLabels:
            statefulset.kubernetes.io/pod-name: vmi-system-es-master-0
        postHooks:
          - exec:
              container: es-master
              command:
                - /usr/share/opensearch/bin/verrazzano-backup-hook
                - -operation
                - restore
                - -velero-backup-name
                - {{ .VeleroBackupName }}
              waitTimeout: 30m
              execTimeout: 30m
              onError: Fail`

const esQueryBody = `
{
	"query": {
  		"terms": {
			"_id": ["{{ .BackupIDBeforeBackup }}"]
  		}
	}
}
`

type accessData struct {
	AccessName             string
	ScrtName               string
	ObjectStoreAccessValue string
	ObjectStoreScrt        string
}

type veleroBackupLocationObjectData struct {
	VeleroBackupStorageName          string
	VeleroNamespaceName              string
	VeleroObjectStoreBucketName      string
	VeleroSecretName                 string
	VeleroObjectStorageNamespaceName string
	VeleroBackupRegion               string
}

type veleroBackupObject struct {
	VeleroBackupName                 string
	VeleroNamespaceName              string
	VeleroBackupStorageName          string
	VeleroOpensearchHookResourceName string
}

type veleroRestoreObject struct {
	VeleroRestore                    string
	VeleroNamespaceName              string
	VeleroBackupName                 string
	VeleroOpensearchHookResourceName string
}

type esQueryObject struct {
	BackupIDBeforeBackup string
}

var _ = t.BeforeSuite(func() {
	start := time.Now()
	gatherInfo()
	backupPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var t = framework.NewTestFramework("opensearch-backup")

// CreateCredentialsSecretFromFile creates opaque secret from the given map of values
func CreateCredentialsSecretFromFile(namespace string, name string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	var b bytes.Buffer
	template, _ := template.New("testsecrets").Parse(secretsData)
	data := accessData{
		AccessName:             objectStoreCredsAccessKeyName,
		ScrtName:               objectStoreCredsSecretAccessKeyName,
		ObjectStoreAccessValue: OciOsAccessKey,
		ObjectStoreScrt:        OciOsAccessSecretKey,
	}
	template.Execute(&b, data)

	secretData := make(map[string]string)
	secretData["cloud"] = b.String()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		t.Logs.Errorf("Error creating secret ", zap.Error(err))
		//return err
	}
	return nil
}

// CreateVeleroBackupLocationObject creates opaque secret from the given map of values
func CreateVeleroBackupLocationObject() error {
	var b bytes.Buffer
	template, _ := template.New("velero-backup-location").Parse(veleroBackupLocation)

	data := veleroBackupLocationObjectData{
		VeleroBackupStorageName:          BackupStorageName,
		VeleroNamespaceName:              VeleroNameSpace,
		VeleroObjectStoreBucketName:      OciBucketName,
		VeleroSecretName:                 VeleroSecretName,
		VeleroObjectStorageNamespaceName: OciNamespaceName,
		VeleroBackupRegion:               BackupRegion,
	}
	template.Execute(&b, data)
	err := pkg.CreateOrUpdateResourceFromBytes(b.Bytes())
	if err != nil {
		t.Logs.Errorf("Error creating velero backup loaction ", zap.Error(err))
		//return err
	}

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "backupstoragelocation.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, VeleroNameSpace)
	cmdArgs = append(cmdArgs, BackupStorageName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "custom-columns=:metadata.name")
	cmdArgs = append(cmdArgs, "--no-headers")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	cmdResponse := Runner(&kcmd, t.Logs)
	if cmdResponse.CommandError != nil {
		return cmdResponse.CommandError
	}
	storageNameRetrieved := strings.TrimSpace(strings.Trim(cmdResponse.StandardOut.String(), "\n"))
	if storageNameRetrieved == BackupStorageName {
		t.Logs.Errorf("backup storage location '%s' already created", BackupStorageName)
		//return fmt.Errorf("backup storage location '%s' already created", BackupStorageName)
	}
	return nil
}

// CreateVeleroBackupObject creates opaque secret from the given map of values
func CreateVeleroBackupObject() error {
	var b bytes.Buffer
	template, _ := template.New("velero-backup").Parse(veleroBackup)
	data := veleroBackupObject{
		VeleroNamespaceName:              VeleroNameSpace,
		VeleroBackupName:                 BackupOpensearchName,
		VeleroBackupStorageName:          BackupStorageName,
		VeleroOpensearchHookResourceName: BackupResourceName,
	}
	template.Execute(&b, data)
	err := pkg.CreateOrUpdateResourceFromBytes(b.Bytes())
	if err != nil {
		t.Logs.Errorf("Error creating velero backup ", zap.Error(err))
		//return err
	}

	// Wait for backup object to be created before checking.
	// It has been observed that backup object gets created after a delay
	time.Sleep(pollingInterval)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "backup.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, VeleroNameSpace)
	cmdArgs = append(cmdArgs, BackupOpensearchName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "custom-columns=:metadata.name")
	cmdArgs = append(cmdArgs, "--no-headers")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	cmdResponse := Runner(&kcmd, t.Logs)
	if cmdResponse.CommandError != nil {
		return cmdResponse.CommandError
	}

	retrievedBackupObject := strings.TrimSpace(strings.Trim(cmdResponse.StandardOut.String(), "\n"))
	if retrievedBackupObject == BackupOpensearchName {
		t.Logs.Errorf("backup '%s' already created", BackupOpensearchName)
		//return fmt.Errorf("backup '%s' already created", BackupOpensearchName)
	}

	if retrievedBackupObject == "" {
		return fmt.Errorf("backup '%s' was not created", BackupOpensearchName)
	}

	return nil
}

func CreateVeleroRestoreObject() error {
	var b bytes.Buffer
	template, _ := template.New("velero-restore").Parse(veleroRestore)
	data := veleroRestoreObject{
		VeleroRestore:                    RestoreName,
		VeleroNamespaceName:              VeleroNameSpace,
		VeleroBackupName:                 BackupOpensearchName,
		VeleroOpensearchHookResourceName: BackupResourceName,
	}

	template.Execute(&b, data)
	err := pkg.CreateOrUpdateResourceFromBytes(b.Bytes())
	if err != nil {
		t.Logs.Infof("Error creating velero restore ", zap.Error(err))
		//return err
	}

	time.Sleep(pollingInterval)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "restore.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, VeleroNameSpace)
	cmdArgs = append(cmdArgs, RestoreName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "custom-columns=:metadata.name")
	cmdArgs = append(cmdArgs, "--no-headers")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	cmdResponse := Runner(&kcmd, t.Logs)
	if cmdResponse.CommandError != nil {
		return cmdResponse.CommandError
	}

	retrievedRestoreObject := strings.TrimSpace(strings.Trim(cmdResponse.StandardOut.String(), "\n"))
	if retrievedRestoreObject == RestoreName {
		t.Logs.Errorf("restore '%s' already created", RestoreName)
		//return fmt.Errorf("restore '%s' already created", RestoreName)
	}

	return nil
}

func GetBackupID() error {
	esURL, err := GetEsURL(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting es url ", zap.Error(err))
		return err
	}

	vzPasswd, err := GetVZPasswd(t.Logs)
	if err != nil {
		t.Logs.Errorf("Error getting vz passwd ", zap.Error(err))
		//return err
	}
	var cmdArgs []string
	url := strconv.Quote(fmt.Sprintf("%s/verrazzano-system/_search?from=0&size=1", esURL))
	creds := fmt.Sprintf("verrazzano:%s", vzPasswd)
	jqIDFetch := "| jq -r '.hits.hits[0]._id'"
	curlCmd := fmt.Sprintf("curl -ks %s -u %s %s", url, creds, jqIDFetch)
	cmdArgs = append(cmdArgs, "/bin/sh")
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, curlCmd)

	var kcmd BashCommand
	kcmd.Timeout = 2 * time.Minute
	kcmd.CommandArgs = cmdArgs

	curlResponse := Runner(&kcmd, t.Logs)
	if curlResponse.CommandError != nil {
		return curlResponse.CommandError
	}
	BackupID = strings.TrimSpace(strings.Trim(curlResponse.StandardOut.String(), "\n"))
	t.Logs.Infof("BackupId ===> = '%s", BackupID)
	if BackupID != "" {
		t.Logs.Errorf("BackupId has already been retrieved = '%s", BackupID)
		//return fmt.Errorf("backupId has already been retrieved = '%s", BackupID)
	}
	return nil
}

func IsRestoreSuccessful() bool {
	esURL, err := GetEsURL(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting es url ", zap.Error(err))
		return false
	}

	vzPasswd, err := GetVZPasswd(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting vz passwd ", zap.Error(err))
		return false
	}
	var b bytes.Buffer
	template, _ := template.New("velero-restore-verify").Parse(esQueryBody)
	data := esQueryObject{
		BackupIDBeforeBackup: BackupID,
	}
	template.Execute(&b, data)

	var cmdArgs []string
	header := "Content-Type: application/json"
	url := strconv.Quote(fmt.Sprintf("%s/verrazzano-system/_search?", esURL))
	creds := fmt.Sprintf("verrazzano:%s", vzPasswd)
	jqIDFetch := "| jq -r '.hits.hits[0]._id'"
	curlCmd := fmt.Sprintf("curl -ks -H %s %s -u %s -d '%s' %s", strconv.Quote(header), url, creds, b.String(), jqIDFetch)
	cmdArgs = append(cmdArgs, "/bin/sh")
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, curlCmd)

	var kcmd BashCommand
	kcmd.Timeout = 2 * time.Minute
	kcmd.CommandArgs = cmdArgs

	curlResponse := Runner(&kcmd, t.Logs)
	if curlResponse.CommandError != nil {
		return false
	}
	backupIDFetched := strings.TrimSpace(strings.Trim(curlResponse.StandardOut.String(), "\n"))
	return backupIDFetched == BackupID
}

func CheckBackupProgress() error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "backup.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, VeleroNameSpace)
	cmdArgs = append(cmdArgs, BackupOpensearchName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "jsonpath={.status.phase}")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs

	bashResponse := Runner(&kcmd, t.Logs)
	if bashResponse.CommandError != nil {
		return bashResponse.CommandError
	}
	response := strings.TrimSpace(strings.Trim(bashResponse.StandardOut.String(), "\n"))
	switch response {
	case "InProgress":
		t.Logs.Infof("Backup '%s' is in progress", BackupOpensearchName)
	case "Completed":
		t.Logs.Infof("Backup '%s' completed successfully", BackupOpensearchName)
	}
	return nil

}

func CheckRestoreProgress() error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "restore.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, VeleroNameSpace)
	cmdArgs = append(cmdArgs, RestoreName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "jsonpath={.status.phase}")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs

	bashResponse := Runner(&kcmd, t.Logs)
	if bashResponse.CommandError != nil {
		return bashResponse.CommandError
	}
	response := strings.TrimSpace(strings.Trim(bashResponse.StandardOut.String(), "\n"))
	switch response {
	case "InProgress":
		t.Logs.Infof("Restore '%s' is in progress", BackupOpensearchName)
		//return nil
	case "Completed":
		t.Logs.Infof("Restore '%s' completed successfully.", BackupOpensearchName)
	}
	return nil
}

func NukeOpensearch() error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	t.Logs.Infof("Scaling down VMO")
	getScale, err := clientset.AppsV1().Deployments(constants.VerrazzanoSystemNamespace).GetScale(context.TODO(), vmoDeploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	newScale := *getScale
	newScale.Spec.Replicas = 0

	_, err = clientset.AppsV1().Deployments(constants.VerrazzanoSystemNamespace).UpdateScale(context.TODO(), vmoDeploymentName, &newScale, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	t.Logs.Infof("Deleting opensearch master sts")
	err = clientset.AppsV1().StatefulSets(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), osStsName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	t.Logs.Infof("Deleting opensearch data deployments")
	for i := 0; i < 3; i++ {
		err = clientset.AppsV1().Deployments(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), fmt.Sprintf("%s-%v", osDataDepPrefix, i), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	t.Logs.Infof("Deleting opensearch ingest deployment")
	err = clientset.AppsV1().Deployments(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), osIngestDeployment, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	t.Logs.Infof("Deleting opensearch master pvc if still present")
	for i := 0; i < 3; i++ {
		err = clientset.CoreV1().PersistentVolumeClaims(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), fmt.Sprintf("%s-%v", osStsPvcPrefix, i), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	t.Logs.Infof("Deleting opensearch data pvc")
	for i := 0; i < 3; i++ {
		if i == 0 {
			err = clientset.CoreV1().PersistentVolumeClaims(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), osDepPvcPrefix, metav1.DeleteOptions{})
		} else {
			err = clientset.CoreV1().PersistentVolumeClaims(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), fmt.Sprintf("%s-%v", osDepPvcPrefix, i), metav1.DeleteOptions{})
		}
		if err != nil {
			return err
		}
	}

	err = checkPodsTerminated("verrazzano-component=opensearch", constants.VerrazzanoSystemNamespace)
	if err != nil {
		return err
	}

	return checkPvcsTerminated("verrazzano-component=opensearch", constants.VerrazzanoSystemNamespace)
}

func checkPodsTerminated(labelSelector, namespace string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	/*
		retryCount := 0
		for {
			listOptions := metav1.ListOptions{LabelSelector: labelSelector}
			pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
			if err != nil {
				return err
			}
			if len(pods.Items) > 0 {
				if retryCount > 100 {
					return fmt.Errorf("retry count to monitor pods exceeded")
				}
				t.Logs.Infof("Pods with label selector '%s' in namespace '%s' are still present", labelSelector, namespace)
				time.Sleep(10 * time.Second)
			} else {
				return nil
			}
		}
	*/
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	if len(pods.Items) > 0 {
		return fmt.Errorf("Pods with label selector '%s' in namespace '%s' are still present", labelSelector, namespace)
	}
	return nil

}

func checkPvcsTerminated(labelSelector, namespace string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}
	/*
		retryCount := 0
		for {
			listOptions := metav1.ListOptions{LabelSelector: labelSelector}
			pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), listOptions)
			if err != nil {
				return err
			}
			if len(pvcs.Items) > 0 {
				if retryCount > 100 {
					return fmt.Errorf("retry count to monitor pvcs exceeded")
				}
				t.Logs.Infof("Pvcs with label selector '%s' in namespace '%s' are still present", labelSelector, namespace)
				time.Sleep(10 * time.Second)
			} else {
				return nil
			}
		}
	*/

	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	if len(pvcs.Items) > 0 {
		return fmt.Errorf("pvcs with label selector '%s' in namespace '%s' are still present", labelSelector, namespace)
	}
	return nil

}

// 'It' Wrapper to only run spec if the Velero is supported on the current Verrazzano version
func WhenVeleroInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.4.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Velero is not supported", description)
	}
}

func backupPrerequisites() {
	t.Logs.Info("Setup backup pre-requisites")
	t.Logs.Info("Create backup secret for velero backup objects")
	Eventually(func() error {
		return CreateCredentialsSecretFromFile(VeleroNameSpace, VeleroSecretName)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Create backup storage location for velero backup objects")
	Eventually(func() error {
		return CreateVeleroBackupLocationObject()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Get backup id before starting the backup process")
	Eventually(func() error {
		return GetBackupID()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

var _ = t.Describe("Backup Flow,", Label("f:platform-verrazzano.backup"), Serial, func() {
	t.Logs.Infof("Start backup")

	t.Context("after velero backup storage location created", func() {
		WhenVeleroInstalledIt("Start velero backup", func() {
			Eventually(func() error {
				return CreateVeleroBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})
	t.Logs.Infof("Check backup")
	t.Context("after velero backup was created", func() {
		WhenVeleroInstalledIt("Check velero backup progress", func() {
			Eventually(func() error {
				return CheckBackupProgress()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})
	t.Logs.Infof("Cleanup opensearch")

	t.Context("Cleanup opensearch once backup is done", func() {
		WhenVeleroInstalledIt("Nuke opensearch", func() {
			Eventually(func() error {
				return NukeOpensearch()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

	})
})

var _ = t.Describe("Start Restore,", Label("f:platform-verrazzano.restore"), Serial, func() {

	t.Logs.Infof("Start restore")
	t.Context("start restore after velero backup is completed", func() {
		WhenVeleroInstalledIt("Start velero restore", func() {
			Eventually(func() error {
				return CreateVeleroRestoreObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Logs.Infof("Check restore")
	t.Context("Create the velero restore object", func() {
		WhenVeleroInstalledIt("Check velero restore progress", func() {
			Eventually(func() error {
				return CheckRestoreProgress()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Logs.Infof("Verify restore")

	t.Context("verify restore after velero restore is completed", func() {
		WhenVeleroInstalledIt("Is Restore good?", func() {
			Eventually(func() bool {
				return IsRestoreSuccessful()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

	})

})
