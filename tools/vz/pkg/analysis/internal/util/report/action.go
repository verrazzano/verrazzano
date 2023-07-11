// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package report handles reporting
package report

import (
	"errors"
	"fmt"

	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"go.uber.org/zap"
)

// TODO: Add helpers for working with Actions

// NOTE: This is part of the contract with the analyzers however it is currently an initial stake in the ground and
//		 will be evolving rapidly initially as we add analysis cases

// Action describes what a user/developer should do to mitigate an issue that has been found. For example:
//   - Description of the action if more general
//   - Link(s) to a Runbook(s) are preferable here as instructions may evolve over time and may be complex
//   - A list of Steps to take
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
	ImagePullBackOff:             {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/imagepullbackoff"},
	ImagePullRateLimit:           {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/imagepullratelimit"},
	ImagePullNotFound:            {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/imagepullnotfound"},
	ImagePullService:             {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/imagepullservice"},
	InsufficientMemory:           {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/insufficientmemory"},
	InsufficientCPU:              {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/insufficientcpu"},
	IngressInstallFailure:        {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/ingressinstallfailure"},
	IngressLBLimitExceeded:       {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/ingresslblimitexceeded"},
	IngressNoLoadBalancerIP:      {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/ingressnoloadbalancerip"},
	IngressOciIPLimitExceeded:    {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/ingressociiplimitexceeded"},
	InstallFailure:               {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/installfailure"},
	PendingPods:                  {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/pendingpods"},
	PodProblemsNotReported:       {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/podproblemsnotreported"},
	IngressNoIPFound:             {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/ingressnoloadbalancerip"},
	IstioIngressNoIP:             {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/istioingressgatewaynoloadbalancerip"},
	IngressShapeInvalid:          {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/ingressinvalidshape"},
	IstioIngressPrivateSubnet:    {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/istioloadbalancercreation"},
	NginxIngressPrivateSubnet:    {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/nginxloadbalancercreation"},
	ExternalDNSConfigureIssue:    {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/externaldnsconfiguration"},
	KeycloakDataMigrationFailure: {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/keycloakdatamigrationfailure"},
	KontainerDriverNotReady:      {"https://verrazzano.io/" + version.GetEffectiveDocsVersion() + "/docs/troubleshooting/diagnostictools/analysisadvice/kontainerdrivernotready"},
}

// KnownActions are Standard Action types
var KnownActions = map[string]Action{
	ImagePullBackOff:             {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ImagePullBackOff][0])},
	ImagePullRateLimit:           {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ImagePullRateLimit][0])},
	ImagePullNotFound:            {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ImagePullNotFound][0])},
	ImagePullService:             {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ImagePullService][0])},
	InsufficientMemory:           {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[InsufficientMemory][0])},
	InsufficientCPU:              {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[InsufficientCPU][0])},
	IngressInstallFailure:        {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressInstallFailure][0])},
	IngressLBLimitExceeded:       {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressLBLimitExceeded][0])},
	IngressNoLoadBalancerIP:      {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressNoLoadBalancerIP][0])},
	IngressOciIPLimitExceeded:    {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressOciIPLimitExceeded][0])},
	InstallFailure:               {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[InstallFailure][0])},
	PendingPods:                  {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[PendingPods][0])},
	PodProblemsNotReported:       {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[PodProblemsNotReported][0])},
	IngressNoIPFound:             {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressNoIPFound][0])},
	IstioIngressNoIP:             {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IstioIngressNoIP][0])},
	IngressShapeInvalid:          {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IngressShapeInvalid][0])},
	IstioIngressPrivateSubnet:    {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[IstioIngressPrivateSubnet][0])},
	NginxIngressPrivateSubnet:    {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[NginxIngressPrivateSubnet][0])},
	ExternalDNSConfigureIssue:    {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[ExternalDNSConfigureIssue][0])},
	KeycloakDataMigrationFailure: {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[KeycloakDataMigrationFailure][0])},
	KontainerDriverNotReady:      {Summary: getConsultRunbookAction(ConsultRunbook, RunbookLinks[KontainerDriverNotReady][0])},
}

func getConsultRunbookAction(summaryF string, runbookLink string) string {
	return fmt.Sprintf(summaryF, runbookLink)
}
