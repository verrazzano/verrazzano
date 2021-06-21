// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"errors"
	"fmt"
	"io"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"strings"
	"time"
)

// PrintTable will print the data in a well-formatted table with the headings at top
func PrintTable(headings []string, data [][]string, w io.Writer) error {
	// make sure the data has the same number of columns as the headings
	if len(headings) != len(data[0]) {
		return errors.New("wrong number of columns in data")
	}

	output, err := formatOutput(headings, data)
	if err != nil {
		return err
	}

	//fmt.Println(output)
	_, err = fmt.Fprintln(w, output)
	return err

}

func formatOutput(headings []string, data [][]string) (string, error) {
	// work out how long each column needs to be
	var lengths []int

	// work through each column
	for i, heading := range headings {
		// start with the length of the heading
		length := len(heading)

		// if any of the items in this column are longer than the heading, use the longest
		for _, row := range data {
			if len(row[i]) > length {
				length = len(row[i])
			}
		}

		// leave a 3 character space between columns
		length += 3
		lengths = append(lengths, length)
	}

	var output = ""

	// now format the string according to the calculated lengths
	for i, heading := range headings {
		output += fmt.Sprintf("%-*s", lengths[i], strings.ToUpper(heading))
	}
	output += "\n"
	for _, row := range data {
		for i, item := range row {
			output += fmt.Sprintf("%-*s", lengths[i], item)
		}
		output += "\n"
	}

	return output, nil
}

// FormatStringSlice formats a string slice as a comma separated list
func FormatStringSlice(in []string) string {
	output := in[0]
	for _, s := range in[1:] {
		output += "," + s
	}
	return output
}

// Age takes a Time and returns the length of time from then to now
// in a compact format like 2d3h
func Age(createTime v1.Time) string {
	return duration.HumanDuration(time.Since(createTime.Time))
}
