// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import "github.com/verrazzano/verrazzano/pkg/test/framework/internal"

func Require(requires ...string) Requires {
	return Requires(requires)
}

type Requires = internal.Requires
