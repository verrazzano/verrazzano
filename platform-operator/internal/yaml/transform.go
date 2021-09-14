package yaml

import "strings"

// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Expand a dot seperated string to a YAML hierachical string
func expand(dotStr string) string {

	const indent "    "
	b := strings.Builder{}
	segs, err := strings.Split(dotStr,".")
	for _,seg := range(segs) {

	}

	return ""
}
