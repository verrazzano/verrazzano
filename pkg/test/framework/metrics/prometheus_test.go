package testmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPrometheusMetricsReceiver(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{PushGatewayUrl: "http://somegateway"}
	rcvr, err := NewPrometheusMetricsReceiver(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, rcvr)

	cfgWithCreds := PrometheusMetricsReceiverConfig{PushGatewayUrl: "http://somegateway", PushGatewayUser: "someuser", PushGatewayPassword: "pass"}
	rcvr, err = NewPrometheusMetricsReceiver(cfgWithCreds)
	assert.NoError(t, err)
	assert.NotNil(t, rcvr)

	// TODO mock pusher and assert on its settings
	//rcvr.promPusher.Push()
}

func TestPrometheusMetricsReceiver_SetGauge(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{PushGatewayUrl: "http://somegateway", PushGatewayUser: "someuser", PushGatewayPassword: "pass"}
	receiver, err := NewPrometheusMetricsReceiver(cfg)
	assert.NoError(t, err)
	err = receiver.SetGauge("MyGauge1", 10.5)
	assert.NoError(t, err)
	// again, to test existing gauge
	err = receiver.SetGauge("MyGauge1", 12.5)
	// TODO mock pusher
}


func TestPrometheusMetricsReceiver_IncrementCounter(t *testing.T) {
	cfg := PrometheusMetricsReceiverConfig{PushGatewayUrl: "http://somegateway", PushGatewayUser: "someuser", PushGatewayPassword: "pass"}
	receiver, err := NewPrometheusMetricsReceiver(cfg)
	assert.NoError(t, err)
	err = receiver.IncrementCounter("MyCounter1")
	assert.NoError(t, err)
	// again, to test existing counter
	receiver.IncrementCounter("MyCounter1")

	// TODO mock pusher
}