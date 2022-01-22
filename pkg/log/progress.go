// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package log

import (
	"time"
)

// futureMap contains a map of futureLog objects
var futureMap = make(map[string]*futureLog)

type Logger interface {
	Info(args ...interface{})
}

// futureLog contains a message to be logged in the future
type futureLog struct {
	// Next time message will be logged
	nextLogTime time.Time

	// Message to log
	msg string
}

// ProgressLogger logs a message periodically
type ProgressLogger struct {
	// name is the name of the logger
	name string

	// Frequency between logs in seconds
	frequencySecs int

	// Logger
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

// Log logs an info message for a specific key.  The message
// will be logged only if it is new or the next log time has been reached
func (p ProgressLogger) Log(msg string) {
	now := time.Now()

	// Get the tracker for this key, create a new one if needed
	future, ok := futureMap[p.name]
	if !ok {
		future = &futureLog{
			nextLogTime: now,
			msg:         msg,
		}
		futureMap[p.name] = future
	}
	// If the message changed then log now
	if msg != future.msg {
		future.nextLogTime = now
	}
	// Log the message if it is time to do so
	if now.Equal(future.nextLogTime) || now.After(future.nextLogTime) {
		p.logger.Info(msg)

		// Calculate next time to log
		waitSecs := time.Duration(p.frequencySecs) * time.Second
		future.nextLogTime = future.nextLogTime.Add(waitSecs)
	}
}

// Set the log frequency
func (p ProgressLogger) SetFrequency(secs int) ProgressLogger {
	p.frequencySecs = secs
	return p
}
