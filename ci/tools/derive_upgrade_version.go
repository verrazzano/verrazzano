// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"flag"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	VersionForInstall             = "install-version"
	InterimVersionForUpgrade      = "interim-version"
	LatestVersionForCurrentBranch = "latest-version-for-branch"
	VersionsGTE                   = "versions-gte"
)

var (
	workspace, versionType, developmentVersion string
	excludeReleaseTags                         []string
)

func main() {

	//Parse command line arguments to extract params
	help := false
	flag.BoolVar(&help, "help", false, "Display usage help")
	flag.Parse()
	if help {
		printUsage()
		os.Exit(0)
	}
	parseCliArgs(flag.Args())

	//Extract release tags from git tag command.
	releaseTags := getReleaseTags(workspace, excludeReleaseTags)
	switch versionType {
	case InterimVersionForUpgrade:
		interimRelease := getInterimRelease(releaseTags)
		fmt.Print(interimRelease)
	case VersionForInstall:
		installRelease := getInstallRelease(releaseTags)
		fmt.Print(installRelease)
	case LatestVersionForCurrentBranch:
		latestRelease := getLatestReleaseForCurrentBranch(releaseTags)
		fmt.Println(latestRelease)
	case VersionsGTE:
		tagsAfter, err := getTagsGTE(releaseTags, excludeReleaseTags[0])
		if err != nil {
			panic(err)
		}
		fmt.Println(tagsAfter)
	default:
		fmt.Printf("invalid command line argument for derive version type \n")
		os.Exit(1)
	}
}

func parseCliArgs(args []string) {

	if len(args) < 1 {
		fmt.Printf("\nno command line arguments were specified\n")
		printUsage()
		os.Exit(1)
	}

	if len(args) > 0 {
		// Receive working directory as a command line argument.
		workspace = args[0]
		// Receive version type such as interimVersionForUpgrade or versionForInstall argument
		versionType = args[1]
	} else {
		fmt.Printf("no worspace path and version type line arguments were specified\n")
		os.Exit(1)
	}

	if len(args) > 2 {
		for index, arg := range args {
			// Remove any ',' or ']' suffixes and remove any '[' prefix
			trimArg := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSuffix(arg, ","), "]"), "[")
			if index > 1 {
				if versionType == LatestVersionForCurrentBranch {
					developmentVersion = trimArg
					return
				}
				excludeReleaseTags = append(excludeReleaseTags, trimArg)
			}
		}
	}
}

func getReleaseTags(workspace string, excludeReleaseTags []string) []string {
	// Change the working directory to the verrazzano workspace
	err := os.Chdir(workspace)
	if err != nil {
		fmt.Printf("\nunable to change the current working directory %v", err.Error())
	}
	// Execute git tag command.
	cmd := exec.Command("git", "tag")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("\nunable to execute git tag command %v", err.Error())
	}

	// Split the output by newline and store it in a slice
	gitTags := strings.Split(string(out), "\n")

	// Extract release tags from gitTags
	var releaseTags []string

	for _, tag := range gitTags {
		if strings.HasPrefix(tag, "v") && !strings.HasPrefix(tag, "v0") {
			// Exclude the release tags if tag exists in excludeReleaseTags
			if !DoesTagExistsInExcludeList(tag, excludeReleaseTags) {
				releaseTags = append(releaseTags, tag)
			}
		}
	}
	return releaseTags
}

// DoesTagExistsInExcludeList returns true if the tag exists in excludeReleasetag
func DoesTagExistsInExcludeList(releaseTag string, excludeReleaseTags []string) bool {
	for _, excludeTag := range excludeReleaseTags {
		if excludeTag == releaseTag {
			return true
		}
	}
	return false
}

func getLatestReleaseForCurrentBranch(releaseTags []string) string {

	developmentVersionSplit := strings.Split(developmentVersion, ".")
	//Split the string into major and minor version values
	majorVersionValue := parseInt(developmentVersionSplit[0])
	minorVersionValue := parseInt(developmentVersionSplit[1]) - 1

	// Iterate over all releases to configure the latest patch release
	latestPatch := 0
	var latestRelease string
	for _, version := range releaseTags {
		versionSplit := strings.Split(strings.TrimPrefix(version, "v"), ".")
		if parseInt(versionSplit[0]) == majorVersionValue && parseInt(versionSplit[1]) == minorVersionValue {
			if parseInt(versionSplit[2]) > latestPatch {
				latestPatch = parseInt(versionSplit[2])
			}
			latestRelease = fmt.Sprintf("v%s.%d.%d", versionSplit[0], minorVersionValue, latestPatch)
		}
	}

	return latestRelease
}

func getInterimRelease(releaseTags []string) string {
	// Get the latest release tag
	latestReleaseTag := releaseTags[len(releaseTags)-1]
	releaseTags = releaseTags[:len(releaseTags)-1]

	//Split the string excluding prefix 'v' into major and minor version values
	latestReleaseTagSplit := strings.Split(strings.TrimPrefix(latestReleaseTag, "v"), ".")
	majorVersionValue := parseInt(latestReleaseTagSplit[0])
	minorInterimVersionValue := parseInt(latestReleaseTagSplit[1]) - 1

	// Handles the case where only two or less minor releases are available for upgrade
	// E.g. where the latest version is 1.5.x and the first supported version for upgrade is 1.4.x, then return 1.5.0
	// as an interim release
	firstReleaseTag := releaseTags[0]
	//Split the string excluding prefix 'v' into major and minor version values
	firstReleaseTagSplit := strings.Split(strings.TrimPrefix(firstReleaseTag, "v"), ".")
	firstMajorVersionValue := parseInt(firstReleaseTagSplit[0])
	firstMinorVersionValue := parseInt(firstReleaseTagSplit[1])
	if firstMajorVersionValue == majorVersionValue && minorInterimVersionValue == firstMinorVersionValue {
		minorInterimVersionValue = parseInt(latestReleaseTagSplit[1])
		return fmt.Sprintf("v%d.%d.0\n", majorVersionValue, minorInterimVersionValue)
	}

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

	// Handles the case where only two or less minor releases are available for upgrade
	// E.g. where the latest version is 1.5.x and the first supported version for upgrade is 1.4.x
	// Get the first supported release tag for upgrade
	firstReleaseTag := releaseTags[0]
	//Split the string excluding prefix 'v' into major and minor version values
	firstReleaseTagSplit := strings.Split(strings.TrimPrefix(firstReleaseTag, "v"), ".")
	firstMajorVersionValue := parseInt(firstReleaseTagSplit[0])
	firstMinorVersionValue := parseInt(firstReleaseTagSplit[1])
	if firstMajorVersionValue == majorVersionValue && minorInstallVersionValue < firstMinorVersionValue {
		minorInstallVersionValue = firstMinorVersionValue
	}
	// Handles the major release case, e.g. where the latest version is 2.0.0 and the previous version is 1.4.2
	if minorInstallVersionValue < 0 {
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

		minorInstallVersionDiff := minorInstallVersionValue
		minorInstallVersionValue = 0
		for _, version := range releaseTags {
			versionSplit := strings.Split(strings.TrimPrefix(version, "v"), ".")
			if parseInt(versionSplit[0]) == majorVersionValue && parseInt(versionSplit[1]) > minorInstallVersionValue {
				if majorReleaseDecrementCount < 2 && minorInstallVersionDiff < -2 {
					minorInstallVersionValue = parseInt(versionSplit[1]) - 1
					//fmt.Println(minorInstallVersionValue)
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

func getTagsGTE(tags []string, oldestAllowedVersion string) (string, error) {
	builder := strings.Builder{}

	o, err := semver.NewSemVersion(oldestAllowedVersion)
	if err != nil {
		return "", err
	}

	for _, tag := range tags {
		var t = tag
		if tag[0] == 'v' || tag[0] == 'V' {
			t = tag[1:]
		}
		tagVersion, err := semver.NewSemVersion(t)
		if err != nil {
			return "", err
		}
		if tagVersion.IsGreaterThanOrEqualTo(o) {
			builder.WriteString(tag)
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		fmt.Printf("\nunable to convert the given string to int %s, %v", s, err.Error())
	}
	return n
}

// printUsage Prints the help for this program
func printUsage() {
	usageString := `

go run derive_upgrade_version.go [args] workspace version-type exclude-releases

Args:
	[workspace]  Uses the workspace path to retrieve the list of release tags using git tag command
	[version-type]     Specify version to derive
	[exclude-releases] list of release tags to exclude 
Options:
	--help	prints usage
`
	fmt.Print(usageString)
}
