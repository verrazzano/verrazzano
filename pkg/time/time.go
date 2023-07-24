// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package time

import (
	"fmt"
	"time"
)

const timeFormat = "%d-%02d-%02dT%02d:%02d:%02dZ"

func GetCurrentTime() string {
	t := time.Now().UTC()
	return fmt.Sprintf(timeFormat,
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func ParseTime(timeString string) (time.Time, error) {
	t,err := time.Parse(timeFormat, timeString)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

