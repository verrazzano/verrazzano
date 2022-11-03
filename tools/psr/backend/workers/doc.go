// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

/*
Package workers contains a set of workers that implement PSR use cases.

# Worker
Each worker does a single task, such as generate logs. The Worker.Work function is called
repeatedly by the workmanager.Runner, which implements the loop.  If the worker
has a dependency that doesn't exist, like OpenSearch, it should return an error.

## Immutable
Each worker should use a value receiver so that the worker is immutable.  Any mutable state should be keep in
a struct or structs pointed to by the worker.  For example, see runner.go *runnerMetrics in the runner struct.

# Metrics
Each worker can and should generate metrics.  The metrics much be thread safe since the collection will be
done from a go routine, see workmanager/runner.go for an example.  The metrics collection is generically handled by the
metrics/collector.go code.
*/
package workers
