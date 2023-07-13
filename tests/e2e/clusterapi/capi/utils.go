// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import "os"

func checkAll(data []bool) bool {
	for _, item := range data {
		// return false if any item is false
		if !item {
			return false
		}
	}
	return true
}

func getEnvDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}
