// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package writelogs

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"strings"
	"testing"
)

// TestGetters tests the worker getters
// GIVEN a worker
//
//	WHEN the getter methods are calls
//	THEN ensure that the correct results are returned
func TestGetters(t *testing.T) {
	w, err := NewWriteLogsWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeWriteLogs, wd.WorkerType)
	assert.Equal(t, "The writelogs worker writes logs to STDOUT, putting a load on OpenSearch", wd.Description)
	assert.Equal(t, config.WorkerTypeWriteLogs, wd.MetricsName)

	el := w.GetEnvDescList()
	assert.Len(t, el, 0)

	logged := w.WantLoopInfoLogged()
	assert.False(t, logged)
}

func TestGetMetricDescList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "logged_lines_count_total", help: "The total number of lines logged"},
		{name: "2", fqName: "logged_chars_total", help: "The total number of characters logged"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewWriteLogsWorker()
			w := wi.(worker)
			assert.NoError(t, err)
			dl := w.GetMetricDescList()
			var found int
			for _, d := range dl {
				s := d.String()
				if strings.Contains(s, test.fqName) && strings.Contains(s, test.help) {
					found++
				}
			}
			assert.Equal(t, 1, found)
		})
	}
}

func TestGetMetricList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "logged_lines_count_total", help: "The total number of lines logged"},
		{name: "2", fqName: "logged_chars_total", help: "The total number of characters logged"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewWriteLogsWorker()
			w := wi.(worker)
			assert.NoError(t, err)
			ml := w.GetMetricList()
			var found int
			for _, m := range ml {
				s := m.Desc().String()
				if strings.Contains(s, test.fqName) && strings.Contains(s, test.help) {
					found++
				}
			}
			assert.Equal(t, 1, found)
		})
	}
}

// TestDoWork tests the DoWork method
// GIVEN a worker
//
//	WHEN the DoWork methods is called
//	THEN ensure that the correct results are returned
func TestDoWork(t *testing.T) {
	wi, err := NewWriteLogsWorker()
	assert.NoError(t, err)
	w := wi.(worker)
	err = w.DoWork(config.CommonConfig{
		WorkerType: "Fake",
	}, vzlog.DefaultLogger())
	assert.NoError(t, err)
	assert.Equal(t, int64(31), w.loggedCharsCountTotal.Val)
	assert.Equal(t, int64(1), w.loggedLinesCountTotal.Val)
}
