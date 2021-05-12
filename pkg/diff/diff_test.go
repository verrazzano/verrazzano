// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package diff

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"testing"
)

// Test structs
type X struct {
	String1     string
	String2     string
	String3     *string
	Int1        int
	Int2        *int
	Int32       int32
	Int64       *int64
	StringSlice []string
	YSlice1     []Y
	YSlice2     []*Y
	StringMap   map[string]string
	YMap        map[string]Y
	Time        time.Time
}

type Y struct {
	String      string
	Int         int
	StringSlice []string
	ZSlice      []Z
	ZMap        map[string]Z
}

type Z struct {
	String string
	Int    int
	Time   time.Time
}

func TestStrings(t *testing.T) {
	fromObj := X{String1: "sanitizeString"}
	toObj := X{String1: "sanitizeString"}
	verifyDiff(t, fromObj, toObj, false, "String unchanged")
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "String not specified")
	toObj = X{String1: ""}
	verifyDiff(t, fromObj, toObj, false, "String empty")
	toObj = X{String1: "fooa"}
	verifyDiff(t, fromObj, toObj, true, "String changed")

	fromObj = X{String1: "sanitizeString", String2: "foo2", String3: NewString("foo3")}
	toObj = X{String1: "sanitizeString"}
	verifyDiff(t, fromObj, toObj, false, "String pointer not specified")
	toObj = X{String1: "sanitizeString", String3: nil}
	verifyDiff(t, fromObj, toObj, false, "String pointer nil")
	toObj = X{String1: "sanitizeString", String3: NewString("foo3a")}
	verifyDiff(t, fromObj, toObj, true, "String pointer changed")
}

func TestInts(t *testing.T) {
	fromObj := X{Int1: 3}
	toObj := X{Int1: 3}
	verifyDiff(t, fromObj, toObj, false, "Int unchanged")
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "Int not specified")
	toObj = X{Int1: 0}
	verifyDiff(t, fromObj, toObj, false, "Int empty")
	toObj = X{Int1: 4}
	verifyDiff(t, fromObj, toObj, true, "Int changed")

	fromObj = X{Int1: 3, Int2: NewInt(4)}
	toObj = X{Int1: 3}
	verifyDiff(t, fromObj, toObj, false, "Int pointer not specified")
	toObj = X{Int1: 3, Int2: nil}
	verifyDiff(t, fromObj, toObj, false, "Int pointer nil")
	toObj = X{Int1: 3, Int2: NewInt(5)}
	verifyDiff(t, fromObj, toObj, true, "Int pointer changed")

	fromObj = X{Int32: 4}
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "Int32 not specified")
	toObj = X{Int32: 5}
	verifyDiff(t, fromObj, toObj, true, "Int32 changed")

	fromObj = X{Int64: NewInt64(6)}
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "Int64 pointer not specified")
	toObj = X{Int64: NewInt64(7)}
	verifyDiff(t, fromObj, toObj, true, "Int64 pointer changed")
}

func TestSlices(t *testing.T) {
	fromObj := X{StringSlice: []string{"foo1", "foo2", "foo3"}}
	toObj := X{StringSlice: []string{"foo1", "foo2", "foo3"}}
	verifyDiff(t, fromObj, toObj, false, "String slice unchanged")
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "String slice not specified")
	toObj = X{StringSlice: []string{}}
	verifyDiff(t, fromObj, toObj, false, "String slice empty")
	toObj = X{StringSlice: []string{"foo1", "foo2", "foo4"}}
	verifyDiff(t, fromObj, toObj, true, "String slice changed")
	toObj = X{StringSlice: []string{"foo1", "foo2"}}
	verifyDiff(t, fromObj, toObj, true, "String slice changed")

	fromObj = X{YSlice1: []Y{{String: "foo1", Int: 2}, {String: "foo2", Int: 3}}}
	toObj = X{YSlice1: []Y{{String: "foo1", Int: 2}, {String: "foo2", Int: 3}}}
	verifyDiff(t, fromObj, toObj, false, "Y slice unchanged")
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "Y slice not specified")
	toObj = X{YSlice1: []Y{}}
	verifyDiff(t, fromObj, toObj, false, "Y slice empty")
	toObj = X{YSlice1: []Y{{String: "foo1", Int: 2}, {String: "foo3", Int: 4}, {String: "foo2", Int: 3}}}
	verifyDiff(t, fromObj, toObj, true, "Y slice members changed")
	toObj = X{YSlice1: []Y{{String: "foo1", Int: 2}, {String: "foo2", Int: 4}}}
	verifyDiff(t, fromObj, toObj, true, "Y slice nested contents changed")

	fromObj = X{YSlice2: []*Y{{String: "foo1", Int: 2}, {String: "foo2", Int: 3}}}
	toObj = X{YSlice2: []*Y{{String: "foo1", Int: 2}, {String: "foo2", Int: 3}}}
	verifyDiff(t, fromObj, toObj, false, "Y pointer slice unchanged")
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "Y pointer slice not specified")
	toObj = X{YSlice2: []*Y{}}
	verifyDiff(t, fromObj, toObj, false, "Y pointer slice empty")
	toObj = X{YSlice2: []*Y{{String: "foo1", Int: 2}, {String: "foo3", Int: 4}, {String: "foo2", Int: 3}}}
	verifyDiff(t, fromObj, toObj, true, "Y pointer slice members changed")
	toObj = X{YSlice2: []*Y{{String: "foo1", Int: 2}, {String: "foo2", Int: 4}}}
	verifyDiff(t, fromObj, toObj, true, "Y pointer  slice nested contents changed")
}

func TestMaps(t *testing.T) {
	fromObj := X{StringMap: map[string]string{"foo1": "bar1", "foo2": "bar2", "foo3": "bar3"}}
	toObj := X{StringMap: map[string]string{"foo1": "bar1", "foo2": "bar2", "foo3": "bar3"}}
	verifyDiff(t, fromObj, toObj, false, "String map unchanged")
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "String map not specified")
	toObj = X{StringMap: map[string]string{}}
	verifyDiff(t, fromObj, toObj, false, "String map empty")
	toObj = X{StringMap: map[string]string{"foo1": "bar1", "foo2": "bar2", "foo3": "bar3", "foo4": "bar4"}}
	verifyDiff(t, fromObj, toObj, true, "String map element added")
	toObj = X{StringMap: map[string]string{"foo1": "bar1", "foo2": "bar2", "foo3": "bar4"}}
	verifyDiff(t, fromObj, toObj, true, "String map element changed")
	toObj = X{StringMap: map[string]string{"foo1": "bar1", "foo2": "bar2", "foo4": "bar4"}}
	verifyDiff(t, fromObj, toObj, true, "String map element added, one removed")
	toObj = X{StringMap: map[string]string{"foo1": "bar1", "foo2": "bar2"}}
	verifyDiff(t, fromObj, toObj, false, "String map value removed only")

	fromObj = X{YMap: map[string]Y{"foo1": {String: "foo1", Int: 2}, "foo2": {String: "foo2", Int: 3}}}
	toObj = X{YMap: map[string]Y{"foo1": {String: "foo1", Int: 2}, "foo2": {String: "foo2", Int: 3}}}
	verifyDiff(t, fromObj, toObj, false, "Y map unchanged")
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "Y map not specified")
	toObj = X{YMap: map[string]Y{}}
	verifyDiff(t, fromObj, toObj, false, "Y map empty")
	toObj = X{YMap: map[string]Y{"foo1": {String: "foo1", Int: 2}, "foo2": {String: "foo2", Int: 3}, "foo3": {String: "foo3", Int: 4}}}
	verifyDiff(t, fromObj, toObj, true, "Y map element added")
	toObj = X{YMap: map[string]Y{"foo1": {String: "foo1", Int: 2}, "foo2": {String: "foo2", Int: 4}}}
	verifyDiff(t, fromObj, toObj, true, "Y map element changed")
	toObj = X{YMap: map[string]Y{"foo1": {String: "foo1", Int: 2}, "foo3": {String: "foo3", Int: 4}}}
	verifyDiff(t, fromObj, toObj, true, "Y map element added, one removed")
	toObj = X{YMap: map[string]Y{"foo1": {String: "foo1", Int: 2}}}
	verifyDiff(t, fromObj, toObj, false, "Y map value removed only")
}

func TestTimes(t *testing.T) {
	now := time.Now()
	fromObj := X{Time: now}
	toObj := X{Time: now}
	verifyDiff(t, fromObj, toObj, false, "Time unchanged")
	toObj = X{}
	verifyDiff(t, fromObj, toObj, false, "Time not specified")
	toObj = X{Time: time.Time{}}
	verifyDiff(t, fromObj, toObj, false, "Time empty")
	toObj = X{Time: now.Add(time.Hour)}
	verifyDiff(t, fromObj, toObj, true, "Time changed")
}

// One unit test that contains a complex diff and actually verifies the full output
func TestOutput(t *testing.T) {
	fromObj := X{
		String1:     "foo1",
		String3:     NewString("foo3"),
		Int1:        0,
		Int2:        NewInt(4),
		Int32:       5,
		StringSlice: []string{"foo1", "foo2", "foo3"},
		YSlice1: []Y{
			{
				String: "foo1",
				Int:    2,
				ZSlice: []Z{{String: "foo1", Int: 3}},
				ZMap:   map[string]Z{"foo1": {String: "foo1", Int: 3}, "foo2": {String: "foo2", Int: 3}},
			},
			{
				String: "foo2",
				Int:    3,
				ZSlice: []Z{{String: "foo2", Int: 4}},
				ZMap:   map[string]Z{"foo1": {String: "foo1", Int: 3}, "foo2": {String: "foo2", Int: 3}},
			},
		},
		StringMap: map[string]string{"foo1": "bar1", "foo2": "bar2"},
		YMap: map[string]Y{
			"foo1": {
				String: "foo1",
				Int:    2,
				ZSlice: []Z{{String: "foo1", Int: 3}},
				ZMap:   map[string]Z{"foo1": {String: "foo1", Int: 3}, "foo2": {String: "foo2", Int: 3}},
			},
			"foo2": {
				String: "foo2",
				Int:    3,
				ZSlice: []Z{{String: "foo3", Int: 5}},
				ZMap:   map[string]Z{"foo3": {String: "foo3", Int: 4}},
			},
		},
	}
	toObj := X{
		String1:     "",
		String3:     nil,
		Int1:        0,
		Int2:        NewInt(5),
		StringSlice: []string{"foo1", "foo2"},
		YSlice1: []Y{
			{
				String: "foo1",
				Int:    2,
				ZSlice: []Z{{String: "foo1", Int: 3}},
				ZMap:   map[string]Z{"foo1": {String: "foo1", Int: 3}, "foo2": {String: "foo2", Int: 3}},
			},
			{
				String: "foo2",
				Int:    3,
				ZSlice: []Z{{String: "foo2", Int: 4}},
				ZMap:   map[string]Z{"foo1": {String: "foo1", Int: 3}, "foo2": {String: "foo2", Int: 4}},
			},
			{
				String: "foo3",
				Int:    4,
				ZSlice: []Z{{String: "foo2", Int: 4}},
				ZMap:   map[string]Z{"foo1": {String: "foo1", Int: 3}, "foo2": {String: "foo2", Int: 4}},
			},
		},
		StringMap: map[string]string{"foo1": "bar1", "foo2": "bar2"},
		YMap: map[string]Y{
			"foo1": {
				String: "foo1",
				Int:    2,
				ZSlice: []Z{{String: "foo1", Int: 3}},
				ZMap:   map[string]Z{"foo1": {String: "foo4", Int: 3}, "foo2": {String: "foo2", Int: 3}},
			},
			"foo3": {
				String: "foo3",
				Int:    4,
				ZSlice: []Z{{String: "foo3", Int: 5}},
				ZMap:   map[string]Z{"foo3": {String: "foo3", Int: 4}},
			},
		},
	}

	expectedResult := `  diff.X{
  	... // 4 ignored fields
- 	Int2: &4,
+ 	Int2: &5,
  	... // 2 ignored fields
  	StringSlice: []string{
  		"foo1",
  		"foo2",
- 		"foo3",
  	},
  	YSlice1: []diff.Y{
  		{String: "foo1", Int: 2, ZSlice: {{String: "foo1", Int: 3}}, ZMap: {"foo1": {String: "foo1", Int: 3}, "foo2": {String: "foo2", Int: 3}}},
  		{
  			... // 1 ignored and 2 identical fields
  			ZSlice: {{String: "foo2", Int: 4}},
  			ZMap: map[string]diff.Z{
  				"foo1": {String: "foo1", Int: 3},
  				"foo2": {
  					String: "foo2",
- 					Int:    3,
+ 					Int:    4,
  					... // 1 ignored field
  				},
  			},
  		},
+ 		{
+ 			String: "foo3",
+ 			Int:    4,
+ 			ZSlice: []diff.Z{{String: "foo2", Int: 4}},
+ 			ZMap:   map[string]diff.Z{"foo1": {String: "foo1", Int: 3}, "foo2": {String: "foo2", Int: 4}},
+ 		},
  	},
  	... // 1 ignored field
  	StringMap: {"foo1": "bar1", "foo2": "bar2"},
  	YMap: map[string]diff.Y{
  		"foo1": {
  			... // 1 ignored and 2 identical fields
  			ZSlice: {{String: "foo1", Int: 3}},
  			ZMap: map[string]diff.Z{
  				"foo1": {
- 					String: "foo1",
+ 					String: "foo4",
  					Int:    3,
  					... // 1 ignored field
  				},
  				"foo2": {String: "foo2", Int: 3},
  			},
  		},
  		... // 1 ignored entry
+ 		"foo3": {
+ 			String: "foo3",
+ 			Int:    4,
+ 			ZSlice: []diff.Z{{String: "foo3", Int: 5}},
+ 			ZMap:   map[string]diff.Z{"foo3": {String: "foo3", Int: 4}},
+ 		},
  	},
  	... // 1 ignored field
  }`

	assert.Equal(t, sanitizeString(expectedResult), sanitizeString(Diff(fromObj, toObj)), "Many nested changes")
}

// Some examples with real k8s Deployment objects
func TestK8sDeployment(t *testing.T) {
	// Empty deployments
	fromDeployment := appsv1.Deployment{}
	toDeployment := appsv1.Deployment{}
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Empty objects")

	// Simple values set in the source deployment and not in the target
	fromDeployment.Name = "sanitizeString"
	fromDeployment.Spec.Replicas = NewInt32(5)
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Empty values in source")

	// Simple values set in the target deployment
	toDeployment.Name = "sanitizeString"
	toDeployment.Spec.Replicas = NewInt32(5)
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Same values in source object as from")
	toDeployment.Spec.Replicas = NewInt32(6)
	assert.NotEqual(t, "", Diff(fromDeployment, toDeployment), "Different value in source object from target")
	toDeployment.Spec.Replicas = NewInt32(5)
	toDeployment.Spec.MinReadySeconds = 500
	assert.NotEqual(t, "", Diff(fromDeployment, toDeployment), "Different value in source object from target")
	toDeployment.Spec.MinReadySeconds = 0

	// List (containers) specified in from deployment, empty in to
	fromContainer := corev1.Container{}
	fromContainer.Name = "bar"
	fromDeployment.Spec.Template.Spec.Containers = []corev1.Container{fromContainer}
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Empty list in source object")
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{}
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Empty list in source object")

	// Non-empty list (containers) in to
	toContainer := corev1.Container{}
	toContainer.Name = "bar"
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{toContainer}
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Same values in source object as from")
	toContainer.Name = "bar1"
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{toContainer}
	assert.NotEqual(t, "", Diff(fromDeployment, toDeployment), "Different value in source object from target")
	toContainer.Name = "bar"
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{toContainer, toContainer}
	assert.NotEqual(t, "", Diff(fromDeployment, toDeployment), "Different sized list in source object from target")

	// Same thing as the above, but go one level deeper into EnvVars for a Container
	// List (EnvVars) specified in source deployment, empty in target deployment
	fromContainer = corev1.Container{}
	fromContainer.Env = []corev1.EnvVar{{Name: "sanitizeString", Value: "bar"}}
	fromDeployment.Spec.Template.Spec.Containers = []corev1.Container{fromContainer}
	toContainer = corev1.Container{}
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{toContainer}
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Empty list in source object")
	toContainer = corev1.Container{}
	toContainer.Env = []corev1.EnvVar{}
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{toContainer}
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Empty list in source object")

	// Non-empty list (EnvVars) in to
	toContainer = corev1.Container{}
	toContainer.Env = []corev1.EnvVar{{Name: "sanitizeString", Value: "bar"}}
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{toContainer}
	assert.Equal(t, "", Diff(fromDeployment, toDeployment), "Same values in source object as target")
	toContainer.Env = []corev1.EnvVar{{Name: "sanitizeString", Value: "bar1"}}
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{toContainer}
	assert.NotEqual(t, "", Diff(fromDeployment, toDeployment), "Different value in source object from target")
	toContainer.Env = []corev1.EnvVar{{Name: "sanitizeString", Value: "bar"}, {Name: "foo1", Value: "bar1"}}
	toDeployment.Spec.Template.Spec.Containers = []corev1.Container{toContainer}
	assert.NotEqual(t, "", Diff(fromDeployment, toDeployment), "Different sized list in source object from target")
}

func verifyDiff(t *testing.T, fromObj interface{}, toObj interface{}, expectDiffs bool, description string) {
	diffResult := Diff(fromObj, toObj)
	if expectDiffs {
		assert.NotEmpty(t, diffResult, description)
	} else {
		assert.Empty(t, diffResult, description)
	}
}

// Removes unicode tabs from the given string - this is needed to workaround issue https://github.com/google/go-cmp/issues/235
// in gocmp, which causes inconsistent Diff() output
func sanitizeString(s string) string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	lines := []string{}
	for scanner.Scan() {
		line, _ := strconv.Unquote(strings.Replace(fmt.Sprintf("%#v", scanner.Text()), "\\u00a0", " ", -1))
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func NewInt(value int) *int {
	var val = value
	return &val
}

func NewInt32(value int32) *int32 {
	var val = value
	return &val
}

func NewInt64(value int64) *int64 {
	var val = value
	return &val
}

func NewString(value string) *string {
	var val = value
	return &val
}
