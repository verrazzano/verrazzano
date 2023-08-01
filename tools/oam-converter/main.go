// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	app "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/app"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 3 {
		print("Not enough arguments. Arrange CLI arguments like this: inputDirectory outputDirectory namespace istioEnabled")
		return
	}

	switch inp := len(os.Args); inp {
	case 3:
		types.InputArgs.InputDirectory = os.Args[1]
		types.InputArgs.OutputDirectory = os.Args[2]
	case 4:
		types.InputArgs.InputDirectory = os.Args[1]
		types.InputArgs.OutputDirectory = os.Args[2]
		types.InputArgs.Namespace = os.Args[3]
	case 5:
		types.InputArgs.InputDirectory = os.Args[1]
		types.InputArgs.OutputDirectory = os.Args[2]
		types.InputArgs.Namespace = os.Args[3]
		types.InputArgs.IstioEnabled, _ = strconv.ParseBool(os.Args[4])
	default:
		print("Incorrect amount of arguments")
	}
	//Configure data from app and comp file to extract traits and workloads
	err := app.ConfData()
	if err != nil {
		print(err)
		return
	}

}
