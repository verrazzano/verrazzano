package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	VersionForInstall        = "versionForInstall"
	InterimVersionForUpgrade = "interimVersionForUpgrade"
)

func main() {
	//Parse command line arguments to extract workspace and derive versionType
	workspace, versionType := parseCliArguments()

	//Extract release tags from git tag command.
	releaseTags := getReleaseTags(workspace)

	if versionType == InterimVersionForUpgrade {
		interimRelease := getInterimRelease(releaseTags)
		fmt.Print(interimRelease)
	} else if versionType == VersionForInstall {
		installRelease := getInstallRelease(releaseTags)
		fmt.Print(installRelease)
	} else {
		fmt.Errorf("invalid command line argument for derive version type")
	}
}

func parseCliArguments() (string, string) {
	var workspace, versionType string

	if len(os.Args) > 2 {
		// Receive working directory as a command line argument.
		workspace = os.Args[1]
		// Receive version type such as interimVersionForUpgrade or versionForInstall argument
		versionType = os.Args[2]
	} else {
		fmt.Errorf("no command cline arguments were specified")
	}
	return workspace, versionType
}

func getReleaseTags(workspace string) []string {
	// Change the working directory to the verrazzano workspace
	err := os.Chdir(workspace)
	if err != nil {
		fmt.Errorf("unable to change the current working directory %v", err)
	}
	// Execute git tag command.
	cmd := exec.Command("git", "tag")
	out, err := cmd.Output()
	if err != nil {
		fmt.Errorf("unable to execute git tag command %v", err)
	}

	// Split the output by newline and store it in a slice
	gitTags := strings.Split(string(out), "\n")

	// Extract release tags from gitTags
	var releaseTags []string
	for _, tag := range gitTags {
		if strings.HasPrefix(tag, "v") && !strings.HasPrefix(tag, "v0") {
			releaseTags = append(releaseTags, tag)
		}
	}
	return releaseTags
}

func getInterimRelease(releaseTags []string) string {
	// Get the latest release tag
	latestReleaseTag := releaseTags[len(releaseTags)-1]
	releaseTags = releaseTags[:len(releaseTags)-1]

	//Split the string excluding prefix 'v' into major and minor version values
	latestReleaseTagSplit := strings.Split(strings.TrimPrefix(latestReleaseTag, "v"), ".")
	majorVersionValue := parseInt(latestReleaseTagSplit[0])
	minorInterimVersionValue := parseInt(latestReleaseTagSplit[1]) - 1

	// Handles the major release case, e.g. where the latest version is 2.0.0 and the previous version is 1.4.2
	if minorInterimVersionValue < 0 {
		minorInterimVersionValue = 0
		majorVersionValue = parseInt(latestReleaseTagSplit[0]) - 1
		for _, version := range releaseTags {
			versionSplit := strings.Split(strings.TrimPrefix(version, "v"), ".")
			if parseInt(versionSplit[0]) == majorVersionValue && parseInt(versionSplit[1]) > minorInterimVersionValue {
				minorInterimVersionValue = parseInt(versionSplit[1])
			}
		}
	}

	// Iterate over all releases to configure the latest patch release
	latestPatch := 0
	var interimRelease string
	for _, version := range releaseTags {
		versionSplit := strings.Split(strings.TrimPrefix(version, "v"), ".")
		if parseInt(versionSplit[0]) == majorVersionValue && parseInt(versionSplit[1]) == minorInterimVersionValue {
			if parseInt(versionSplit[2]) > latestPatch {
				latestPatch = parseInt(versionSplit[2])
			}
			interimRelease = fmt.Sprintf("%s.%d.%d", versionSplit[0], minorInterimVersionValue, latestPatch)
		}
	}

	// Return the interim release tag
	return fmt.Sprintf("v%s\n", interimRelease)
}

func getInstallRelease(releaseTags []string) string {
	// Get the latest release tag
	latestReleaseTag := releaseTags[len(releaseTags)-1]
	releaseTags = releaseTags[:len(releaseTags)-1]

	//Split the string excluding prefix 'v' into major and minor version values
	latestReleaseTagSplit := strings.Split(strings.TrimPrefix(latestReleaseTag, "v"), ".")
	majorVersionValue := parseInt(latestReleaseTagSplit[0])
	minorInstallVersionValue := parseInt(latestReleaseTagSplit[1]) - 2

	// Handles the major release case, e.g. where the latest version is 2.0.0 and the previous version is 1.4.2
	if minorInstallVersionValue < 0 {
		minorInstallVersionValue = 0
		majorVersionValue = parseInt(latestReleaseTagSplit[0]) - 1
		majorReleaseDecrementCount := 0

		//Handles the case where we have releases such as 2.0.0. 3.0.0, 4.0.0
		totalMinorReleaseCounter := 0
		for _, version := range releaseTags {
			versionSplit := strings.Split(strings.TrimPrefix(version, "v"), ".")
			if parseInt(versionSplit[0]) == majorVersionValue && parseInt(versionSplit[1]) != 0 {
				totalMinorReleaseCounter++
			}
		}
		if totalMinorReleaseCounter == 0 {
			majorVersionValue = majorVersionValue - totalMinorReleaseCounter - 1
			majorReleaseDecrementCount = parseInt(latestReleaseTagSplit[0]) - majorVersionValue
		}
		for _, version := range releaseTags {
			versionSplit := strings.Split(strings.TrimPrefix(version, "v"), ".")
			if parseInt(versionSplit[0]) == majorVersionValue && parseInt(versionSplit[1]) > minorInstallVersionValue {
				if majorReleaseDecrementCount < 2 {
					minorInstallVersionValue = parseInt(versionSplit[1]) - 1
				} else {
					minorInstallVersionValue = parseInt(versionSplit[1])
				}
			}
		}
	}

	// Iterate over all releases to configure the latest patch release
	latestPatch := 0
	var installRelease string
	for _, version := range releaseTags {
		versionSplit := strings.Split(strings.TrimPrefix(version, "v"), ".")
		if parseInt(versionSplit[0]) == majorVersionValue && parseInt(versionSplit[1]) == minorInstallVersionValue {
			if parseInt(versionSplit[2]) > latestPatch {
				latestPatch = parseInt(versionSplit[2])
			}
			installRelease = fmt.Sprintf("%s.%d.%d", versionSplit[0], minorInstallVersionValue, latestPatch)
		}
	}

	// Return the interim release tag
	return fmt.Sprintf("v%s\n", installRelease)
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		fmt.Errorf("unable to convert the given string to int %s, %v", s, err)
	}
	return n
}
