// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

// Usecase specifies a PSR usecase that does a single worker task running in a pod
type Usecase struct {
	UsecasePath  string
	OverrideFile string
	Description  string
}

// Scenario specifies a PSR scenario which consists of multiple use cases
type Scenario struct {
	Name        string
	ID          string
	Description string
	Usecases    []Usecase
}
