// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	ErrFormatMustSpecifyFlag = "must specify %s using --%s or -%s"
	ErrFormatNotEmpty        = "%s can not be empty"
)

func GetMandatoryStringFlagValueOrError(cmd *cobra.Command, flagName string, flagShorthand string) (string, error) {
	flagValue, err := cmd.PersistentFlags().GetString(flagName)
	if err != nil {
		return "", err
	}

	if flagValue == "" {
		return "", fmt.Errorf(ErrFormatMustSpecifyFlag, flagName, flagName, flagShorthand)
	}

	if len(strings.TrimSpace(flagValue)) == 0 {
		return "", fmt.Errorf(ErrFormatNotEmpty, flagName)
	}

	return flagValue, nil
}
