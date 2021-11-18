package testmetrics

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
		return NewPrometheusMetricsReceiver(*promConfig)
	case "FileMetricsReceiver":
		return nil,nil
	default:
		return nil, fmt.Errorf("unknown MetricsReceiver type %s", cfg.GetReceiverType())
	}
}