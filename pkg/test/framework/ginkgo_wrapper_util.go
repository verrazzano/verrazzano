// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import "reflect"

// isBodyFunc - return boolean indicating if the interface is a function
func isBodyFunc(body interface{}) bool {
	bodyType := reflect.TypeOf(body)
	return bodyType.Kind() == reflect.Func
}
