// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import "fmt"

type LogsFormat string

const (
	LogsFormatPretty LogsFormat = "pretty"
	LogsFormatJSON   LogsFormat = "json"
)

// Implement the pflag.Value interface to support validating the logs format options

func (lf *LogsFormat) String() string {
	return string(*lf)
}

// Type is only used in help text
func (lf *LogsFormat) Type() string {
	return "string"
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (lf *LogsFormat) Set(value string) error {
	switch value {
	case string(LogsFormatJSON), string(LogsFormatPretty):
		*lf = LogsFormat(value)
		return nil
	default:
		return fmt.Errorf("allowed values are %q and %q", string(LogsFormatPretty), string(LogsFormatJSON))
	}
}
