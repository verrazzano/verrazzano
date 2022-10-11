// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	testmetrics "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

const (
	metricsReceiverTypeEnvVarName = "METRICS_RECEIVER_TYPE"
	promReceiverType              = "Prometheus"

	// Prometheus related env vars and constants
	promPushURLEnvVarName      = "PROMETHEUS_GW_URL"
	promPushUserEnvVarName     = "PROMETHEUS_CREDENTIALS_USR"
	promPushPasswordEnvVarName = "PROMETHEUS_CREDENTIALS_PSW"
	defaultPushInterval        = time.Minute
)

var getenvFunc = os.Getenv

// isBodyFunc - return boolean indicating if the interface is a function
func isBodyFunc(body interface{}) bool {
	bodyType := reflect.TypeOf(body)
	return bodyType.Kind() == reflect.Func
}

// createTestMetricsReceiver - Creates a MetricsReceiver for the test to use
func createTestMetricsReceiver(testName string) (testmetrics.MetricsReceiver, error) {
	// sanitize the test name to have no spaces and be all lower case
	name := strings.ReplaceAll(strings.ToLower(testName), " ", "_")
	metricsConfig, err := createMetricsConfigFromEnv(name)
	if err != nil {
		return nil, err
	}
	return testmetrics.NewMetricsReceiver(metricsConfig)
}

// createMetricsConfigFromEnv - creates a MetricsReceiverConfig based on env vars, which will be used to
// create the appropriate metrics receiver
func createMetricsConfigFromEnv(name string) (testmetrics.MetricsReceiverConfig, error) {
	metricsReceiverType := getMetricsReceiverType()
	switch metricsReceiverType {
	case promReceiverType:
		cfg := &testmetrics.PrometheusMetricsReceiverConfig{
			PushGatewayURL:      getenvFunc(promPushURLEnvVarName),
			PushGatewayUser:     getenvFunc(promPushUserEnvVarName),
			PushGatewayPassword: getenvFunc(promPushPasswordEnvVarName),
			PushInterval:        defaultPushInterval,
			Name:                name,
		}
		if cfg.PushGatewayURL == "" || cfg.PushGatewayUser == "" || cfg.PushGatewayPassword == "" {
			return nil, fmt.Errorf("missing entries in Prometheus configuration. %s, %s and %s must be set",
				promPushURLEnvVarName, promPushUserEnvVarName, promPushPasswordEnvVarName)
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("unsupported %s value: %s", metricsReceiverTypeEnvVarName, metricsReceiverType)
	}
}

// Get the metrics receiver type set in the environment, defaulting to Prometheus
func getMetricsReceiverType() string {
	metricsReceiverType, isSet := os.LookupEnv(metricsReceiverTypeEnvVarName)
	if !isSet {
		metricsReceiverType = promReceiverType
	}
	return metricsReceiverType
}

func EmitGauge(testName string, metricName string, value float64) error {
	metricsReceiver, err := createTestMetricsReceiver(testName)
	if err != nil {
		return err
	}
	return metricsReceiver.SetGauge(metricName, value)
}

func IncrementCounter(testName string, metricName string) error {
	metricsReceiver, err := createTestMetricsReceiver(testName)
	if err != nil {
		return err
	}
	return metricsReceiver.IncrementCounter(metricName)
}
