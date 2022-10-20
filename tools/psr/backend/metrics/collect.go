// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

//import (
//	"fmt"
//	"github.com/prometheus/client_golang/prometheus"
//)
//
//// https://build.verrazzano.io/api/json?depth=3&tree=jobs[name,url,lastCompletedBuild[url,number,duration,timestamp,result],jobs[name,url,lastCompletedBuild[url,number,duration,timestamp,result]]]
//// repeat jobs section <depth> times
//// https://build.verrazzano.io/job/verrazzano-new-oci-dns-acceptance-tests/job/master/wfapi/runs
//// https://build.verrazzano.io/api/json
//
//var (
//	PrometheusNamespace = "vzjenkins"
//	runLabels           = []string{
//		"jenkins_job",
//		"branch_name",
//		"status",
//	}
//	stageLabels = []string{
//		"jenkins_job",
//		"branch_name",
//		"stage_id",
//		"stage_name",
//		"status",
//	}
//)
//
//type RunCollector struct {
//	prometheus.Collector
//	runDesc                *prometheus.Desc
//	runDurationDesc        *prometheus.Desc
//	stageDesc              *prometheus.Desc
//	stageDurationDesc      *prometheus.Desc
//	stagePauseDurationDesc *prometheus.Desc
//	stageQueueDurationDesc *prometheus.Desc
//	jenkinsJobCollector    *jenkins.JobCollector
//}
//
//func NewRunCollector(collectors []prometheus.Collector) (*RunCollector, error) {
//	constLabels := prometheus.Labels{
//		"jenkins_url": jobCollector.JenkinsURL,
//	}
//
//	rc := &RunCollector{
//		runDesc: prometheus.NewDesc(
//			prometheus.BuildFQName(PrometheusNamespace, "", "last_run"),
//			"The last run status (-1 UNKNOWN, 0 SUCCESS, 1 FAIULRE, 2 ABORTED)",
//			runLabels,
//			constLabels,
//		),
//		runDurationDesc: prometheus.NewDesc(
//			prometheus.BuildFQName(PrometheusNamespace, "", "last_run_duration_seconds"),
//			"The duration of the last run in seconds.",
//			runLabels,
//			constLabels,
//		),
//		stageDesc: prometheus.NewDesc(
//			prometheus.BuildFQName(PrometheusNamespace, "", "last_run_stage"),
//			"The status of the stage in the last run (-1 UNKNOWN, 0 SUCCESS, 1 FAIULRE)",
//			stageLabels,
//			constLabels,
//		),
//		stageDurationDesc: prometheus.NewDesc(
//			prometheus.BuildFQName(PrometheusNamespace, "", "last_run_stage_duration_seconds"),
//			"The duration in seconds of the stage in the last run.",
//			stageLabels,
//			constLabels,
//		),
//		stagePauseDurationDesc: prometheus.NewDesc(
//			prometheus.BuildFQName(PrometheusNamespace, "", "last_run_stage_pause_duration_seconds"),
//			"The duration in seconds of the pause time for the stage in the last run.",
//			stageLabels,
//			constLabels,
//		),
//		stageQueueDurationDesc: prometheus.NewDesc(
//			prometheus.BuildFQName(PrometheusNamespace, "", "last_run_stage_queue_duration_seconds"),
//			"The duration in seconds of the queue time for the stage in the last run.",
//			stageLabels,
//			constLabels,
//		),
//	}
//
//	rc.jenkinsJobCollector = jobCollector
//
//	return rc, nil
//}
//
//func (rc RunCollector) Describe(ch chan<- *prometheus.Desc) {
//	prometheus.DescribeByCollect(rc, ch)
//}
//
//func (rc RunCollector) Collect(ch chan<- prometheus.Metric) {
//
//	jch, err := rc.jenkinsJobCollector.Collect()
//	if err != nil {
//		fmt.Printf("Get Jobs Error: %v\n", err)
//		return
//	}
//	for {
//		v, ok := <-jch
//		if ok == false {
//			break
//		}
//		rc.recordMetricsForJob(v, ch)
//	}
//}
//
//func (rc RunCollector) recordMetricsForJob(job *jenkins.JenkinsJob, ch chan<- prometheus.Metric) {
//	ch <- prometheus.MustNewConstMetric(
//		rc.runDesc,
//		prometheus.GaugeValue,
//		float64(job.JobStatus),
//		job.JobSummary.Name, job.Branch, job.JobStatus.String(),
//	)
//	ch <- prometheus.MustNewConstMetric(
//		rc.runDurationDesc,
//		prometheus.GaugeValue,
//		float64(job.JobSummary.LastCompletedBuild.DurationMillis/1000),
//		job.JobSummary.Name, job.Branch, job.JobStatus.String(),
//	)
//	if job.WorkflowRunSummary == nil {
//		return
//	}
//	for _, stage := range job.WorkflowRunSummary.Stages {
//		stageResult := -1
//		switch stage.Status {
//		case "SUCCESS":
//			stageResult = 0
//		case "FAILED":
//			stageResult = 1
//		}
//		ch <- prometheus.MustNewConstMetric(
//			rc.stageDesc,
//			prometheus.GaugeValue,
//			float64(stageResult),
//			job.JobSummary.Name, job.Branch, stage.Id, stage.Name, stage.Status,
//		)
//		ch <- prometheus.MustNewConstMetric(
//			rc.stageDurationDesc,
//			prometheus.GaugeValue,
//			float64(stage.DurationMillis/1000),
//			job.JobSummary.Name, job.Branch, stage.Id, stage.Name, stage.Status,
//		)
//		ch <- prometheus.MustNewConstMetric(
//			rc.stagePauseDurationDesc,
//			prometheus.GaugeValue,
//			float64(stage.PauseDurationMillis/1000),
//			job.JobSummary.Name, job.Branch, stage.Id, stage.Name, stage.Status,
//		)
//		ch <- prometheus.MustNewConstMetric(
//			rc.stageQueueDurationDesc,
//			prometheus.GaugeValue,
//			float64(stage.QueueDurationMillis/1000),
//			job.JobSummary.Name, job.Branch, stage.Id, stage.Name, stage.Status,
//		)
//	}
//
//}
