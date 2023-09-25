// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"k8s.io/apimachinery/pkg/util/rand"
	"strings"
)

type NameGenerator struct {
	length  int
	charset []rune
}

var SimpleNameGenerator = newNameGenerator()

func newNameGenerator() NameGenerator {
	var charset []rune
	for i := 'a'; i <= 'z'; i++ {
		charset = append(charset, i)
	}
	return NameGenerator{
		length:  5,
		charset: charset,
	}
}

func (n NameGenerator) New(prefix string) string {
	sb := strings.Builder{}
	sb.WriteString(prefix)
	for i := 0; i < n.length; i++ {
		sb.WriteRune(n.charset[rand.Intn(len(n.charset))])
	}
	return sb.String()
}
