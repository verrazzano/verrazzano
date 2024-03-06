// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"flag"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"os"
	"os/exec"
	"sort"
	"strings"
)

const (
	VersionForInstall             = "install-version"
	InterimVersionForUpgrade      = "interim-version"
	LatestVersionForCurrentBranch = "latest-version-for-branch"
	VersionsGTE                   = "versions-gte"
	VersionsLT                    = "versions-lt"
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
	case VersionsLT:
		tagsBefore, err := getTagsLT(releaseTags, developmentVersion)
		if err != nil {
			panic(err)
		}
		fmt.Println(tagsBefore)
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
		// Receive development version of the current branch.
		developmentVersion = args[2]
	} else {
		fmt.Printf("no worspace path and version type line arguments were specified\n")
		os.Exit(1)
	}

	if len(args) > 2 {
		for index, arg := range args {
			// Remove any ',' or ']' suffixes and remove any '[' prefix
			trimArg := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSuffix(arg, ","), "]"), "[")
			if index > 2 {
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
	majorMinorReleaseTag := removePatchVersion(releaseTag)
	builder := strings.Builder{}
	if !strings.HasPrefix(majorMinorReleaseTag, strings.ToLower("v")) {
		builder.WriteString("v" + majorMinorReleaseTag)
		majorMinorReleaseTag = builder.String()
	}
	for _, excludeTag := range excludeReleaseTags {
		builder.Reset()
		majorMinorExcludeTag := removePatchVersion(excludeTag)
		if !strings.HasPrefix(majorMinorExcludeTag, strings.ToLower("v")) {
			builder.WriteString("v" + majorMinorExcludeTag)
			majorMinorExcludeTag = builder.String()
		}
		if majorMinorExcludeTag == majorMinorReleaseTag {
			return true
		}
	}
	return false
}

func getLatestReleaseForCurrentBranch(releaseTags []string) string {

	builder := strings.Builder{}
	var latestForCurrentBranch *semver.SemVersion

	o, err := semver.NewSemVersion(developmentVersion)
	o.Patch = 0
	if err != nil {
		return ""
	}
	for _, tag := range releaseTags {
		var t = tag
		tagVersion, err := semver.NewSemVersion(t)
		if err != nil {
			return ""
		}
		if tagVersion.IsLessThan(o) {
			latestForCurrentBranch = tagVersion
		}
	}
	builder.WriteString("v" + latestForCurrentBranch.ToString())

	return builder.String()
}

// Derives interim version which is the latest git release tag - 1 minor version.
// If there are only two unique minor versions then latest patch is derived.
func getInterimRelease(releaseTags []string) string {
	resultTags, _ := getTagsLTMinorRelease(releaseTags, developmentVersion)
	minorAndPatchesVersionMap, uniqueMinorVersions := getUniqueMajorMinorAndPatchVersionMap(resultTags)
	uniqueMinorReleaseCount := len(uniqueMinorVersions)

	// Handles edge cases such as having less than 2 minor releases.
	var interimRelease string
	var installReleasePatchVersions []string
	if uniqueMinorReleaseCount == 1 {
		installReleasePatchVersions = minorAndPatchesVersionMap[uniqueMinorVersions[0]]
		interimRelease = installReleasePatchVersions[len(releaseTags)-1]
	} else if uniqueMinorReleaseCount == 2 {
		secondLatestRelease := uniqueMinorVersions[len(uniqueMinorVersions)-2]
		installReleasePatchVersions = minorAndPatchesVersionMap[secondLatestRelease]
		interimRelease = installReleasePatchVersions[len(installReleasePatchVersions)-1]
	} else if uniqueMinorReleaseCount > 2 {
		secondLatestRelease := uniqueMinorVersions[len(uniqueMinorVersions)-2]
		installReleasePatchVersions = minorAndPatchesVersionMap[secondLatestRelease]
		interimRelease = installReleasePatchVersions[len(installReleasePatchVersions)-1]
	}
	return interimRelease
}

// Derives install version which is the latest git release tag - 2 minor version.
// If there are only two unique minor versions then oldest patch is derived.
func getInstallRelease(releaseTags []string) string {
	resultTags, _ := getTagsLTMinorRelease(releaseTags, developmentVersion)
	minorAndPatchesVersionMap, uniqueMinorVersions := getUniqueMajorMinorAndPatchVersionMap(resultTags)
	uniqueMinorReleaseCount := len(uniqueMinorVersions)

	// Handles edge cases such as having less than 2 minor releases.
	var installRelease string
	var installReleasePatchVersions []string
	if uniqueMinorReleaseCount == 1 {
		installReleasePatchVersions = minorAndPatchesVersionMap[uniqueMinorVersions[0]]
		installRelease = installReleasePatchVersions[0]
	} else if uniqueMinorReleaseCount == 2 {
		thirdLatestRelease := uniqueMinorVersions[len(uniqueMinorVersions)-2]
		installReleasePatchVersions = minorAndPatchesVersionMap[thirdLatestRelease]
		installRelease = installReleasePatchVersions[0]
	} else if uniqueMinorReleaseCount > 2 {
		thirdLatestRelease := uniqueMinorVersions[len(uniqueMinorVersions)-3]
		installReleasePatchVersions = minorAndPatchesVersionMap[thirdLatestRelease]
		installRelease = installReleasePatchVersions[len(installReleasePatchVersions)-1]
	}
	return installRelease
}

func getTagsLT(tags []string, oldestAllowedVersion string) (string, error) {
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
		if tagVersion.IsLessThan(o) {
			builder.WriteString(tag)
			builder.WriteString(" ")
		}
	}

	return builder.String(), nil
}

func getTagsLTMinorRelease(tags []string, developmentVersion string) ([]string, error) {
	builder := strings.Builder{}
	var resultTags []string
	o, err := semver.NewSemVersion(developmentVersion)
	o.Patch = 0
	if err != nil {
		return resultTags, err
	}

	for _, tag := range tags {
		var t = tag
		if tag[0] == 'v' || tag[0] == 'V' {
			t = tag[1:]
		}
		tagVersion, err := semver.NewSemVersion(t)
		if err != nil {
			return resultTags, err
		}
		if tagVersion.IsLessThan(o) {
			builder.WriteString(tag)
			builder.WriteString(" ")
		}
	}
	resultTags = strings.Fields(builder.String())
	return resultTags, nil
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

func removePatchVersion(tag string) string {
	split := strings.Split(tag, ".")
	return strings.Join(split[:2], ".")
}

func compareVersions(v1, v2 string) bool {
	v1Split := strings.Split(v1, ".")
	v2Split := strings.Split(v2, ".")

	for i := 0; i < len(v1Split) && i < len(v2Split); i++ {
		var num int
		v1Num, _ := fmt.Sscanf(v1Split[i], "%d", &num)
		v2Num, _ := fmt.Sscanf(v2Split[i], "%d", &num)

		if v1Num != v2Num {
			return v1Num < v2Num
		}
	}
	return false
}

func getUniqueMajorMinorAndPatchVersionMap(releaseTags []string) (map[string][]string, []string) {
	// Remove patch minorVersion
	majorMinorVersions := make([]string, len(releaseTags))
	for i, tag := range releaseTags {
		majorMinorVersions[i] = removePatchVersion(tag)
	}

	// Sort based on the major and minor minorVersion
	sort.SliceStable(majorMinorVersions, func(i, j int) bool {
		return compareVersions(majorMinorVersions[i], majorMinorVersions[j])
	})

	// Remove duplicates
	uniqueMinorVersions := make([]string, 0)
	seen := make(map[string]bool)
	for _, minorVersion := range majorMinorVersions {
		if !seen[minorVersion] {
			seen[minorVersion] = true
			uniqueMinorVersions = append(uniqueMinorVersions, minorVersion)
		}
	}

	// Create a map of unique majorMinorVersions and related patch versions
	minorAndPatchesVersionMap := make(map[string][]string)
	for _, version := range majorMinorVersions {
		if _, ok := minorAndPatchesVersionMap[version]; !ok {
			minorAndPatchesVersionMap[version] = make([]string, 0)
		}
	}

	for _, tag := range releaseTags {
		version := removePatchVersion(tag)
		minorAndPatchesVersionMap[version] = append(minorAndPatchesVersionMap[version], tag)
	}

	return minorAndPatchesVersionMap, uniqueMinorVersions
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
