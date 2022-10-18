// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPrometheusMetricsReceiver(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{PushGatewayURL: "http://somegateway"}
	rcvr, err := NewPrometheusMetricsReceiver(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, rcvr)

	cfgWithCreds := PrometheusMetricsReceiverConfig{PushGatewayURL: "http://somegateway", PushGatewayUser: "someuser", PushGatewayPassword: "pass"}
	rcvr, err = NewPrometheusMetricsReceiver(cfgWithCreds)
	assert.NoError(t, err)
	assert.NotNil(t, rcvr)

	// TODO mock pusher and assert on its settings
	//rcvr.promPusher.Push()
}

func TestPrometheusMetricsReceiver_SetGauge(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{
		PushGatewayURL: "http://somegateway", PushGatewayUser: "someuser", PushGatewayPassword: "pass",
		Name: "Test1"}
	receiver, err := NewPrometheusMetricsReceiver(cfg)
	assert.NoError(t, err)
	err = receiver.SetGauge("MyGauge1", 10.5)
	assert.NoError(t, err)
	// again, to test existing gauge
	err = receiver.SetGauge("MyGauge1", 12.5)
	assert.NoError(t, err)
	// TODO mock pusher
}

func TestPrometheusMetricsReceiver_IncrementCounter(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{PushGatewayURL: "http://somegateway", PushGatewayUser: "someuser", PushGatewayPassword: "pass"}
	receiver, err := NewPrometheusMetricsReceiver(cfg)
	assert.NoError(t, err)
	err = receiver.IncrementCounter("MyCounter1")
	assert.NoError(t, err)
	// again, to test existing counter
	receiver.IncrementCounter("MyCounter1")

	// TODO mock pusher
}
