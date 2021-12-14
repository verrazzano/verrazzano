// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsreceiver

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewPrometheusMetricsReceiver(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{PushGatewayURL: "http://somegateway"}
	rcvr, err := NewMetricsReceiver(cfg.Name)
	assert.NoError(t, err)
	assert.NotNil(t, rcvr)

	// TODO mock pusher and assert on its settings
	//rcvr.promPusher.Push()
}

func TestPrometheusMetricsReceiver_IncrementCounter(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{PushGatewayURL: "http://somegateway", PushGatewayUser: "someuser", PushGatewayPassword: "pass"}
	receiver, err := NewMetricsReceiver(cfg.Name)
	assert.NoError(t, err)
	desc := MetricDesc{Name: "MyCounter1"}
	err = receiver.IncrementGokitCounter(desc)
	assert.NoError(t, err)
	// again, to test existing counter
	receiver.IncrementGokitCounter(desc)

	// TODO mock pusher
}
