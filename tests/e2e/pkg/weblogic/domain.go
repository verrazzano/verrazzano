// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

// Domain describes a WebLogic domain CR
type Domain struct {
	Status struct {
		Servers []struct {
			DesiredState string `json:"desiredState"`
			Health       struct {
				OverallHealth string `json:"overallHealth"`
			} `json:"health"`
		} `json:"servers"`
	} `json:"status"`
}
