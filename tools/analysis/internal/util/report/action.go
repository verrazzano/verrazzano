// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package report handles reporting
package report

import (
	"errors"
	"fmt"
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
	ConsultRunbook = "Consult %s using supporting details identified in the report"
)

// RunbookLinks are known runbook links
var RunbookLinks = map[string][]string{
	ImagePullBackOff:          {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/ImagePullBackOffAction.md"},
	ImagePullRateLimit:        {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/ImagePullRateLimit.md"},
	ImagePullNotFound:         {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/ImagePullNotFound.md"},
	ImagePullService:          {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/ImagePullService.md"},
	InsufficientMemory:        {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/InsufficientMemory.md"},
	IngressInstallFailure:     {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/IngressInstallFailure.md"},
	IngressNoLoadBalancerIP:   {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/IngressNoLoadBalancerIP.md"},
	IngressOciIPLimitExceeded: {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/IngressOciIPLimitExceeded.md"},
	InstallFailure:            {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/InstallFailure.md"},
	PendingPods:               {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/PendingPods.md"},
	PodProblemsNotReported:    {"https://github.com/verrazzano/verrazzano/tree/master/tools/analysis/advice/PodProblemsNotReported.md"},
}

// KnownActions are Standard Action types
var KnownActions = map[string]Action{
	ImagePullBackOff:          {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ImagePullBackOff][0])},
	ImagePullRateLimit:        {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ImagePullRateLimit][0])},
	ImagePullNotFound:         {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ImagePullNotFound][0])},
	ImagePullService:          {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ImagePullService][0])},
	InsufficientMemory:        {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[InsufficientMemory][0])},
	IngressInstallFailure:     {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressInstallFailure][0])},
	IngressNoLoadBalancerIP:   {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressNoLoadBalancerIP][0])},
	IngressOciIPLimitExceeded: {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressOciIPLimitExceeded][0])},
	InstallFailure:            {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[InstallFailure][0])},
	PendingPods:               {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[PendingPods][0])},
	PodProblemsNotReported:    {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[PodProblemsNotReported][0])},
}

func getConsultRunbookAction(summaryF string, runbookLink string) string {
	return fmt.Sprintf(summaryF, runbookLink)
}
