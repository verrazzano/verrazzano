// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"io"
	"k8s.io/api/batch/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	config "github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// getOverrides gets the install overrides
func getOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.MySQLOperator != nil {
			return effectiveCR.Spec.Components.MySQLOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.MySQLOperator != nil {
			return effectiveCR.Spec.Components.MySQLOperator.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// AppendOverrides Build the set of MySQL operator overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

	var secret corev1.Secret
	if err := compContext.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: constants.GlobalImagePullSecName}, &secret); err != nil {
		if errors.IsNotFound(err) {
			// Global secret not found
			return kvs, nil
		}
		// we had an unexpected error
		return kvs, err
	}

	// We found the global secret, set the image.pullSecrets.enabled value to true
	kvs = append(kvs, bom.KeyValue{Key: "image.pullSecrets.enabled", Value: "true"})
	return kvs, nil
}

// isReady - component specific checks for being ready
func (c mysqlOperatorComponent) isReady(ctx spi.ComponentContext) bool {
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), c.AvailabilityObjects.DeploymentNames, 1, getPrefix(ctx))
}

// isInstalled checks that the deployment exists
func (c mysqlOperatorComponent) isInstalled(ctx spi.ComponentContext) bool {
	return ready.DoDeploymentsExist(ctx.Log(), ctx.Client(), c.AvailabilityObjects.DeploymentNames, 1, getPrefix(ctx))
}

func getPrefix(ctx spi.ComponentContext) string {
	return fmt.Sprintf("Component %s", ctx.GetComponent())
}

func getDeploymentList() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
}

// validateMySQLOperator checks scenarios in which the Verrazzano CR violates install verification
// MySQLOperator must be enabled if Keycloak is enabled
func (c mysqlOperatorComponent) validateMySQLOperator(vz *installv1beta1.Verrazzano) error {
	// Validate install overrides
	if vz.Spec.Components.MySQLOperator != nil {
		if err := vzapi.ValidateInstallOverridesV1Beta1(vz.Spec.Components.MySQLOperator.ValueOverrides); err != nil {
			return err
		}
	}
	// Must be enabled if Keycloak is enabled
	if config.IsKeycloakEnabled(vz) {
		if !c.IsEnabled(vz) {
			return fmt.Errorf("MySQLOperator must be enabled if Keycloak is enabled")
		}
	}
	return nil
}

// doesInnoDBClusterExist returns true if the InnoDBCluster resource exists
func doesInnoDBClusterExist(ctx spi.ComponentContext) (bool, error) {
	innoDBClusterGVK := schema.GroupVersionKind{
		Group:   "mysql.oracle.com",
		Version: "v2",
		Kind:    "InnoDBCluster",
	}
	const innoDBName = "mysql"

	innoDBCluster := unstructured.Unstructured{}
	innoDBCluster.SetGroupVersionKind(innoDBClusterGVK)

	// the InnoDBCluster resource name is the helm release name
	nsn := types.NamespacedName{Namespace: ComponentNamespace, Name: innoDBName}
	if err := ctx.Client().Get(context.Background(), nsn, &innoDBCluster); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		ctx.Log().Errorf("Error retrieving InnoDBCluster %v: %v", nsn, err)
		return false, err
	}
	return true, nil
}

// IsMysqlOperatorJob returns true if the job is spawned directly or indirectly by MySQL operator
func IsMysqlOperatorJob(c client.Client, job batchv1.Job, log vzlog.VerrazzanoLogger) bool {

	// Filter events to only be for the MySQL namespace
	if job.Namespace != constants2.KeycloakNamespace {
		return false
	}

	// see if the job ownerReferences point to a cron job owned by the mysql operato
	for _, owner := range job.GetOwnerReferences() {
		if owner.Kind == "CronJob" {
			// get the cronjob reference
			cronJob := &v1.CronJob{}
			err := c.Get(context.TODO(), client.ObjectKey{Name: owner.Name, Namespace: job.Namespace}, cronJob)
			if err != nil {
				log.Errorf("Could not find cronjob %s to ascertain job source", owner.Name)
				return false
			}
			return isResourceCreatedByMysqlOperator(cronJob.Labels, log)
		}
	}

	// see if the job has been directly created by the mysql operator
	return isResourceCreatedByMysqlOperator(job.Labels, log)
}

// isResourceCreatedByMysqlOperator checks whether the created-by label is set to "mysql-operator"
func isResourceCreatedByMysqlOperator(labels map[string]string, log vzlog.VerrazzanoLogger) bool {
	createdBy, ok := labels["app.kubernetes.io/created-by"]
	if !ok || createdBy != constants2.MySQLOperator {
		return false
	}
	log.Debug("Resource created by MySQL Operator")
	return true
}

// CleanupMysqlBackupJob checks for the existence of a stale MySQL restore job and deletes the job if one is found
func CleanupMysqlBackupJob(log vzlog.VerrazzanoLogger, cli client.Client) error {
	// Check if jobs for running the restore jobs exist
	jobsFound := &batchv1.JobList{}
	err := cli.List(context.TODO(), jobsFound, &client.ListOptions{Namespace: constants2.KeycloakNamespace})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for _, job := range jobsFound.Items {
		// get and inspect the job pods to see if restore container is completed
		podList := &corev1.PodList{}
		podReq, _ := labels.NewRequirement("job-name", selection.Equals, []string{job.Name})
		podLabelSelector := labels.NewSelector()
		podLabelSelector = podLabelSelector.Add(*podReq)
		err := cli.List(context.TODO(), podList, &client.ListOptions{LabelSelector: podLabelSelector})
		if err != nil {
			return err
		}
		backupJob := job
		for i := range podList.Items {
			jobPod := &podList.Items[i]
			if isJobExecutionContainerCompleted(jobPod) {
				// persist the job logs
				persisted := persistJobLog(backupJob, jobPod, log)
				if !persisted {
					log.Infof("Unable to persist job log for %s", backupJob.Name)
				}
				// can delete job since pod has completed
				log.Debugf("Deleting stale backup job %s", job.Name)
				propagationPolicy := v12.DeletePropagationBackground
				deleteOptions := &client.DeleteOptions{PropagationPolicy: &propagationPolicy}
				err = cli.Delete(context.TODO(), &backupJob, deleteOptions)
				if err != nil {
					return err
				}

				return nil
			}

			return fmt.Errorf("Pod %s has not completed the database backup", backupJob.Name)
		}
	}

	return nil
}

// persistJobLog will persist the backup job log to the VPO log
func persistJobLog(backupJob batchv1.Job, jobPod *corev1.Pod, log vzlog.VerrazzanoLogger) bool {
	containerName := BackupContainerName
	if strings.Contains(backupJob.Name, "-schedule-") {
		containerName = containerName + "-cron"
	}
	podLogOpts := corev1.PodLogOptions{Container: containerName}
	clientSet, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return false
	}
	req := clientSet.CoreV1().Pods(jobPod.Namespace).GetLogs(jobPod.Name, &podLogOpts)
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return false
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(buf)
	scanner.Split(bufio.ScanLines)
	log.Debugf("---------- Begin backup job %s log ----------", backupJob.Name)
	for scanner.Scan() {
		log.Debug(scanner.Text())
	}
	log.Debugf("---------- End backup job %s log ----------", backupJob.Name)

	return true
}

// isJobExecutionContainerCompleted checks to see whether the backup container has terminated with an exit code of 0
func isJobExecutionContainerCompleted(pod *corev1.Pod) bool {
	for _, container := range pod.Status.ContainerStatuses {
		if strings.HasPrefix(container.Name, BackupContainerName) && container.State.Terminated != nil && container.State.Terminated.ExitCode == 0 {
			return true
		}
	}
	return false
}
