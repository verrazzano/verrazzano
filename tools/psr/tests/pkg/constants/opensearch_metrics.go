// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

var OpensearchGetLogsMetrics = []string{
	"psr_opensearch_getlogs_data_chars_total",
	"psr_opensearch_getlogs_failure_count_total",
	"psr_opensearch_getlogs_failure_latency_nanoseconds",
	"psr_opensearch_getlogs_loop_count_total",
	"psr_opensearch_getlogs_success_count_total",
	"psr_opensearch_getlogs_success_latency_nanoseconds",
	"psr_opensearch_getlogs_worker_last_loop_nanoseconds",
	"psr_opensearch_getlogs_worker_running_seconds_total",
	"psr_opensearch_getlogs_worker_thread_count_total",
}

var OpensearchPostLogsMetrics = []string{
	"psr_opensearch_postlogs_data_chars_total",
	"psr_opensearch_postlogs_failure_count_total",
	"psr_opensearch_postlogs_failure_latency_nanoseconds",
	"psr_opensearch_postlogs_loop_count_total",
	"psr_opensearch_postlogs_success_count_total",
	"psr_opensearch_postlogs_success_latency_nanoseconds",
	"psr_opensearch_postlogs_worker_last_loop_nanoseconds",
	"psr_opensearch_postlogs_worker_running_seconds_total",
	"psr_opensearch_postlogs_worker_thread_count_total",
}

var OpensearchRestartMetrics = []string{
	"psr_opensearch_restart_loop_count_total",
	"psr_opensearch_restart_pod_restart_count",
	"psr_opensearch_restart_pod_restart_time_nanoseconds",
	"psr_opensearch_restart_worker_last_loop_nanoseconds",
	"psr_opensearch_restart_worker_running_seconds_total",
	"psr_opensearch_restart_worker_thread_count_total",
}

var OpensearchScalingMetrics = []string{
	"psr_opensearch_scaling_loop_count_total",
	"psr_opensearch_scaling_scale_in_count_total",
	"psr_opensearch_scaling_scale_in_seconds",
	"psr_opensearch_scaling_scale_out_count_total",
	"psr_opensearch_scaling_scale_out_seconds",
	"psr_opensearch_scaling_worker_last_loop_nanoseconds",
	"psr_opensearch_scaling_worker_running_seconds_total",
	"psr_opensearch_scaling_worker_thread_count_total",
}

var OpensearchWritelogsMetrics = []string{
	"psr_opensearch_writelogs_logged_chars_total",
	"psr_opensearch_writelogs_logged_lines_count_total",
	"psr_opensearch_writelogs_loop_count_total",
	"psr_opensearch_writelogs_worker_last_loop_nanoseconds",
	"psr_opensearch_writelogs_worker_running_seconds_total",
	"psr_opensearch_writelogs_worker_thread_count_total",
}

func GetOpsS2Metrics() []string {
	metrics := OpensearchGetLogsMetrics
	metrics = append(metrics, OpensearchWritelogsMetrics...)
	return metrics
}
