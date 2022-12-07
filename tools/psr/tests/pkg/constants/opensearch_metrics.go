// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	GetLogsDataCharsTotalMetric            = "psr_opensearch_getlogs_data_chars_total"
	GetLogsFailureCountTotalMetric         = "psr_opensearch_getlogs_failure_count_total"
	GetLogsFailureLatencyNanosMetric       = "psr_opensearch_getlogs_failure_latency_nanoseconds"
	GetLogsLoopCountTotalMetric            = "psr_opensearch_getlogs_loop_count_total"
	GetLogsSuccessCountTotalMetric         = "psr_opensearch_getlogs_success_count_total"
	GetLogsSuccessLatencyNanosMetric       = "psr_opensearch_getlogs_success_latency_nanoseconds"
	GetLogsWorkerLastLoopNanosMetric       = "psr_opensearch_getlogs_worker_last_loop_nanoseconds"
	GetLogsWorkerRunningSecondsTotalMetric = "psr_opensearch_getlogs_worker_running_seconds_total"
	GetLogsWorkerThreadCountTotalMetric    = "psr_opensearch_getlogs_worker_thread_count_total"

	PostLogsDataCharsTotalMetric         = "psr_opensearch_postlogs_data_chars_total"
	PostLogsFailureCountTotalMetric      = "psr_opensearch_postlogs_failure_count_total"
	PostLogsFailureLatencyNanosMetric    = "psr_opensearch_postlogs_failure_latency_nanoseconds"
	PostLogsLoopCountTotalMetric         = "psr_opensearch_postlogs_loop_count_total"
	PostLogsSuccessCountTotalMetric      = "psr_opensearch_postlogs_success_count_total"
	PostLogsSuccessLatencyTotalMetric    = "psr_opensearch_postlogs_success_latency_nanoseconds"
	PostLogsWorkerLoopNanosMetric        = "psr_opensearch_postlogs_worker_last_loop_nanoseconds"
	PostLogsWorkerRunningSecondsMetric   = "psr_opensearch_postlogs_worker_running_seconds_total"
	PostLogsWorkerThreadCountTotalMetric = "psr_opensearch_postlogs_worker_thread_count_total"

	WriteLogsLoggedCharsTotal                = "psr_opensearch_writelogs_logged_chars_total"
	WriteLogsLoggedLinesTotalCountMetric     = "psr_opensearch_writelogs_logged_lines_count_total"
	WriteLogsLoopCountTotalMetric            = "psr_opensearch_writelogs_loop_count_total"
	WriteLogsWorkerLastLoopNanosMetric       = "psr_opensearch_writelogs_worker_last_loop_nanoseconds"
	WriteLogsWorkerRunningSecondsTotalMetric = "psr_opensearch_writelogs_worker_running_seconds_total"
	WriteLogsWorkerThreadCountTotalMetric    = "psr_opensearch_writelogs_worker_thread_count_total"

	RestartLoopCountTotalMetric            = "psr_opensearch_restart_loop_count_total"
	RestartPodRestartCountMetric           = "psr_opensearch_restart_pod_restart_count"
	RestartPodRestartTimeNanosMetric       = "psr_opensearch_restart_pod_restart_time_nanoseconds"
	RestartWorkerLastLoopNanosMetric       = "psr_opensearch_restart_worker_last_loop_nanoseconds"
	RestartWorkerRunningSecondsTotalMetric = "psr_opensearch_restart_worker_running_seconds_total"
	RestartWorkerThreadCountTotalMetric    = "psr_opensearch_restart_worker_thread_count_total"

	ScalingLoopCountTotalMetric            = "psr_opensearch_scaling_loop_count_total"
	ScalingScaleInCountTotalMetric         = "psr_opensearch_scaling_scale_in_count_total"
	ScalingScaleInSecondsMetric            = "psr_opensearch_scaling_scale_in_seconds"
	ScalingScaleOutCountTotalMetric        = "psr_opensearch_scaling_scale_out_count_total"
	ScalingScaleOutSecondsMetric           = "psr_opensearch_scaling_scale_out_seconds"
	ScalingWorkerLastLoopNanos             = "psr_opensearch_scaling_worker_last_loop_nanoseconds"
	ScalingWorkerRunningSecondsTotalMetric = "psr_opensearch_scaling_worker_running_seconds_total"
	ScalingWorkerThreadCountTotalMetric    = "psr_opensearch_scaling_worker_thread_count_total"
)

var OpensearchGetLogsMetrics = []string{
	GetLogsDataCharsTotalMetric,
	GetLogsFailureCountTotalMetric,
	GetLogsFailureLatencyNanosMetric,
	GetLogsLoopCountTotalMetric,
	GetLogsSuccessCountTotalMetric,
	GetLogsSuccessLatencyNanosMetric,
	GetLogsWorkerLastLoopNanosMetric,
	GetLogsWorkerRunningSecondsTotalMetric,
	GetLogsWorkerThreadCountTotalMetric,
}

var OpensearchPostLogsMetrics = []string{
	PostLogsDataCharsTotalMetric,
	PostLogsFailureCountTotalMetric,
	PostLogsFailureLatencyNanosMetric,
	PostLogsLoopCountTotalMetric,
	PostLogsSuccessCountTotalMetric,
	PostLogsSuccessLatencyTotalMetric,
	PostLogsWorkerLoopNanosMetric,
	PostLogsWorkerRunningSecondsMetric,
	PostLogsWorkerThreadCountTotalMetric,
}

var OpensearchRestartMetrics = []string{
	RestartLoopCountTotalMetric,
	RestartPodRestartCountMetric,
	RestartPodRestartTimeNanosMetric,
	RestartWorkerLastLoopNanosMetric,
	RestartWorkerRunningSecondsTotalMetric,
	RestartWorkerThreadCountTotalMetric,
}

var OpensearchScalingMetrics = []string{
	ScalingLoopCountTotalMetric,
	ScalingScaleInCountTotalMetric,
	ScalingScaleInSecondsMetric,
	ScalingScaleOutCountTotalMetric,
	ScalingScaleOutSecondsMetric,
	ScalingWorkerLastLoopNanos,
	ScalingWorkerRunningSecondsTotalMetric,
	ScalingWorkerThreadCountTotalMetric,
}

var OpensearchWritelogsMetrics = []string{
	WriteLogsLoggedCharsTotal,
	WriteLogsLoggedLinesTotalCountMetric,
	WriteLogsLoopCountTotalMetric,
	WriteLogsWorkerLastLoopNanosMetric,
	WriteLogsWorkerRunningSecondsTotalMetric,
	WriteLogsWorkerThreadCountTotalMetric,
}
