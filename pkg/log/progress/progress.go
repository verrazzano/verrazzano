// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package progress

import (
	"fmt"
	"go.uber.org/zap"
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

// ProgressLogger is a logger interface that provides Verrazzano base and progress logging
type ProgressLogger interface {
	Progress(args ...interface{})
	Progressf(template string, args ...interface{})
	SetFrequency(secs int) VerrazzanoLogger
}

// VerrazzanoLogger is a logger interface that provides sugared and progress logging
type VerrazzanoLogger interface {
	SugaredLogger
	ProgressLogger
	SetZapLogger(zap *zap.SugaredLogger)
	GetZapLogger() *zap.SugaredLogger
}

// LogContext is the log context for a given resource.
// This logger can be used to manage logging for the resource and sub-resources, such as components
type LogContext struct {
	// zapLogger is the zap SugaredLogger
	zapLogger *zap.SugaredLogger

	// sLogger is the interface used to log
	sLogger SugaredLogger

	// loggerMap contains a map of verrazzanoLogger objects
	loggerMap map[string]*verrazzanoLogger
}

// verrazzanoLogger implements the VerrazzanoLogger interface
type verrazzanoLogger struct {
	// context is the log context
	context *LogContext

	// frequency between logs in seconds
	frequencySecs int

	// history is a set of log messages for this logger
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
func EnsureLogContext(key string, sLogger SugaredLogger, zap *zap.SugaredLogger) *LogContext {
	log, ok := LogContextMap[key]
	if !ok {
		log = &LogContext{
			zapLogger: zap,
			sLogger:   sLogger,
			loggerMap: make(map[string]*verrazzanoLogger),
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

// DefaulLogger ensures that a new default verrazzanoLogger exists
func (c *LogContext) DefaulLogger() VerrazzanoLogger {
	return c.EnsureLogger("default")
}

// EnsureLogger ensures that a new VerrazzanoLogger exists for the given key
func (c *LogContext) EnsureLogger(key string) VerrazzanoLogger {
	log, ok := c.loggerMap[key]
	if !ok {
		log = &verrazzanoLogger{
			context:         c,
			frequencySecs:   60,
			historyMessages: make(map[string]bool),
		}
		c.loggerMap[key] = log
	}
	return log
}

// Progressf formats a message and logs it
func (m *verrazzanoLogger) Progressf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	m.Progress(s)
}

// Progress logs an info message either now or sometime in the future.  The message
// will be logged only if it is new or the next log time has been reached.
// This function allows a controller to constantly log info messages very frequently,
// such as "waiting for Verrazzano secret", but the message will only be logged
// once periodically according to the frequency (e.g. once every 60 seconds).
// If the log message is new or has changed then it is logged immediately.
func (m *verrazzanoLogger) Progress(args ...interface{}) {
	msg := fmt.Sprint(args...)
	now := time.Now()

	// If this message is in the history map, that means it has been
	// logged already, previous to the current message.  This happens
	// if a controller reconcile loop is called repeatedly.  In this
	// case we never want to display this message again, so just ignore it.
	_, ok := m.historyMessages[msg]
	if ok {
		return
	}
	// Log now if the message changed or wait time exceeded
	logNow := true

	// If the message has changed, then save the old message in the history
	// so that it is never displayed again
	if m.lastLog != nil {
		if msg != m.lastLog.msgLogged {
			m.historyMessages[msg] = true
		} else {
			// Check if it is time to log since the message didn't change
			waitSecs := time.Duration(m.frequencySecs) * time.Second
			nextLogTime := m.lastLog.lastLogTime.Add(waitSecs)
			logNow = now.Equal(nextLogTime) || now.After(nextLogTime)
		}
	}

	// Log the message if it is time and save the lastLog info
	if logNow {
		m.context.sLogger.Info(msg)
		m.lastLog = &lastLog{
			lastLogTime: &now,
			msgLogged:   msg,
		}
	}
}

// SetFrequency sets the log frequency
func (v *verrazzanoLogger) SetFrequency(secs int) VerrazzanoLogger {
	v.frequencySecs = secs
	return v
}

// Debug is a wrapper for SugaredLogger Debug
func (v *verrazzanoLogger) Debug(args ...interface{}) {
	v.context.sLogger.Info(args...)
}

// Debugf is a wrapper for SugaredLogger Debugf
func (v *verrazzanoLogger) Debugf(template string, args ...interface{}) {
	v.context.sLogger.Infof(template, args...)
}

// Info is a wrapper for SugaredLogger Info
func (v *verrazzanoLogger) Info(args ...interface{}) {
	v.context.sLogger.Info(args...)
}

// Infof is a wrapper for SugaredLogger Infof
func (v *verrazzanoLogger) Infof(template string, args ...interface{}) {
	v.context.sLogger.Infof(template, args...)
}

// Error is a wrapper for SugaredLogger Error
func (v *verrazzanoLogger) Error(args ...interface{}) {
	v.context.sLogger.Error(args...)
}

// Errorf is a wrapper for SugaredLogger Errorf
func (v *verrazzanoLogger) Errorf(template string, args ...interface{}) {
	v.context.sLogger.Errorf(template, args...)
}

// SetZapLogger sets the zap logger
func (v *verrazzanoLogger) SetZapLogger(zap *zap.SugaredLogger) {
	v.context.zapLogger = zap
	v.context.sLogger = zap
}

// GetZapLogger gets the zap logger
func (v *verrazzanoLogger) GetZapLogger() *zap.SugaredLogger {
	return v.context.zapLogger
}
