// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

// EnsureCluster will check that there is either a cluster already configured (by someone outside, the person/process
// who is running this test), or if not, will create a local Kind cluster at the specified version for the test to use
func EnsureCluster(version string) {

}

// EnsureVerrazzanoInstalled will install the requested version of Verrazzano
func EnsureVerrazzanoInstalled(version string) {

}

// EnsureVerrazzanoVersion will check that the version of Verrazzano reported by the Verrazzano CR matches the argument
func EnsureVerrazzanoVersion(version string) {

}

// UpgradeVerrazzanoToRelease will upgrade the VPO to the specified release, and then initiate an upgrade of Verrazzano
// itself to that release
func UpgradeVerrazzanoToRelease(version string) {

}