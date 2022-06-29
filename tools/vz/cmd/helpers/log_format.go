// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import "fmt"

type LogFormat string

const (
	LogFormatSimple LogFormat = "simple"
	LogFormatJSON   LogFormat = "json"
)

// Implement the pflag.Value interface to support validating the logs format options

func (lf *LogFormat) String() string {
	return string(*lf)
}

// Type is only used in help text
func (lf *LogFormat) Type() string {
	return "format"
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (lf *LogFormat) Set(value string) error {
	switch value {
	case string(LogFormatJSON), string(LogFormatSimple):
		*lf = LogFormat(value)
		return nil
	default:
		return fmt.Errorf("allowed values are %q and %q", string(LogFormatSimple), string(LogFormatJSON))
	}
}
