// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// displayInstallationProgress Checks state of components until all of them are Ready or specified timeout is reached
func displayInstallationProgress(cmd *cobra.Command, vzHelper helpers.VZHelper, duration time.Duration) error {
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		fmt.Println("Error in client", err)
		return err
	}
	time.Sleep(time.Second * 10)
	fmt.Print("\033[2J\033[H") // Clear the console before entering the loop
	startTime := time.Now()
	for {
		// Save the cursor position
		fmt.Print("\033[s")
		statusMap, err := getEnabledComponentMap(client)
		if err != nil {
			fmt.Println("Error in fetching verrazzano resource", err)
			return err
		}
		if len(statusMap) == 0 {
			return fmt.Errorf("No enabled components found")
		}
		displayComponentStatus(statusMap)
		progress, nonReady := calculateProgress(statusMap)
		displayProgressBar(progress)
		if isAllReady(statusMap) {
			fmt.Println("\nInstallation Completed. All components are ready")
			break
		} else {
			if time.Since(startTime) >= duration {

				return fmt.Errorf("timed out waiting for components to be ready-> %s", strings.TrimPrefix(strings.Join(nonReady, ", "), ", "))
			}
			time.Sleep(constants.RefreshRate)
			// Restore the cursor position to the beginning of the table and the progress bar
			fmt.Print("\033[u")
		}
	}
	return nil
}

// displayComponentStatus displays the component status in the console.
// Parameter:- statusMap: A map containing the component names and their status
func displayComponentStatus(statusMap map[string]v1alpha1.CompStateType) {
	// Move the cursor to the top-left position and clear the console
	fmt.Fprintf(os.Stdout, "\033[H\033[J")
	fmt.Println("Component\t\t\t\t\tStatus")
	fmt.Println("-------------------------------------------------------")
	// Create a slice to hold the component names for sorting
	var componentNames []string
	for componentName := range statusMap {
		componentNames = append(componentNames, componentName)
	}
	sort.Strings(componentNames)
	// Display the components in alphabetical order
	for _, componentName := range componentNames {
		status := statusMap[componentName]
		// Move the cursor to the beginning of the current line and clear the line
		fmt.Fprintf(os.Stdout, "\r\033[K")
		fmt.Printf("%-40s\t%s\n", componentName, status)
	}
}

// isAllReady checks if all components are in "Ready" state.
// Returns: bool: True if all components are in "Ready" state, false otherwise
func isAllReady(statusMap map[string]v1alpha1.CompStateType) bool {
	for _, status := range statusMap {
		if status != "Ready" {
			return false
		}
	}
	return true
}

// getEnabledComponentMap retrieves the enabled components and their status.
func getEnabledComponentMap(client client.Client) (map[string]v1alpha1.CompStateType, error) {
	vz, err := helpers.FindVerrazzanoResource(client)
	if err != nil {
		return nil, err
	}
	statusMap := make(map[string]v1alpha1.CompStateType)
	for componentName, componentStatus := range vz.Status.Components {
		if componentStatus.State != "Disabled" {
			statusMap[componentName] = v1alpha1.CompStateType(componentStatus.State)
		}
	}
	return statusMap, nil
}

// calculateProgress calculates the installation progress based on the number of components in "Ready" state.
func calculateProgress(statusMap map[string]v1alpha1.CompStateType) (float64, []string) {
	sync := make([]string, 1)
	readyCount := 0
	totalCount := len(statusMap)
	for index, status := range statusMap {
		if status == "Ready" {
			readyCount++
		} else {
			sync = append(sync, index)
		}
	}
	return float64(readyCount) / float64(totalCount), sync
}

// displayProgressBar displays the progress bar based on the installation progress.
// Parameter:- progress: The installation progress percentage
func displayProgressBar(progress float64) {
	fmt.Print("[")
	slides := int(progress * float64(constants.TotalWidth))
	for i := 0; i < constants.TotalWidth; i++ {
		if i < slides {
			// Set color to green
			fmt.Print("\033[32m#")
		} else {
			fmt.Print(" ")
		}
	}
	// Reset color back to the default
	fmt.Print("\033[0m")
	fmt.Printf("] %.0f%%\r", progress*100)
}
