// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package module

// The module package implements the handlers for the module controllers used by Verrazzano.
// The controllers themselves use the common module controller code in module-operator repo,
// see https://github.com/verrazzano/verrazzano-modules/tree/main/module-operator/controllers/module
// This is exact same controller code used by the module-operator, VPO just imports the package and
// creates the controllers using the module controller manager.
// See https://github.com/verrazzano/verrazzano/blob/master/platform-operator/internal/operatorinit/run_operator.go#L245
