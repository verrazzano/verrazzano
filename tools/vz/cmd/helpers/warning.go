// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"github.com/mattn/go-isatty"
	"io"
	"k8s.io/client-go/rest"
	"os"
	"runtime"
)

// getWarningHandler returns an implementation of WarningHandler that outputs code 299 warnings to the specified writer.
func getWarningHandler(w io.Writer) rest.WarningHandler {
	// deduplicate and attempt color warnings when running from a terminal
	return rest.NewWarningWriter(w, rest.WarningWriterOptions{
		Deduplicate: true,
		Color:       allowsColorOutput(w),
	})
}

// allowsColorOutput returns true if the specified writer is a terminal and
// the process environment indicates color output is supported and desired.
// Copied from k8s.io/kubectl/pkg/util/term.AllowsColorOutput.
func allowsColorOutput(w io.Writer) bool {
	if !isTerminal(w) {
		return false
	}

	// https://en.wikipedia.org/wiki/Computer_terminal#Dumb_terminals
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	// https://no-color.org/
	if _, nocolor := os.LookupEnv("NO_COLOR"); nocolor {
		return false
	}

	// On Windows WT_SESSION is set by the modern terminal component.
	// Older terminals have poor support for UTF-8, VT escape codes, etc.
	if runtime.GOOS == "windows" && os.Getenv("WT_SESSION") == "" {
		return false
	}

	return true
}

func isTerminal(w io.Writer) bool {
	if w, ok := w.(*os.File); ok && isatty.IsTerminal(w.Fd()) {
		return true
	}
	return false
}
