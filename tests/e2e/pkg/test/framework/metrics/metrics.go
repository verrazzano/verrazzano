// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"fmt"
	"time"
)

const (
	defaultPushInterval time.Duration = time.Minute
)

type MetricsReceiver interface {
	SetGauge(name string, value float64) error
	IncrementCounter(name string) error
}

type MetricsReceiverConfig interface {
	GetReceiverType() string
}

func NewMetricsReceiver(cfg MetricsReceiverConfig) (MetricsReceiver, error) {
	switch cfg.GetReceiverType() {
	case "PrometheusMetricsReceiver":
		promConfig := cfg.(*PrometheusMetricsReceiverConfig)
		//reg := prom.NewRegistry()
		pushInterval := promConfig.PushInterval
		if pushInterval == 0 {
			pushInterval = defaultPushInterval
		}
		promConfig.PushInterval = pushInterval
		return NewPrometheusMetricsReceiver(*promConfig)
	case "FileMetricsReceiver":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown MetricsReceiver type %s", cfg.GetReceiverType())
	}
}
