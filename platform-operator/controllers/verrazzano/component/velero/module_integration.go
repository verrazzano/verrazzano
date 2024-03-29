// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
)

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (v veleroHelmComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return nil
}
