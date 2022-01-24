// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package progress

// The progress package provides logging for progress messages in controllers and other
// code that wants to throttle messages.  The Verrazzano Platform Operator controller for
// the Verrazzano resource is an example where this should be used.  Since the Reconcile function
// is called repeatedly every few seconds during installation, the code has no way of
// knowing if it already displayed an info message, such as waiting for Verrazzano secret.  Using
// normal logging, that message might be displayed dozens of times.  With the progress logger, the
// message is only displayed periodically, once a minute by default.  Furthermore, once a message is
// display, and a new message is logged, that old message will never be displayed again.  This allows
// controllers to log informative progress messages, without overwhelming the log files with superfluous
// information.
//
// This logger is initialized with the zap.SugaredLogger logger then used throught the instead of the zap logger.
// The same SugaredLogger calls can be made.  When the code needs to log progress, it should get the progress logger.
// The following psuedo-code shows how it should be used
//
//   l := EnsureResourceLogger("namespace/myresource", zap.S()).DefaultProgressLogger()
//
//   Display info and errors as usual
//   p.Errorf(...)
//   p.Info(...)
//
//   Display progress
//   p.Progress("Reconciling namespace/resource")
//
// cl := rl.EnsureProgressLogger("Keycloak")
// p.Progress("Reconciling namespace/resource")
