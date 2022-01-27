// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzlog

import (
	"fmt"
	"go.uber.org/zap"
	"sync"
	"time"
)

// LogContextMap contains a map of LogContext objects
var LogContextMap = make(map[string]*LogContext)

// Lock for map access
var lock sync.Mutex

// SugaredLogger is a logger interface that provides base logging
type SugaredLogger interface {
	// Debug logs a message at Debug log level
	Debug(args ...interface{})

	// Debugf formats a message and logs it once at Debug log level
	Debugf(template string, args ...interface{})

	// Info logs a message at Info log level
	Info(args ...interface{})

	// Infof formats a message and logs it once at Info log level
	Infof(template string, args ...interface{})

	// Error logs a message at Error log level
	Error(args ...interface{})

	// Errorf formats a message and logs it once at Error log level
	Errorf(template string, args ...interface{})
}

// ProgressLogger is a logger interface that provides Verrazzano base and progress logging
type ProgressLogger interface {
	// Once logs a message once at Info log level
	Once(args ...interface{})

	// Oncef formats a message and logs it once at Info log level
	Oncef(template string, args ...interface{})

	// Progress logs a message periodically at Info log level
	Progress(args ...interface{})

	// Progress formats a message and logs it periodically at Info log level
	Progressf(template string, args ...interface{})

	// SetFrequency sets the logging frequency of a progress message
	SetFrequency(secs int) VerrazzanoLogger
}

// VerrazzanoLogger is a logger interface that provides sugared and progress logging
type VerrazzanoLogger interface {
	SugaredLogger
	ProgressLogger
	SetZapLogger(zap *zap.SugaredLogger)
	GetZapLogger() *zap.SugaredLogger
	GetContext() *LogContext
}

// LogContext is the log context for a given resource.
// This logger can be used to manage logging for the resource and sub-resources, such as components
type LogContext struct {
	// loggerMap contains a map of verrazzanoLogger objects
	loggerMap map[string]*verrazzanoLogger
}

// verrazzanoLogger implements the VerrazzanoLogger interface
type verrazzanoLogger struct {
	// context is the LogContext
	context *LogContext

	// zapLogger is the zap SugaredLogger
	zapLogger *zap.SugaredLogger

	// sLogger is the interface used to log
	sLogger SugaredLogger

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

// Ensure the default logger exists.  This is typically used for testing
func DefaultLogger() VerrazzanoLogger {
	return EnsureLogContext("default").EnsureLogger("default", zap.S(), zap.S())
}

// EnsureLogContext ensures that a LogContext exists
// The key must be unique for the process, for example a namespace/name combo.
func EnsureLogContext(key string) *LogContext {
	lock.Lock()
	defer lock.Unlock()
	log, ok := LogContextMap[key]
	if !ok {
		log = &LogContext{
			loggerMap: make(map[string]*verrazzanoLogger),
		}
		LogContextMap[key] = log
	}
	return log
}

// DeleteLogContext deletes the LogContext for the given key
func DeleteLogContext(key string) {
	lock.Lock()
	defer lock.Unlock()
	_, ok := LogContextMap[key]
	if ok {
		delete(LogContextMap, key)
	}
}

// EnsureLogger ensures that a new VerrazzanoLogger exists for the given key
func (c *LogContext) EnsureLogger(key string, sLogger SugaredLogger, zap *zap.SugaredLogger) VerrazzanoLogger {
	lock.Lock()
	defer lock.Unlock()
	log, ok := c.loggerMap[key]
	if !ok {
		log = &verrazzanoLogger{
			context:         c,
			frequencySecs:   60,
			historyMessages: make(map[string]bool),
		}
		c.loggerMap[key] = log
	}

	// Always replace the zap logger so that we get a clean set of
	// with clauses
	log.sLogger = sLogger
	log.zapLogger = zap

	return log
}

// Oncef formats a message and logs it once
func (v *verrazzanoLogger) Oncef(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	v.Once(s)
}

// Once logs a message once
func (v *verrazzanoLogger) Once(args ...interface{}) {
	v.doLog(true, args...)
}

// Progressf formats a message and logs it periodically
func (v *verrazzanoLogger) Progressf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	v.Progress(s)
}

// Progress logs a message periodically
func (v *verrazzanoLogger) Progress(args ...interface{}) {
	v.doLog(false, args...)
}

// doLog logs an info message either now or sometime in the future.  The message
// will be logged only if it is new or the next log time has been reached.
// This function allows a controller to constantly log info messages very frequently,
// such as "waiting for Verrazzano secret", but the message will only be logged
// once periodically according to the frequency (e.g. once every 60 seconds).
// If the log message is new or has changed then it is logged immediately.
func (v *verrazzanoLogger) doLog(once bool, args ...interface{}) {
	msg := fmt.Sprint(args...)
	now := time.Now()

	// If this message is in the history map, that means it has been
	// logged already, previous to the current message.  This happens
	// if a controller reconcile loop is called repeatedly.  In this
	// case we never want to display this message again, so just ignore it.
	_, ok := v.historyMessages[msg]
	if ok {
		return
	}
	// Log now if the message changed or wait time exceeded
	logNow := true

	if v.lastLog != nil {
		// If "once" is true or the message has changed, then save the old message
		// in the history so that it is never displayed again
		if once || msg != v.lastLog.msgLogged {
			v.historyMessages[v.lastLog.msgLogged] = true
		} else {
			// Check if it is time to log since the message didn't change
			waitSecs := time.Duration(v.frequencySecs) * time.Second
			nextLogTime := v.lastLog.lastLogTime.Add(waitSecs)
			logNow = now.Equal(nextLogTime) || now.After(nextLogTime)
		}
	}

	// Log the message if it is time and save the lastLog info
	if logNow {
		v.sLogger.Info(msg)
		v.lastLog = &lastLog{
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
	v.sLogger.Debug(args...)
}

// Debugf is a wrapper for SugaredLogger Debugf
func (v *verrazzanoLogger) Debugf(template string, args ...interface{}) {
	v.sLogger.Debugf(template, args...)
}

// Info is a wrapper for SugaredLogger Info
func (v *verrazzanoLogger) Info(args ...interface{}) {
	v.sLogger.Info(args...)
}

// Infof is a wrapper for SugaredLogger Infof
func (v *verrazzanoLogger) Infof(template string, args ...interface{}) {
	v.sLogger.Infof(template, args...)
}

// Error is a wrapper for SugaredLogger Error
func (v *verrazzanoLogger) Error(args ...interface{}) {
	v.sLogger.Error(args...)
}

// Errorf is a wrapper for SugaredLogger Errorf
func (v *verrazzanoLogger) Errorf(template string, args ...interface{}) {
	v.sLogger.Errorf(template, args...)
}

// SetZapLogger sets the zap logger
func (v *verrazzanoLogger) SetZapLogger(zap *zap.SugaredLogger) {
	v.zapLogger = zap
	v.sLogger = zap
}

// GetZapLogger gets the zap logger
func (v *verrazzanoLogger) GetZapLogger() *zap.SugaredLogger {
	return v.zapLogger
}

// GetContext gets the logger context
func (v *verrazzanoLogger) GetContext() *LogContext {
	return v.context
}
