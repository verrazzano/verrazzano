// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package internal

import . "github.com/onsi/gomega" //nolint

type TestStruct struct {
}

func (t *TestStruct) ReceiverThatCallsExpect() error {
	Expect(false).To(BeTrue())
	return nil
}
