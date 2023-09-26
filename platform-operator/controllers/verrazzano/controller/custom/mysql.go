// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package custom

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// CleanupMysqlBackupJob checks for the existence of a stale MySQL restore job and deletes the job if one is found
func CleanupMysqlBackupJob(log vzlog.VerrazzanoLogger, cli client.Client) error {
	// Check if jobs for running the restore jobs exist
	jobsFound := &batchv1.JobList{}
	err := cli.List(context.TODO(), jobsFound, &client.ListOptions{Namespace: mysql.ComponentNamespace})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for _, job := range jobsFound.Items {
		// get and inspect the job pods to see if restore container is completed
		podList := &corev1.PodList{}
		podReq, _ := kblabels.NewRequirement("job-name", selection.Equals, []string{job.Name})
		podLabelSelector := kblabels.NewSelector()
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
					log.Oncef("Unable to persist job log for %s", backupJob.Name)
				}
				// can delete job since pod has completed
				log.Debugf("Deleting stale backup job %s", job.Name)
				propagationPolicy := metav1.DeletePropagationBackground
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
	containerName := mysqloperator.BackupContainerName
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
		if strings.HasPrefix(container.Name, mysqloperator.BackupContainerName) && container.State.Terminated != nil && container.State.Terminated.ExitCode == 0 {
			return true
		}
	}
	return false
}

// IsMysqlOperatorJob returns true if this is the MySQL backup job
func IsMysqlOperatorJob(cli client.Client, object client.Object) bool {
	// Cast object to job
	job := object.(*batchv1.Job)

	// Filter events to only be for the MySQL namespace
	if job.Namespace != mysql.ComponentNamespace {
		return false
	}

	// see if the job ownerReferences point to a cron job owned by the mysql operato
	for _, owner := range job.GetOwnerReferences() {
		if owner.Kind == "CronJob" {
			// get the cronjob reference
			cronJob := &batchv1.CronJob{}
			err := cli.Get(context.TODO(), client.ObjectKey{Name: owner.Name, Namespace: job.Namespace}, cronJob)
			if err != nil {
				return false
			}
			return isResourceCreatedByMysqlOperator(cronJob.Labels)
		}
	}

	// see if the job has been directly created by the mysql operator
	return isResourceCreatedByMysqlOperator(job.Labels)
}

// isResourceCreatedByMysqlOperator checks whether the created-by label is set to "mysql-operator"
func isResourceCreatedByMysqlOperator(labels map[string]string) bool {
	createdBy, ok := labels["app.kubernetes.io/created-by"]
	if !ok || createdBy != constants.MySQLOperator {
		return false
	}
	return true
}
