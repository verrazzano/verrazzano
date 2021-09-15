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
func Expand(name string, val string, indent int) (string, error) {
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

		// Add the indent
		if len(pad) > 0 {
			if _, err := b.WriteString(pad); err != nil {
				return "", err
			}
		}
		// Write the name followed by colon
		if _, err := b.WriteString(seg + ":"); err != nil {
			return "", err
		}
		// If this is the last segment then write the value, else LF
		if i == len(nameSegs)-1 {
			if _, err := b.WriteString(" " + val); err != nil {
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
