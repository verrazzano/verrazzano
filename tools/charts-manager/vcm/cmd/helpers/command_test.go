// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	vcmtesthelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/helpers"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
)

// TestGetMandatoryStringFlagValueOrError tests the execution of GetMandatoryStringFlagValueOrError
// GIVEN a call to GetMandatoryStringFlagValueOrError with specific parameters to get non empty value of a flag
//
//	WHEN the input flag is not declared
//	THEN the execution results in an error.
//
//	WHEN the flag value is empty or nil
//	THEN the execution results in an error.
//
//	WHEN the flag value is non empty
//	THEN the execution returns the flag value.
func TestGetMandatoryStringFlagValueOrError(t *testing.T) {
	anyError := fmt.Errorf("")
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	tests := []struct {
		name          string
		flagName      string
		flagValue     string
		testFlagName  string
		testFlagValue string
		wantError     error
	}{
		{
			name:         "testGetMandatoryStringFlagValueOrErrorNoFlagsCommand",
			testFlagName: "testFlag",
			wantError:    anyError,
		},
		{
			name:         "testGetMandatoryStringFlagValueOrErrorNilFlagValue",
			flagName:     "testFlag",
			testFlagName: "testFlag",
			wantError:    fmt.Errorf(ErrFormatMustSpecifyFlag, "testFlag", "testFlag", ""),
		},
		{
			name:         "testGetMandatoryStringFlagValueOrErrorEmptyFlagValue",
			flagName:     "testFlag",
			flagValue:    "\t\t\n",
			testFlagName: "testFlag",
			wantError:    fmt.Errorf(ErrFormatNotEmpty, "testFlag"),
		},
		{
			name:          "testGetMandatoryStringFlagValueOrErrorValidFlag",
			flagName:      "testFlag",
			flagValue:     "value",
			testFlagName:  "testFlag",
			testFlagValue: "value",
			wantError:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdhelpers.NewCommand(rc, "test", "", "")
			if tt.flagName != "" {
				cmd.PersistentFlags().StringP(tt.flagName, "", "", "")
			}

			if tt.flagValue != "" {
				cmd.PersistentFlags().Set(tt.flagName, tt.flagValue)
			}

			value, err := GetMandatoryStringFlagValueOrError(cmd, tt.testFlagName, "")
			if err != nil && tt.wantError == nil {
				t.Errorf("unexpected error %v", err)
			}

			if err != nil && tt.wantError != nil && tt.wantError != anyError && err.Error() != tt.wantError.Error() {
				t.Errorf("error %v, expected %v", err, tt.wantError)
			}

			if err == nil && tt.wantError != nil {
				t.Errorf("resulted in no error, expected %v", tt.wantError)
			}

			if tt.wantError == anyError {
				assert.Error(t, err)
			}

			assert.Equal(t, tt.testFlagValue, value)
		})
	}
}
