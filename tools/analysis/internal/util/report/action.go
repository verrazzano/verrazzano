// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package report handles reporting
package report

import (
	"errors"
	"go.uber.org/zap"
)

// TODO: Add helpers for working with Actions

// NOTE: This is part of the contract with the analyzers however it is currently an initial stake in the ground and
//		 will be evolving rapidly initially as we add analysis cases

// Action describes what a user/developer should do to mitigate an issue that has been found. For example:
//    - Description of the action if more general
//    - Link(s) to a Runbook(s) are preferable here as instructions may evolve over time and may be complex
//    - A list of Steps to take
type Action struct {
	Summary string   // Required, Summary of the action to take
	Links   []string // Optional, runbook or other related Links with action details
	Steps   []string // Optional, list of Steps to take (pointing to runbook is preferable if Actions are complex)
}

// Validate validates the action
func (action *Action) Validate(log *zap.SugaredLogger) (err error) {
	if len(action.Summary) == 0 {
		return errors.New("A Summary is required for an Action")
	}
	return nil
}

// Standard Action Summaries
const (
	ConsultRunbookUsingSupportingData = "Consult the runbook using the supporting data supplied"
)

// RunbookLinks are known runbook links
var RunbookLinks = map[string][]string{
	ImagePullBackOff:       {"TBD-ImagePullBackOffAction-runbook"},
	InsufficientMemory:     {"TBD-InsufficientMemory-runbook"},
	PodProblemsNotReported: {"TBD-PodProblemsNotReported-runbook"},
}

// KnownActions are Standard Action types
var KnownActions = map[string]Action{
	ImagePullBackOff:       {Summary: ConsultRunbookUsingSupportingData, Links: RunbookLinks[ImagePullBackOff]},
	InsufficientMemory:     {Summary: ConsultRunbookUsingSupportingData, Links: RunbookLinks[InsufficientMemory]},
	PodProblemsNotReported: {Summary: ConsultRunbookUsingSupportingData, Links: RunbookLinks[PodProblemsNotReported]},
}
