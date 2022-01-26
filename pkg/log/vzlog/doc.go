// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzlog

// The vzlog package provides logging for messages in controllers and other
// code that wants to throttle messages.  The Verrazzano Platform Operator controller for
// the Verrazzano resource is an example where this should be used.  During a reconcile cycle,
// the controller-runtime calls the controller Reconcile method repeatedly until the resource has
// been reconciled.  For something like Verrazzano install, this can translate into scores of
// reconcile calls.  The controller Reconcile code has no way of knowing if it already displayed
// an info message, such as "Keyclaok waiting for Verrazzano secret".  Using normal logging, that message
// might be displayed dozens of times.  With the progress logger, the message is only displayed
// periodically, once a minute by default.  Furthermore, once a message is display, and a new message
// is logged, that old message will never be displayed again.  This allows controllers to log
// informative progress messages, without overwhelming the log files with superfluous information.
//
// This logger is initialized with the zap.SugaredLogger and then used instead of the zap logger directly.
// The same SugaredLogger calls can be made: Debug, Debugf, Info, Infof, Error, and Errorf.  The
// two new calls are Progress and Progressf. The S() method will return the underlying SugaredLogger.
// The following psuedo-code shows how this should be used:
//
//   log := vzlog.EnsureLogContext(key).EnsureLogger("default", zaplog, zaplog)
//
// Display info and errors as usual
//   p.Errorf(...)
//   p.Info(...)
//
// Display progress
//   p.Progress("Reconciling namespace/resource")
//
// Display Keycloak progress
//   cl := l.GetContext().EnsureLogger("Keycloak")
//   cl.Progress("Waiting for Verrazzano secret")
//   cl.Errorf(...)
//
// Display Istio progress
//   cl := l.GetContext().EnsureLogger("Istio")
//   cl.Progress("Waiting for Istio to start")
//   cl.Errorf(...)
