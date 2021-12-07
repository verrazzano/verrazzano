// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics_receiver

import (
"testing"

"github.com/stretchr/testify/assert"
)

func TestNewPrometheusMetricsReceiver(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{pushGatewayURL: "http://somegateway"}
	rcvr, err := newMetricsReceiver(cfg.Name)
	assert.NoError(t, err)
	assert.NotNil(t, rcvr)

	// TODO mock pusher and assert on its settings
	//rcvr.promPusher.Push()
}

func TestPrometheusMetricsReceiver_IncrementCounter(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{pushGatewayURL: "http://somegateway", pushGatewayUser: "someuser", pushGatewayPassword: "pass"}
	receiver, err := newMetricsReceiver(cfg.Name)
	assert.NoError(t, err)
	desc := MetricDesc{Name: "MyCounter1"}
	err = receiver.IncrementCounter(desc)
	assert.NoError(t, err)
	// again, to test existing counter
	err = receiver.IncrementCounter(desc)

	// TODO mock pusher
}