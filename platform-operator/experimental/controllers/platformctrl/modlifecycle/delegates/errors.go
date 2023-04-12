// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package delegates

import "fmt"

type notReadyError struct {
	message string
}

func (n notReadyError) Error() string {
	return n.message
}

func NotReadyErrorf(message string, a ...interface{}) error {
	return notReadyError{message: fmt.Sprintf(message, a...)}
}

func IsNotReadyError(err error) bool {
	_, ok := err.(notReadyError)
	return ok
}
