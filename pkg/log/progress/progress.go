// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package progress

import (
	"fmt"
	"time"
)

// LogContextMap contains a map of LogContext objects
var LogContextMap = make(map[string]*LogContext)

// SugaredLogger is a logger interface that provides base logging
type SugaredLogger interface {
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
}

// VerrazzanoLogger is a logger interface that provides Verrazzano base and progress logging
type VerrazzanoLogger interface {
	SugaredLogger
	Progress(args ...interface{})
	Progressf(template string, args ...interface{})
}

// LogContext is the log context for a given resource.
// This logger can be used to manage logging for the resource and sub-resources, such as components
type LogContext struct {
	// sLogger is the interface used to log
	sLogger SugaredLogger

	// progressLoggerMap contains a map of ProgressLogger objects
	progressLoggerMap map[string]*ProgressLogger
}

// ProgressLogger logs a message periodically
type ProgressLogger struct {
	// sLogger is the interface used to log
	sLogger SugaredLogger

	// frequency between logs in seconds
	frequencySecs int

	// history is a set of log messages for this ProgressLogger
	historyMessages map[string]bool

	// lastLog keeps track of the last logged message
	*lastLog
}

// lastLog tracks the last message logged
type lastLog struct {
	// lastLogTime is the last time the message was logged
	lastLogTime *time.Time

	// msgLogged is the message that was logged
	msgLogged string
}

// EnsureLogContext ensures that a LogContext exists
// The key must be unique for the process, for example a namespace/name combo.
func EnsureLogContext(key string, sLogger SugaredLogger) *LogContext {
	log, ok := LogContextMap[key]
	if !ok {
		log = &LogContext{
			sLogger:           sLogger,
			progressLoggerMap: make(map[string]*ProgressLogger),
		}
		LogContextMap[key] = log
	}
	return log
}

// DeleteLogContext deletes the LogContext for the given key
func DeleteLogContext(key string) {
	_, ok := LogContextMap[key]
	if ok {
		delete(LogContextMap, key)
	}
}

// DefaultProgressLogger ensures that a new default ProgressLogger exists
func (r *LogContext) DefaultProgressLogger() *ProgressLogger {
	return r.EnsureProgressLogger("default")
}

// EnsureProgressLogger ensures that a new ProgressLogger exists for the given key
func (r *LogContext) EnsureProgressLogger(key string) *ProgressLogger {
	log, ok := r.progressLoggerMap[key]
	if !ok {
		log = &ProgressLogger{
			sLogger:         r.sLogger,
			frequencySecs:   60,
			historyMessages: make(map[string]bool),
		}
		r.progressLoggerMap[key] = log
	}
	return log
}

// Progressf formats a message and logs it
func (p *ProgressLogger) Progressf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	p.Progress(s)
}

// Progress logs an info message either now or sometime in the future.  The message
// will be logged only if it is new or the next log time has been reached.
// This function allows a controller to constantly log info messages very frequently,
// such as "waiting for Verrazzano secret", but the message will only be logged
// once periodically according to the frequency (e.g. once every 60 seconds).
// If the log message is new or has changed then it is logged immediately.
func (p *ProgressLogger) Progress(args ...interface{}) {
	msg := fmt.Sprint(args...)
	now := time.Now()

	// If this message is in the history map, that means it has been
	// logged already, previous to the current message.  This happens
	// if a controller reconcile loop is called repeatedly.  In this
	// case we never want to display this message again, so just ignore it.
	_, ok := p.historyMessages[msg]
	if ok {
		return
	}
	// Log now if the message changed or wait time exceeded
	logNow := true

	// If the message has changed, then save the old message in the history
	// so that it is never displayed again
	if p.lastLog != nil {
		if msg != p.lastLog.msgLogged {
			p.historyMessages[msg] = true
		} else {
			// Check if it is time to log since the message didn't change
			waitSecs := time.Duration(p.frequencySecs) * time.Second
			nextLogTime := p.lastLog.lastLogTime.Add(waitSecs)
			logNow = now.Equal(nextLogTime) || now.After(nextLogTime)
		}
	}

	// Log the message if it is time and save the lastLog info
	if logNow {
		p.sLogger.Info(msg)
		p.lastLog = &lastLog{
			lastLogTime: &now,
			msgLogged:   msg,
		}
	}
}

// Debug is a wrapper for SugaredLogger Debug
func (p *ProgressLogger) Debug(args ...interface{}) {
	p.sLogger.Info(args...)
}

// Debugf is a wrapper for SugaredLogger Debugf
func (p *ProgressLogger) Debugf(template string, args ...interface{}) {
	p.sLogger.Infof(template, args...)
}

// Info is a wrapper for SugaredLogger Info
func (p *ProgressLogger) Info(args ...interface{}) {
	p.sLogger.Info(args...)
}

// Infof is a wrapper for SugaredLogger Infof
func (p *ProgressLogger) Infof(template string, args ...interface{}) {
	p.sLogger.Infof(template, args...)
}

// Error is a wrapper for SugaredLogger Error
func (p *ProgressLogger) Error(args ...interface{}) {
	p.sLogger.Error(args...)
}

// Errorf is a wrapper for SugaredLogger Errorf
func (p *ProgressLogger) Errorf(template string, args ...interface{}) {
	p.sLogger.Errorf(template, args...)
}

// SetFrequency sets the log frequency
func (p ProgressLogger) SetFrequency(secs int) ProgressLogger {
	p.frequencySecs = secs
	return p
}
