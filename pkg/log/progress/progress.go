// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package progress

import (
	"fmt"
	"time"
)

// historyMap contains a map of historyLog objects
var historyMap = make(map[string]*historyLog)

type Logger interface {
	Info(args ...interface{})
}

// historyLog contains the history messages logged
type historyLog struct {
	// Last time message was logged
	lastLogTime *time.Time

	// Message logged
	msg string
}

// ProgressLogger logs a message periodically
type ProgressLogger struct {
	// name is the name of the logger
	name string

	// frequency between logs in seconds
	frequencySecs int

	// logger is the logger used to log the message.
	logger Logger
}

// NewProgressLogger creates a new ProgressLogger
func NewProgressLogger(log Logger, name string) ProgressLogger {
	return ProgressLogger{
		logger:        log,
		name:          name,
		frequencySecs: 60,
	}
}

// Infof formats a message and logs it
func (p ProgressLogger) Infof(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	p.Info(s)
}

// Info logs an info message either now or sometime in the future.  The message
// will be logged only if it is new or the next log time has been reached.
// This function allows a controller to constantly log info messages very frequently,
// such as "waiting for Verrazzano secret", but the message will only be logged
// once periodically according to the frequency (e.g. once every 60 seconds).
// If the log message is new or has changed then it is logged immediately.
func (p ProgressLogger) Info(args ...interface{}) {
	msg := fmt.Sprint(args...)
	now := time.Now()

	// Get the history for this key, create a new one if needed
	history, ok := historyMap[p.name]
	if !ok {
		history = &historyLog{}
		historyMap[p.name] = history
	}
	// Log now if the message changed or wait time exceeded
	logNow := true
	if msg == history.msg {
		// Check if it is time to log since the message didn't change
		waitSecs := time.Duration(p.frequencySecs) * time.Second
		nextLogTime := history.lastLogTime.Add(waitSecs)
		logNow = now.Equal(nextLogTime) || now.After(nextLogTime)
	}
	if logNow {
		p.logger.Info(msg)
		history.lastLogTime = &now
		history.msg = msg
	}
}

// Set the log frequency
func (p ProgressLogger) SetFrequency(secs int) ProgressLogger {
	p.frequencySecs = secs
	return p
}
