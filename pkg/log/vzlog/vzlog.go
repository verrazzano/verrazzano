// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzlog

import (
	"errors"
	"fmt"
	"sync"
	"time"

	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"

	"go.uber.org/zap"
)

// ResourceConfig is the configuration of a logger for a resource that is being reconciled
type ResourceConfig struct {
	// Name is the name of the resource
	Name string

	// Namespace is the namespace of the resource
	Namespace string

	// ID is the resource uid
	ID string

	// Generation is the resource generation
	Generation int64

	// Controller name is the name of the controller
	ControllerName string
}

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

	// Debugw formats a message and logs it once at Debug log level
	Debugw(msg string, args ...interface{})

	// Info logs a message at Info log level
	Info(args ...interface{})

	// Infof formats a message and logs it once at Info log level
	Infof(template string, args ...interface{})

	// Infow formats a message and logs it once at Info log level
	Infow(msg string, keysAndValues ...interface{})

	// Error logs a message at Error log level
	Error(args ...interface{})

	// Errorf formats a message and logs it once at Error log level
	Errorf(template string, args ...interface{})

	// Errorw formats a message and logs it once at Error log level
	Errorw(msg string, keysAndValues ...interface{})
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

	// ErrorNewErr logs and error, then returns the error
	ErrorNewErr(args ...interface{}) error

	// ErrorfNewErr formats an error, logs it, then returns the formatted error
	ErrorfNewErr(template string, args ...interface{}) error

	// ErrorfThrottledNewErr Records a formatted error message throttled at the ProgressLogger frequency then returns the formatted error
	ErrorfThrottledNewErr(template string, args ...interface{}) error

	// ErrorfThrottled Records a formatted error message throttled at the ProgressLogger frequency
	ErrorfThrottled(template string, args ...interface{})

	// SetFrequency sets the logging frequency of a progress message
	SetFrequency(secs int) VerrazzanoLogger
}

// VerrazzanoLogger is a logger interface that provides sugared and progress logging
type VerrazzanoLogger interface {
	SugaredLogger
	ProgressLogger

	// SetZapLogger sets the zap logger
	SetZapLogger(zap *zap.SugaredLogger)

	// GetZapLogger gets the zap logger
	GetZapLogger() *zap.SugaredLogger

	// GetRootZapLogger gets the root zap logger
	GetRootZapLogger() *zap.SugaredLogger

	// GetContext gets the log context
	GetContext() *LogContext
}

// LogContext is the log context for a given resource.
// This logger can be used to manage logging for the resource and sub-resources, such as components
type LogContext struct {
	// loggerMap contains a map of verrazzanoLogger objects
	loggerMap map[string]*verrazzanoLogger

	// Generation is the generation of the resource being logged
	Generation int64

	// RootZapLogger is the zap SugaredLogger for the resource. Component loggers are derived from this.
	RootZapLogger *zap.SugaredLogger
}

// verrazzanoLogger implements the VerrazzanoLogger interface
// Notice that the SugaredLogger methods (Debug, Info, Error) have another level
// of indirection (Debug2).  This is because it has to match the 2 levels of call stack
// used by Progrss and Once, with both call doLog.  We setup the logger to skip 2 calls
// in the stack frame when logging, so the caller file/line is displayed.  Otherwise, you
// would see "vzlog.go/line xyz"
type verrazzanoLogger struct {
	// context is the LogContext
	context *LogContext

	// zapLogger is the zap SugaredLogger
	zapLogger *zap.SugaredLogger

	// sLogger is the interface used to log
	sLogger SugaredLogger

	// frequency between logs in seconds
	frequencySecs int

	// trashMessages is a set of log messages that can never be displayed again
	trashMessages map[string]bool

	// progressHistory is a map of progress log messages that are displayed periodically
	progressHistory map[string]*logEvent
}

// lastLog tracks the last message logged for progress messages
type logEvent struct {
	// logTime is the last time the message was logged
	logTime *time.Time

	// msgLogged is the message that was logged
	msgLogged string
}

// DefaultLogger ensures the default logger exists.  This is typically used for testing
func DefaultLogger() VerrazzanoLogger {
	return EnsureContext("default").EnsureLogger("default", zap.S(), zap.S())
}

// EnsureResourceLogger ensures that a logger exists for a specific generation of a Kubernetes resource.
// When a resource is getting reconciled, the status may frequently get updated during
// the reconciliation.  This is the case for the Verrazzano resource.  As a result,
// the controller-runtime queue gets filled with updated instances of a resource that
// have the same generation. The side-effect is that after a resource is completely reconciled,
// the controller Reconcile method may still be called many times. In this case, the existing
// context must be used so that 'once' and 'progress' messages don't start from a new context,
// causing them to be displayed when they shouldn't.  This mehod ensures that the same
// logger is used for a given resource and generation.
func EnsureResourceLogger(config *ResourceConfig) (VerrazzanoLogger, error) {
	// Build a logger, skipping 2 call frames so that the correct caller file/line is displayed in the log.
	// If the callerSkip was 0, then you would see the vzlog.go/line# instead of the file/line of the caller
	// that called the VerrazzanoLogger
	zaplog, err := vzlogInit.BuildZapInfoLogger(2)
	if err != nil {
		// This is a fatal error which should never happen
		return nil, errors.New("Failed initializing logger for controller")
	}
	return ForZapLogger(config, zaplog), nil
}

func ForZapLogger(config *ResourceConfig, zaplog *zap.SugaredLogger) VerrazzanoLogger {
	// Ensure a Verrazzano logger exists, using zap SugaredLogger as the underlying logger.
	zaplog = zaplog.With(vzlogInit.FieldResourceNamespace, config.Namespace, vzlogInit.FieldResourceName,
		config.Name, vzlogInit.FieldController, config.ControllerName)

	// Get a log context.  If the generation doesn't match then delete it and
	// create a new one.  This will ensure we have a new context for a new
	// generation of a resource
	context := EnsureContext(config.ID)
	if context.Generation != 0 && context.Generation != config.Generation {
		DeleteLogContext(config.ID)
		context = EnsureContext(config.ID)
	}
	context.Generation = config.Generation
	context.RootZapLogger = zaplog

	// Finally, get the logger using this context.
	logger := context.EnsureLogger("default", zaplog, zaplog)
	return logger
}

// EnsureContext ensures that a LogContext exists
// The key must be unique for the process, for example a namespace/name combo.
func EnsureContext(key string) *LogContext {
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
			trashMessages:   make(map[string]bool),
			progressHistory: make(map[string]*logEvent),
		}
		c.loggerMap[key] = log
	}

	// Always replace the zap logger so that we get a clean set of
	// with clauses
	log.sLogger = sLogger
	log.zapLogger = zap
	if log.context.RootZapLogger == nil {
		log.context.RootZapLogger = zap
	}

	return log
}

// Oncef formats a message and logs it once
func (v *verrazzanoLogger) Oncef(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	v.doLog(true, s)
}

// Once logs a message once
func (v *verrazzanoLogger) Once(args ...interface{}) {
	v.doLog(true, args...)
}

// Progressf formats a message and logs it periodically
func (v *verrazzanoLogger) Progressf(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	v.doLog(false, s)
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
	cacheUpdated := v.shouldLogMessage(once, msg)
	if cacheUpdated {
		v.sLogger.Info(msg)
	}
}

// doError Logs an error message first checking against the log cache; same behavior as doLog() except that messages
// are recorded as errors at the throttling frequency.  Errors are never once-only.
func (v *verrazzanoLogger) doError(args ...interface{}) {
	msg := fmt.Sprint(args...)
	doLog := v.shouldLogMessage(false, msg)
	if doLog {
		v.sLogger.Error(msg)
	}
}

// shouldLogMessage Checks candidate log message against the cache and returns true if that message should be recorded in the log.
//
// A message should be recorded when
// - A message is newly added to the cache (seen for the first time)
// - A message is throttled, but it has not exceeded its frequency threshold since the last occurrence
func (v *verrazzanoLogger) shouldLogMessage(once bool, msg string) bool {
	lock.Lock()
	defer lock.Unlock()
	// If the message is in the trash, that means it should never be logged again.
	_, ok := v.trashMessages[msg]
	if ok {
		return false
	}

	// If this is log "once", log it and save in trash so it is never logged again, then return
	if once {
		v.trashMessages[msg] = true
		return true
	}

	// If this message has already been logged, then check if time to log again
	now := time.Now()
	history := v.progressHistory[msg]
	if history != nil {
		waitSecs := time.Duration(v.frequencySecs) * time.Second
		nextLogTime := history.logTime.Add(waitSecs)

		// Log now if the message wait time exceeded
		if now.Equal(nextLogTime) || now.After(nextLogTime) {
			history.logTime = &now
			return true
		}
	} else {
		// This is a new message log it
		v.progressHistory[msg] = &logEvent{
			logTime:   &now,
			msgLogged: msg,
		}
		return true
	}
	return false
}

// SetFrequency sets the log frequency
func (v *verrazzanoLogger) SetFrequency(secs int) VerrazzanoLogger {
	v.frequencySecs = secs
	return v
}

// SetZapLogger sets the zap logger
func (v *verrazzanoLogger) SetZapLogger(zap *zap.SugaredLogger) {
	v.zapLogger = zap
	v.sLogger = zap
}

// GetZapLogger zap logger gets a clone of the zap logger
func (v *verrazzanoLogger) GetZapLogger() *zap.SugaredLogger {
	return v.zapLogger
}

// GetRootZapLogger gets the root zap logger at the context level
func (v *verrazzanoLogger) GetRootZapLogger() *zap.SugaredLogger {
	return v.context.RootZapLogger
}

// EnsureContext gets the logger context
func (v *verrazzanoLogger) GetContext() *LogContext {
	return v.context
}

// ErrorNewErr logs an error, then returns it.
func (v *verrazzanoLogger) ErrorNewErr(args ...interface{}) error {
	s := fmt.Sprint(args...)
	err := errors.New(s)
	v.Error2(err)
	return err
}

// ErrorfNewErr formats an error, logs it, then returns it.
func (v *verrazzanoLogger) ErrorfNewErr(template string, args ...interface{}) error {
	err := fmt.Errorf(template, args...)
	v.Error2(err)
	return err
}

func (v *verrazzanoLogger) ErrorfThrottledNewErr(template string, args ...interface{}) error {
	err := fmt.Errorf(template, args...)
	v.doError(err.Error())
	return err
}

func (v *verrazzanoLogger) ErrorfThrottled(template string, args ...interface{}) {
	s := fmt.Sprintf(template, args...)
	v.doError(s)
}

// Debug is a wrapper for SugaredLogger Debug
func (v *verrazzanoLogger) Debug(args ...interface{}) {
	v.Debug2(args...)
}

// Debugf is a wrapper for SugaredLogger Debugf
func (v *verrazzanoLogger) Debugf(template string, args ...interface{}) {
	v.Debugf2(template, args...)
}

// Debugw is a wrapper for SugaredLogger Debugw
func (v *verrazzanoLogger) Debugw(msg string, keysAndValues ...interface{}) {
	v.Debugw2(msg, keysAndValues...)
}

// Info is a wrapper for SugaredLogger Info
func (v *verrazzanoLogger) Info(args ...interface{}) {
	v.Info2(args...)
}

// Infof is a wrapper for SugaredLogger Infof
func (v *verrazzanoLogger) Infof(template string, args ...interface{}) {
	v.Infof2(template, args...)
}

// Infow is a wrapper for SugaredLogger Infow
func (v *verrazzanoLogger) Infow(msg string, keysAndValues ...interface{}) {
	v.Infow2(msg, keysAndValues...)
}

// Error is a wrapper for SugaredLogger Error
func (v *verrazzanoLogger) Error(args ...interface{}) {
	v.Error2(args...)
}

// Errorf is a wrapper for SugaredLogger Errorf
func (v *verrazzanoLogger) Errorf(template string, args ...interface{}) {
	v.Errorf2(template, args...)
}

// Errorw is a wrapper for SugaredLogger Errorw
func (v *verrazzanoLogger) Errorw(msg string, keysAndValues ...interface{}) {
	v.Errorw2(msg, keysAndValues...)
}

// Debug is a wrapper for SugaredLogger Debug
func (v *verrazzanoLogger) Debug2(args ...interface{}) {
	v.sLogger.Debug(args...)
}

// Debugf is a wrapper for SugaredLogger Debugf
func (v *verrazzanoLogger) Debugf2(template string, args ...interface{}) {
	v.sLogger.Debugf(template, args...)
}

// Debugw2 is a wrapper for SugaredLogger Debugw
func (v *verrazzanoLogger) Debugw2(msg string, keysAndValues ...interface{}) {
	v.sLogger.Debugw(msg, keysAndValues...)
}

// Info is a wrapper for SugaredLogger Info
func (v *verrazzanoLogger) Info2(args ...interface{}) {
	v.sLogger.Info(args...)
}

// Infof is a wrapper for SugaredLogger Infof
func (v *verrazzanoLogger) Infof2(template string, args ...interface{}) {
	v.sLogger.Infof(template, args...)
}

// Infow2 is a wrapper for SugaredLogger Infow
func (v *verrazzanoLogger) Infow2(msg string, keysAndValues ...interface{}) {
	v.sLogger.Infow(msg, keysAndValues...)
}

// Error is a wrapper for SugaredLogger Error
func (v *verrazzanoLogger) Error2(args ...interface{}) {
	v.sLogger.Error(args...)
}

// Errorf is a wrapper for SugaredLogger Errorf
func (v *verrazzanoLogger) Errorf2(template string, args ...interface{}) {
	v.sLogger.Errorf(template, args...)
}

// Errorw2 is a wrapper for SugaredLogger Errorw
func (v *verrazzanoLogger) Errorw2(msg string, keysAndValues ...interface{}) {
	v.sLogger.Errorw(msg, keysAndValues...)
}
