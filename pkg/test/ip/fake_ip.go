// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ip

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func RandomIP() string {
	n1, _ := rand.Int(rand.Reader, big.NewInt(256))
	n2, _ := rand.Int(rand.Reader, big.NewInt(256))
	n3, _ := rand.Int(rand.Reader, big.NewInt(256))
	n4, _ := rand.Int(rand.Reader, big.NewInt(256))
	return fmt.Sprintf("%d.%d.%d.%d", n1, n2, n3, n4)
}
