// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package progress

import (
	"fmt"
	"go.uber.org/zap"
	"time"
)

// RootLoggerMap contains a map of RootLogger objects
var RootLoggerMap = make(map[string]*RootLogger)

var fakeInfoLogger infoLogger

type infoLogger interface {
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Progress(args ...interface{})
	Progressf(template string, args ...interface{})
}

// RootLogger is the root logger for a given resource.
// This logger can be used to manage logging for the resource and sub-resources, such as components
type RootLogger struct {
	// zapLog is the logger used to log messages
	zapLogger *zap.SugaredLogger

	// pregressLoggerMap contains a map of ProgressLogger objects
	progressLoggerMap map[string]*ProgressLogger
}

// ProgressLogger logs a message periodically
type ProgressLogger struct {
	// zapLog is the logger used to log messages
	rootLogger *RootLogger

	// frequency between logs in seconds
	frequencySecs int

	// history is a set of log messages for this ProgressLogger
	historyMessages map[string]bool

	// lastLog keeps track of the last logged message
	*lastLog
}

// lastLog contains the history messages logged
type lastLog struct {
	// Last time message was logged
	lastLogTime *time.Time

	// Message logged
	msgLogged string
}

// EnsureRootLogger ensures that a RootLogger exists
// The key must be unique for the process, for example a namespace/name combo.
func EnsureRootLogger(key string, zapLog *zap.SugaredLogger) *RootLogger {
	log, ok := RootLoggerMap[key]
	if !ok {
		log = &RootLogger{
			zapLogger:         zapLog,
			progressLoggerMap: make(map[string]*ProgressLogger),
		}
		RootLoggerMap[key] = log
	}
	return log
}

// DeleteRootLogger deletes the ResouceLogger for the given key
func DeleteRootLogger(key string) {
	_, ok := RootLoggerMap[key]
	if ok {
		delete(RootLoggerMap, key)
	}
}

// DefaultProgressLogger ensures that a new default ProgressLogger exists
func (r *RootLogger) DefaultProgressLogger() *ProgressLogger {
	return r.EnsureProgressLogger("default")
}

// EnsureProgressLogger ensures that a new ProgressLogger exists for the given key
func (r *RootLogger) EnsureProgressLogger(key string) *ProgressLogger {
	log, ok := r.progressLoggerMap[key]
	if !ok {
		log = &ProgressLogger{
			rootLogger:      r,
			frequencySecs:   60,
			historyMessages: make(map[string]bool),
		}
		r.progressLoggerMap[key] = log
	}
	return log
}

// Infof formats a message and logs it
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
		if fakeInfoLogger == nil {
			p.rootLogger.zapLogger.Info(msg)
		} else {
			fakeInfoLogger.Info(msg)
		}
		p.lastLog = &lastLog{
			lastLogTime: &now,
			msgLogged:   msg,
		}
	}
}

// Info is a wrapper for SugaredLogger Info
func (p *ProgressLogger) Info(args ...interface{}) {
	p.rootLogger.zapLogger.Info(args...)
}

// Infof is a wrapper for SugaredLogger Infof
func (p *ProgressLogger) Infof(template string, args ...interface{}) {
	p.rootLogger.zapLogger.Infof(template, args...)
}

// Error is a wrapper for SugaredLogger Error
func (p *ProgressLogger) Error(args ...interface{}) {
	p.rootLogger.zapLogger.Error(args...)
}

// Errorf is a wrapper for SugaredLogger Errorf
func (p *ProgressLogger) Errorf(template string, args ...interface{}) {
	p.rootLogger.zapLogger.Errorf(template, args...)
}

// Set the log frequency
func (p ProgressLogger) SetFrequency(secs int) ProgressLogger {
	p.frequencySecs = secs
	return p
}

func setFakeLogger(logger infoLogger) {
	fakeInfoLogger = logger
}
