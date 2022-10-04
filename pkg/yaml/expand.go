// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"errors"
	"strings"
)

// Expand a dot notated name to a YAML string.  The value can be a string or string list.
// The simplest YAML is:
// a: b
//
// Nested values are expanded as follows:
//
//	a.b.c : v1
//	  expands to
//	a:
//	  b:
//	    c: v1
//
// If there is more than one value then
//
//	a.b : {v1,v2}
//	  expands to
//	a:
//	  b:
//	    - v1
//	    - v2
//
// The last segment of the name might be a quoted string, for example:
//
//	controller.service.annotations."service\.beta\.kubernetes\.io/oci-load-balancer-shape" : 10Mbps
//
// which translates to
//
//	controller:
//	  service:
//	    annotations:
//	      service.beta.kubernetes.io/oci-load-balancer-shape: 10Mbps
//
// If forcelist is true then always use the list format.
func Expand(leftMargin int, forceList bool, name string, vals ...string) (string, error) {
	const indent = 2
	b := strings.Builder{}

	// Remove trailing quote and split the string at a quote if is exists
	name = strings.TrimSpace(name)
	name = strings.TrimRight(name, "\"")
	quoteSegs := strings.Split(name, "\"")
	if len(quoteSegs) > 2 {
		return "", errors.New("Name/Value pair has invalid name with more than 1 quoted string")
	}
	if len(quoteSegs) == 0 {
		return "", errors.New("Name/Value pair has invalid name")
	}
	// Remove any trailing dot and split the first part of the string at the dots.
	unquotedPart := strings.TrimRight(quoteSegs[0], ".")
	// Replace backslashed dots because of no negative lookbehind
	placeholder := "/*placeholder*/"
	unquotedPart = strings.Replace(unquotedPart, "\\.", placeholder, -1)
	nameSegs := strings.Split(unquotedPart, ".")
	if len(quoteSegs) == 2 {
		// Add back the original quoted string if it existed
		// e.g. change service\.beta\.kubernetes\.io/oci-load-balancer-shape to
		//             service.beta.kubernetes.io/oci-load-balancer-shape
		s := strings.Replace(quoteSegs[1], "\\", "", -1)
		nameSegs = append(nameSegs, s)
	}
	// Loop through all the name segments, for example, these 4:
	//    controller, service, annotations, service.beta.kubernetes.io/oci-load-balancer-shape
	listIndents := 0
	nextValueList := false
	indentVal := " "
	for i, seg := range nameSegs {
		// Get rid of placeholder
		seg = strings.Replace(seg, placeholder, ".", -1)

		// Create the padded indent
		pad := strings.Repeat(indentVal, leftMargin+(indent*(i+listIndents)))

		// last value for formatting
		lastVal := i == len(nameSegs)-1

		// Check if current value is a new list value
		listValueString := ""
		if nextValueList {
			listValueString = "- "
			listIndents++
			nextValueList = false
		}

		// Check if internal list value next
		splitList := strings.Split(seg, `[`)
		if len(splitList) > 1 {
			seg = splitList[0]
			nextValueList = true
		}

		// Write the indent padding, then name followed by colon
		if _, err := b.WriteString(pad + listValueString + seg + ":"); err != nil {
			return "", err
		}
		// If this is the last segment then write the value, else LF
		if lastVal {
			// indent is different based on if the last value was a list
			indentSize := 1
			if nextValueList {
				indentSize = 2
			}
			pad += strings.Repeat(indentVal, indent*indentSize)
			if err := writeVals(&b, forceList || nextValueList, pad, vals...); err != nil {
				return "", err
			}
		} else {
			if _, err := b.WriteString("\n"); err != nil {
				return "", err
			}
		}
	}
	return b.String(), nil
}

// writeVals writes a single value or a list of values to the string builder.
// If forcelist is true then always use the list format.
func writeVals(b *strings.Builder, forceList bool, pad string, vals ...string) error {
	// check for multiline value
	if len(vals) == 1 && strings.Contains(vals[0], "\n") {
		b.WriteString(" |\n")
		b.WriteString(pad + strings.Replace(vals[0], "\n", "\n"+pad, -1))
		return nil
	}
	if len(vals) == 1 && !forceList {
		// Write the single value, for example:
		// key: val1
		_, err := b.WriteString(" " + vals[0])
		return err
	}
	// Write the list of values, for example
	//  key:
	//    - val1
	//    - val2
	for _, val := range vals {
		if _, err := b.WriteString("\n"); err != nil {
			return err
		}
		if _, err := b.WriteString(pad + "- " + val); err != nil {
			return err
		}
	}
	return nil
}
