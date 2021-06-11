// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

// resolveNamesapce will return the default verrzzano system namespace unless the namespace
// is specified
func resolveVerrazzanoNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return vzDefaultNamespace
}
