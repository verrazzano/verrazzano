// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"os"
	"strconv"
	"time"

	"github.com/onsi/ginkgo"
)

// LogLevel is the logging level used to control log output
type LogLevel int

const (
	// Error level designates error events that might still allow the application to continue running
	Error LogLevel = 1
	// Info level designates informational messages that highlight the progress of the application at coarse-grained level
	Info LogLevel = 4
	// Debug level designates fine-grained informational events that are most useful to debug an application
	Debug LogLevel = 7
)

// the global log level, i.e. the level of logging that was requested by
// the caller (by setting the LOG_LEVEL environment variable).
var globalLogLevel LogLevel

// Log prints out a log message in a standard format and filters out messages
// based on the global log level
func Log(level LogLevel, message string) {
	// if the global log level has not been set yet, then set it
	if globalLogLevel == 0 {
		level, _ := os.LookupEnv("LOG_LEVEL")
		switch level {
		case "ERROR":
			globalLogLevel = Error
		case "INFO":
			globalLogLevel = Info
		case "DEBUG":
			globalLogLevel = Debug
		default:
			globalLogLevel = Info
		}
	}

	// only print messages at or below the global log level
	if level <= globalLogLevel {
		var levelHeader string
		switch level {
		case Error:
			levelHeader = "[ERROR]"
		case Info:
			levelHeader = "[INFO]"
		case Debug:
			levelHeader = "[DEBUG]"
		default:
			ginkgo.Fail("Bad (non-existent) error level requested: " + strconv.Itoa(int(level)))
		}
		ginkgo.GinkgoWriter.Write([]byte(levelHeader + " " + time.Now().Format("2020-01-02 15:04:05.000000") + " " + message + "\n"))
	}
}
