// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"github.com/mattn/go-isatty"
	"io"
	"k8s.io/client-go/rest"
	"os"
)

// getWarningHandler returns an implementation of WarningHandler that outputs code 299 warnings to the specified writer.
func getWarningHandler(w io.Writer) rest.WarningHandler {
	// deduplicate and attempt color warnings when running from a terminal
	return rest.NewWarningWriter(w, rest.WarningWriterOptions{
		Deduplicate: true,
		Color:       isTerminal(w),
	})
}

func isTerminal(w io.Writer) bool {
	if w, ok := w.(*os.File); ok && isatty.IsTerminal(w.Fd()) {
		return true
	}
	return false
}
