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
	input := types.ConversionInput{}
	switch inp := len(os.Args); inp {
	case 3:
		input.InputDirectory = os.Args[1]
		input.OutputDirectory = os.Args[2]
	case 4:
		input.InputDirectory = os.Args[1]
		input.OutputDirectory = os.Args[2]
		input.Namespace = os.Args[3]
	case 5:
		input.InputDirectory = os.Args[1]
		input.OutputDirectory = os.Args[2]
		input.Namespace = os.Args[3]
		input.IstioEnabled, _ = strconv.ParseBool(os.Args[4])
	default:
		print("Incorrect amount of arguments")
	}
	//Configure data from app and comp file to extract traits and workloads
	err := app.ConfData(input)
	if err != nil {
		print(err)
		return
	}

}
