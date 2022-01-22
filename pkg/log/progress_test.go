// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package log

import (
	"fmt"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
	"time"

	"go.uber.org/zap"
)

type fakeLogger struct {
}

func TestLog(t *testing.T) {
	testOpts := kzap.Options{}
	testOpts.Development = true
	testOpts.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	InitLogs(testOpts)
	l := NewProgressLogger(&fakeLogger{}, "a").SetFrequency(3)
	for i := 0; i < 30; i++ {
		l.Log("test1")
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func (l fakeLogger) Info(args ...interface{}) {
	fmt.Println(args...)
}
