// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

// TestIsBodyFunc - test function for introspecting an interface value
func TestIsBodyFunc(t *testing.T) {
	type args struct {
		body interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test using a function",
			args: args{body: func() {}},
			want: true,
		},
		{
			name: "Test using a struct",
			args: args{body: args{}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBodyFunc(tt.args.body); got != tt.want {
				t.Errorf("isBodyFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createMetricsConfigFromEnv(t *testing.T) {
	_, err := createMetricsConfigFromEnv("testname")
	assert.Error(t, err)

	defer func() {
		getenvFunc = os.Getenv
	}()

	testURL := "https://some.pushgateway.url"
	testUser := "myuser"
	getenvFunc = promGetEnvTestFunc(testURL, testUser, "")
	_, err = createMetricsConfigFromEnv("testname")

	assert.Error(t, err, "expected error creating config when Prometheus push gateway password is not set")

	testPass := "somepass"
	getenvFunc = promGetEnvTestFunc(testURL, testUser, testPass)
	cfg, err := createMetricsConfigFromEnv("testname")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "*metrics.PrometheusMetricsReceiverConfig", reflect.TypeOf(cfg).String())
	promCfg := cfg.(*metrics.PrometheusMetricsReceiverConfig)
	assert.Equal(t, testURL, promCfg.PushGatewayURL)
	assert.Equal(t, testUser, promCfg.PushGatewayUser)
	assert.Equal(t, testPass, promCfg.PushGatewayPassword)
}

func promGetEnvTestFunc(testURL string, testUser string, testPass string) func(key string) string {
	return func(key string) string {
		switch key {
		case promPushURLEnvVarName:
			return testURL
		case promPushUserEnvVarName:
			return testUser
		case promPushPasswordEnvVarName:
			return testPass
		default:
			return ""
		}
	}
}
