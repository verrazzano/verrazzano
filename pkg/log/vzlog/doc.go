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
// The 'Once' and 'Progress' logging is done in the context of a reconcile session for a resource change.
// This means if you change resource foo, and the controller reconciler is called 100 times, A call to log.Once will
// log the message once.  When the resource foo is finally reconciled (return ctrl.Result{}, nil to controller runtime),
// then we delete the logging context. So, if you changed the resource foo an hour later,
// and the same code path is executed, then the message will be displayed once, for the new reconcile session.
//
// The main purpose of this package is to provide logging during Kubernetes resource reconciliation.  For that
// use case, use the EnsureResourceLogger method as follows.  See the function descrition for more details.
//
//   	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
//		Name:           vz.Name,
//		Namespace:      vz.Namespace,
//		ID:             string(vz.UID),
//		Generation:     vz.Generation,
//		ControllerName: "verrazzano",
//	})
//
// For other use cases, you can call the lower level functions to explicitly create a LogContext and VerrazzanoLogger
// as described here. The logger is initialized with the zap.SugaredLogger and then used instead of the zap logger directly.
// The same SugaredLogger calls can be made: Debug, Debugf, Info, Infof, Error, and Errorf.  The
// two new calls are Progress and Progressf. The S() method will return the underlying SugaredLogger.
// The following pseudo-code shows how this should be used:
//
//   log := vzlog.EnsureContext(key).EnsureLogger("default", zaplog, zaplog)
//
// Display info and errors as usual
//   p.Errorf(...)
//   p.Info(...)
//
// Display progress
//   p.Progress("Reconciling namespace/resource")
//
// Display Keycloak progress
//   cl := l.EnsureContext().EnsureLogger("Keycloak")
//   cl.Progress("Waiting for Verrazzano secret")
//   cl.Errorf(...)
//
// Display Istio progress
//   cl := l.EnsureContext().EnsureLogger("Istio")
//   cl.Progress("Waiting for Istio to start")
//   cl.Errorf(...)
