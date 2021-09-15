// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"errors"
	"strings"
)

// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Expand a dot separated name to a YAML hierachical string
//
// Handle the case where the last segment of the name is a quoted string, for example:
//
//   controller.service.annotations."service\.beta\.kubernetes\.io/oci-load-balancer-shape"
// which translates to
//   controller:
//     service:
//       annotations:
//         service.beta.kubernetes.io/oci-load-balancer-shape: 10Mbps
//
func Expand(indent int, name string, vals ...string) (string, error) {
	b := strings.Builder{}

	// Remove trailing quote and split the string at a quote if is exists
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
	for i, seg := range nameSegs {
		// Create the padded indent
		pad := strings.Repeat(" ", indent*i)

		// Write the indent padding, then name followed by colon
		if _, err := b.WriteString(pad + seg + ":"); err != nil {
			return "", err
		}
		// If this is the last segment then write the value, else LF
		if i == len(nameSegs)-1 {
			if err := writeVals(&b, pad, vals...); err != nil {
				return "", err
			}
		} else {
			if _, err := b.WriteString("\n"); err != nil {
				return "", err
			}
		}
	}
	return b.String(), nil
	// TODO add valueList
}

// writeVals writes a single value or a list of values to the string builder
func writeVals(b *strings.Builder, pad string, vals ...string) error {
	if len(vals) == 1 {
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
		if _, err := b.WriteString(pad + "  - " + val); err != nil {
			return err
		}
	}
	return nil
}
