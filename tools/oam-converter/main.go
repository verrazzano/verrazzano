// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	app "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/app"
)

func main() {
	err := app.ConfData()
	if err != nil {
		print(err)
		return
	}

}
