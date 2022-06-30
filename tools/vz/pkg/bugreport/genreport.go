// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

var bugReportDir string

func GenerateBugReport(kubeClient kubernetes.Interface, client clipkg.Client, bugReportFile string, vzHelper helpers.VZHelper) error {
	tmpDir, err := ioutil.TempDir("", constants.BugReportDir)
	if err != nil {
		return fmt.Errorf("an error creating the temporary: %s, to place cluster resources")
	}
	defer os.RemoveAll(tmpDir)

	bugReportDir = tmpDir + string(os.PathSeparator) + constants.BugReportDir
	err = os.Mkdir(bugReportDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("an error creating the directory: %s", bugReportDir)
	}

	err = CaptureInstallIssues(client, kubeClient, bugReportDir)
	if err != nil {
		return fmt.Errorf("there is an error with capturing the resources: %s", err.Error())
	}

	// TODO: Redact sensitive information from all the files in bugReportDir

	// Create the report file
	err = CreateReportArchive(bugReportDir, bugReportFile)
	if err != nil {
		return fmt.Errorf("there is an error in creating the bug report: %s", err.Error())
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Successfully generated the bug report %s\n", bugReportFile))

	// Display a warning message to review the contents of the report
	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("WARNING: Please examine the contents of the bug report for sensitive data.\n"))
	return nil
}
