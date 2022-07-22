// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 15 * time.Minute
	pollingInterval      = 30 * time.Second
	vmoDeploymentName    = "verrazzano-monitoring-operator"
	osStsName            = "vmi-system-es-master"
	osStsPvcPrefix       = "elasticsearch-master-vmi-system-es-master"
	osDataDepPrefix      = "vmi-system-es-data"
	osIngestDeployment   = "vmi-system-es-ingest"
	osDepPvcPrefix       = "vmi-system-es-data"
)

var (
	VeleroNameSpace, VeleroSecretName                                                                    string
	RancherSecretName                                                                                    string
	OciBucketID, OciBucketName, OciOsAccessKey, OciOsAccessSecretKey, OciCompartmentID, OciNamespaceName string
	BackupName, RestoreName, BackupResourceName, BackupOpensearchName, BackupRancherName                 string
	RestoreOpensearchName, RestoreRancherName                                                            string
	BackupStorageName                                                                                    string
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
}

const secretsData = `[default]
aws_access_key_id={{ .ObjectStoreAccessKeyID }}
aws_secret_access_key={{ .ObjectStoreAccessKey }}
` //nolint:gosec

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
        region: us-phoenix-1
        s3ForcePathStyle: "true"
        s3Url: https://{{ .VeleroObjectStorageNamespaceName }}.compat.objectstorage.us-phoenix-1.oraclecloud.com`

const veleroBackup = `apiVersion: velero.io/v1
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

type accessData struct {
	ObjectStoreAccessKeyID string
	ObjectStoreAccessKey   string
}

type veleroBackupLocationObjectData struct {
	VeleroBackupStorageName          string
	VeleroNamespaceName              string
	VeleroObjectStoreBucketName      string
	VeleroSecretName                 string
	VeleroObjectStorageNamespaceName string
}

type veleroBackupObject struct {
	VeleroBackupName                 string
	VeleroNamespaceName              string
	VeleroBackupStorageName          string
	VeleroOpensearchHookResourceName string
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
	data := accessData{ObjectStoreAccessKeyID: OciOsAccessKey, ObjectStoreAccessKey: OciOsAccessSecretKey}
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
		t.Logs.Infof("Error creating secret ", zap.Error(err))
		return err
	}

	return err
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
	}
	spew.Dump(data)
	template.Execute(&b, data)
	spew.Dump(b.String())
	err := pkg.CreateOrUpdateResourceFromBytes(b.Bytes())
	if err != nil {
		t.Logs.Infof("Error creating velero backup loaction ", zap.Error(err))
		return err
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
		t.Logs.Infof("Error creating velero backup ", zap.Error(err))
		return err
	}
	return nil
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

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs

	retryCount := 0

	for {
		bashResponse := Runner(&kcmd, t.Logs)
		if bashResponse.CommandError != nil {
			return bashResponse.CommandError
		}
		switch bashResponse.StandardOut.String() {
		case "InProgress":
			if retryCount > 100 {
				return fmt.Errorf("retry count to monitor backup '%s' exceeded", BackupOpensearchName)
			}
			t.Logs.Infof("Backup '%s' is in progress. Check after 10 seconds", BackupOpensearchName)
			time.Sleep(10 * time.Second)
		case "Completed":
			t.Logs.Infof("Backup '%s' completed successfully.", BackupOpensearchName)
			return nil
		default:
			return fmt.Errorf("Backup '%s' did not complete successfully. State = '%s'", BackupOpensearchName, bashResponse.StandardOut.String())
		}
		retryCount = retryCount + 1
	}
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

	t.Logs.Infof("Deleting opensearch amster sts")
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

	err = checkPods("verrazzano-component=opensearch", constants.VerrazzanoSystemNamespace)
	if err != nil {
		return err
	}

	return checkPvcs("verrazzano-component=opensearch", constants.VerrazzanoSystemNamespace)
}

func checkPods(labelSelector, namespace string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

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

}

func checkPvcs(labelSelector, namespace string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	retryCount := 0
	for {
		listOptions := metav1.ListOptions{LabelSelector: labelSelector}
		pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return err
		}
		if len(pvcs.Items) > 0 {
			if retryCount > 100 {
				return fmt.Errorf("retry count to monitor pvcs exceeded!!")
			}
			t.Logs.Infof("Pvcs with label selector '%s' in namespace '%s' are still present", labelSelector, namespace)
			time.Sleep(10 * time.Second)
		} else {
			return nil
		}
	}

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
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create backup storage location for velero backup objects")
	Eventually(func() error {
		return CreateVeleroBackupLocationObject()
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

}

var _ = t.Describe("Start Backup,", Label("f:platform-verrazzano.backup"), func() {
	t.Context("after velero backup storage location created", func() {
		// GIVEN the Velero is installed
		// WHEN we check to make sure the namespace exists
		// THEN we successfully find the namespace
		WhenVeleroInstalledIt("Start velero backup", func() {
			Eventually(func() error {
				return CreateVeleroBackupObject()
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())
		})

		WhenVeleroInstalledIt("Check velero backup progress", func() {
			Eventually(func() error {
				return CheckBackupProgress()
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())
		})

		WhenVeleroInstalledIt("Nuke opensearch", func() {
			Eventually(func() error {
				return NukeOpensearch()
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())
		})

	})
})
