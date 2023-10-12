// Copyright (C) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/verrazzano/pkg/diff"
	vmcontrollerv1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

type (
	DataStreams struct {
		DataStreams []DataStream `json:"data_streams,omitempty"`
	}
	DataStream struct {
		Name           string    `json:"name,omitempty"`
		TimeStampField TimeStamp `json:"timestamp_field,omitempty"`
		Indices        []Index   `json:"indices,omitempty"`
		Generation     int       `json:"generation,omitempty"`
		Status         string    `json:"status,omitempty"`
		Template       string    `json:"template,omitempty"`
	}
	TimeStamp struct {
		Name string `json:"name,omitempty"`
	}
	Index struct {
		Name string `json:"index_name,omitempty"`
		UUID string `json:"index_uuid,omitempty"`
	}
	PolicyList struct {
		Policies      []ISMPolicy `json:"policies"`
		TotalPolicies int         `json:"total_policies"`
	}

	ISMPolicy struct {
		ID             *string      `json:"_id,omitempty"`
		PrimaryTerm    *int         `json:"_primary_term,omitempty"`
		SequenceNumber *int         `json:"_seq_no,omitempty"`
		Status         *int         `json:"status,omitempty"`
		Policy         InlinePolicy `json:"policy"`
	}

	InlinePolicy struct {
		DefaultState string        `json:"default_state"`
		Description  string        `json:"description"`
		States       []PolicyState `json:"states"`
		ISMTemplate  []ISMTemplate `json:"ism_template"`
	}

	ISMTemplate struct {
		IndexPatterns []string `json:"index_patterns"`
		Priority      int      `json:"priority"`
	}

	PolicyState struct {
		Name        string                   `json:"name"`
		Actions     []map[string]interface{} `json:"actions,omitempty"`
		Transitions []PolicyTransition       `json:"transitions,omitempty"`
	}

	PolicyTransition struct {
		StateName  string            `json:"state_name"`
		Conditions *PolicyConditions `json:"conditions,omitempty"`
	}

	PolicyConditions struct {
		MinIndexAge    string `json:"min_index_age,omitempty"`
		MinRolloverAge string `json:"min_rollover_age,omitempty"`
	}
)

const (
	minIndexAgeKey = "min_index_age"

	// Default amount of time before a policy-managed index is deleted
	defaultMinIndexAge = "7d"
	// Default amount of time before a policy-managed index is rolled over
	defaultRolloverIndexAge = "1d"
	// Descriptor to identify policies as being managed by the integration controller
	operatorManagedPolicy       = "Verrazzano-managed"
	systemDefaultPolicyFileName = "vz-system-default-ISM-policy.json"
	appDefaultPolicyFileName    = "vz-application-default-ISM-policy.json"
	defaultPolicyPath           = "opensearch-operator"
	systemDefaultPolicy         = "vz-system"
	applicationDefaultPolicy    = "vz-application"
)

var (
	defaultISMPoliciesMap = map[string]string{systemDefaultPolicy: systemDefaultPolicyFileName, applicationDefaultPolicy: appDefaultPolicyFileName}
)

// createISMPolicy creates an ISM policy if it does not exist, else the policy will be updated.
// If the policy already exists and its spec matches the VZ policy spec, no update will be issued
func (o *OSClient) createISMPolicy(opensearchEndpoint string, policy vmcontrollerv1.IndexManagementPolicy) error {
	policyURL := fmt.Sprintf("%s/_plugins/_ism/policies/%s", opensearchEndpoint, policy.PolicyName)
	existingPolicy, err := o.getPolicyByName(policyURL)
	if err != nil {
		return err
	}
	updatedPolicy, err := o.putUpdatedPolicy(opensearchEndpoint, policy.PolicyName, toISMPolicy(&policy), existingPolicy)
	if err != nil {
		return err
	}
	return o.addPolicyToExistingIndices(opensearchEndpoint, &policy, updatedPolicy)
}

func (o *OSClient) getPolicyByName(policyURL string) (*ISMPolicy, error) {
	req, err := http.NewRequest("GET", policyURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.DoHTTP(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	existingPolicy := &ISMPolicy{}
	existingPolicy.Status = &resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		return existingPolicy, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(existingPolicy); err != nil {
		return nil, err
	}
	return existingPolicy, nil
}

// putUpdatedPolicy updates a policy in place, if the update is required. If no update was necessary, the returned
// ISMPolicy will be nil.
func (o *OSClient) putUpdatedPolicy(opensearchEndpoint string, policyName string, policy *ISMPolicy, existingPolicy *ISMPolicy) (*ISMPolicy, error) {
	if !policyNeedsUpdate(policy, existingPolicy) {
		return nil, nil
	}
	payload, err := serializeIndexManagementPolicy(policy)
	if err != nil {
		return nil, err
	}

	var url string
	var statusCode int
	existingPolicyStatus := *existingPolicy.Status
	switch existingPolicyStatus {
	case http.StatusOK: // The policy exists and must be updated in place if it has changed
		url = fmt.Sprintf("%s/_plugins/_ism/policies/%s?if_seq_no=%d&if_primary_term=%d",
			opensearchEndpoint,
			policyName,
			*existingPolicy.SequenceNumber,
			*existingPolicy.PrimaryTerm,
		)
		statusCode = http.StatusOK
	case http.StatusNotFound: // The policy doesn't exist and must be updated
		url = fmt.Sprintf("%s/_plugins/_ism/policies/%s", opensearchEndpoint, policyName)
		statusCode = http.StatusCreated
	default:
		return nil, fmt.Errorf("invalid status when fetching ISM Policy %s: %d", policyName, existingPolicy.Status)
	}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Add(contentTypeHeader, applicationJSON)
	resp, err := o.DoHTTP(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != statusCode {
		return nil, fmt.Errorf("got status code %d when updating policy %s, expected %d", resp.StatusCode, policyName, statusCode)
	}
	updatedISMPolicy := &ISMPolicy{}
	err = json.NewDecoder(resp.Body).Decode(updatedISMPolicy)
	if err != nil {
		return nil, err
	}

	return updatedISMPolicy, nil
}

// addPolicyToExistingIndices updates any pre-existing cluster indices to be managed by the ISMPolicy
func (o *OSClient) addPolicyToExistingIndices(opensearchEndpoint string, policy *vmcontrollerv1.IndexManagementPolicy, updatedPolicy *ISMPolicy) error {
	// If no policy was updated, then there is nothing to do
	if updatedPolicy == nil {
		return nil
	}
	url := fmt.Sprintf("%s/_plugins/_ism/add/%s", opensearchEndpoint, policy.IndexPattern)
	body := strings.NewReader(fmt.Sprintf(`{"policy_id": "%s"}`, *updatedPolicy.ID))
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Add(contentTypeHeader, applicationJSON)
	resp, err := o.DoHTTP(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status code %d when updating indicies for policy %s", resp.StatusCode, policy.PolicyName)
	}
	return nil
}

func (o *OSClient) cleanupPolicies(opensearchEndpoint string, policies []vmcontrollerv1.IndexManagementPolicy) error {
	policyList, err := o.getAllPolicies(opensearchEndpoint)
	if err != nil {
		return err
	}

	expectedPolicyMap := map[string]bool{}
	for _, policy := range policies {
		expectedPolicyMap[policy.PolicyName] = true
	}

	// A policy is eligible for deletion if it is marked as operator managed, but the VZ no longer
	// has a policy entry for it
	for _, policy := range policyList.Policies {
		if isEligibleForDeletion(policy, expectedPolicyMap) {
			if _, err := o.deletePolicy(opensearchEndpoint, *policy.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *OSClient) getAllPolicies(opensearchEndpoint string) (*PolicyList, error) {
	url := fmt.Sprintf("%s/_plugins/_ism/policies", opensearchEndpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.DoHTTP(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got status code %d when querying policies for cleanup", resp.StatusCode)
	}
	policies := &PolicyList{}
	if err := json.NewDecoder(resp.Body).Decode(policies); err != nil {
		return nil, err
	}
	return policies, nil
}

func (o *OSClient) deletePolicy(opensearchEndpoint, policyName string) (*http.Response, error) {
	url := fmt.Sprintf("%s/_plugins/_ism/policies/%s", opensearchEndpoint, policyName)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.DoHTTP(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("got status code %d when deleting policy %s", resp.StatusCode, policyName)
	}
	return resp, nil
}

// updateISMPolicyFromFile creates or updates the ISM policy from the given json file.
// If ISM policy doesn't exist, it will create new. Otherwise, it'll create one.
func (o *OSClient) updateISMPolicy(openSearchEndpoint string, policyName string, policy *ISMPolicy) (*ISMPolicy, error) {
	existingPolicyURL := fmt.Sprintf("%s/_plugins/_ism/policies/%s", openSearchEndpoint, policyName)
	existingPolicy, err := o.getPolicyByName(existingPolicyURL)
	if err != nil {
		return nil, err
	}
	return o.putUpdatedPolicy(openSearchEndpoint, policyName, policy, existingPolicy)
}

// createOrUpdateDefaultISMPolicy creates the default ISM policies if not exist, else the policies will be updated.
func (o *OSClient) createOrUpdateDefaultISMPolicy(log vzlog.VerrazzanoLogger, openSearchEndpoint string) ([]*ISMPolicy, error) {
	var defaultPolicies []*ISMPolicy
	allPolicyList, err := o.getAllPolicies(openSearchEndpoint)
	if err != nil {
		return nil, err
	}
	log.Debugf("os system has %v policies", len(allPolicyList.Policies))
	for policyName, policyFile := range defaultISMPoliciesMap {
		policy, err := getISMPolicyFromFile(policyFile)
		if err != nil {
			return nil, err
		}
		log.Debugf("checking if custom policy exists for %s from file %s", policyName, policyFile)
		if !o.isCustomPolicyExists(log, policy, policyName, allPolicyList.Policies) {
			log.Debugf("creating default policy for policy %s", policyName)
			createdPolicy, err := o.updateISMPolicy(openSearchEndpoint, policyName, policy)
			if err != nil {
				return defaultPolicies, err
			}
			// When the default policy is created or updated
			// Add default policy to the current write index of the data stream
			if createdPolicy != nil {
				err = o.addDefaultPolicyToDataStream(log, openSearchEndpoint, policyName, policy)
				if err != nil {
					return defaultPolicies, err
				}
			}
			defaultPolicies = append(defaultPolicies, createdPolicy)
		}
	}
	return defaultPolicies, nil
}

// addDefaultPolicyToDataStream adds the default policy to the current write index of the data stream
func (o *OSClient) addDefaultPolicyToDataStream(log vzlog.VerrazzanoLogger, openSearchEndpoint, policyName string, policy *ISMPolicy) error {
	if len(policy.Policy.ISMTemplate) <= 0 {
		return fmt.Errorf("no index template defined in policy %s", policyName)
	}
	indexPatterns := policy.Policy.ISMTemplate[0].IndexPatterns

	for _, pattern := range indexPatterns {
		// Get current write index for each data stream corresponding to the pattern
		// Data streams and therefore write index will be multiple for application and only one for system
		writeIndices, err := o.getWriteIndexForDataStream(log, openSearchEndpoint, pattern)
		if err != nil {
			return fmt.Errorf("failed to get the write index for %s: %v", pattern, err)
		}

		for _, index := range writeIndices {
			// Check if the index is currently being managed by our default policy or no policy at all
			// If yes then attach the default policy to the index
			ok, err := o.shouldAddOrRemoveDefaultPolicy(openSearchEndpoint, index, policyName)
			if err != nil {
				return err
			}
			if ok {
				err = o.removePolicyForIndex(openSearchEndpoint, index)
				if err != nil {
					return fmt.Errorf("failed to remove policy for index %s", index)
				}
				err = o.addPolicyForIndex(openSearchEndpoint, index, policyName)
				if err != nil {
					return fmt.Errorf("failed to add default policy for index %s", index)
				}
				log.Debugf("Added default policy %s to index %s", policyName, index)
			}
		}
	}
	return nil
}

// shouldAddOrRemoveDefaultPolicy returns true if the policy is being managed by the default policy
// or if it is not being managed by any policy at all
func (o *OSClient) shouldAddOrRemoveDefaultPolicy(openSearchEndpoint, index, policyID string) (bool, error) {
	url := fmt.Sprintf("%s/_plugins/_ism/explain/%s", openSearchEndpoint, index)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	resp, err := o.DoHTTP(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to get attached policy status for index %s", index)
	}
	defer resp.Body.Close()
	var indexInfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&indexInfo)
	if err != nil {
		return false, err
	}

	if indexInfo[index] != nil {
		policy := indexInfo[index].(map[string]interface{})["index.opendistro.index_state_management.policy_id"]
		// If the index is not being managed by any policy or is being managed by our policy return true
		if policy == nil || policy == policyID {
			return true, nil
		}
	}
	return false, nil
}

// addPolicyForIndex attaches an ISM policy to an index
func (o *OSClient) addPolicyForIndex(openSearchEndpoint, index, policyID string) error {
	url := fmt.Sprintf("%s/_plugins/_ism/add/%s", openSearchEndpoint, index)
	body := strings.NewReader(fmt.Sprintf(`{"policy_id": "%s"}`, policyID))
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Add(contentTypeHeader, applicationJSON)
	resp, err := o.DoHTTP(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status code %d when adding policy for index %s", resp.StatusCode, index)
	}
	return nil
}

// removePolicyForIndex removes the ISM policy attached to an index
func (o *OSClient) removePolicyForIndex(openSearchEndpoint, index string) error {
	url := fmt.Sprintf("%s/_plugins/_ism/remove/%s", openSearchEndpoint, index)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	resp, err := o.DoHTTP(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status code %d when removing policy from index %s", resp.StatusCode, index)
	}
	return nil
}

// getWriteIndexForDataStream returns the current write indices for a given data stream pattern
func (o *OSClient) getWriteIndexForDataStream(log vzlog.VerrazzanoLogger, openSearchEndpoint, pattern string) ([]string, error) {
	url := fmt.Sprintf("%s/_data_stream/%s", openSearchEndpoint, pattern)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.DoHTTP(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Do not return an error if the data stream doesn't exist yet
	if resp.StatusCode == http.StatusNotFound {
		log.Infof("Couldn't find data stream %s when creating default policy", pattern)
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got status code %d when fecthing data streams %s", resp.StatusCode, pattern)
	}
	dataStreams := &DataStreams{}
	err = json.NewDecoder(resp.Body).Decode(dataStreams)
	if err != nil {
		return nil, err
	}

	var writeIndex []string
	for _, dataStream := range dataStreams.DataStreams {
		// Current write index for a data stream is the last index in the indices list
		indices := dataStream.Indices
		size := len(indices)
		if size > 0 {
			writeIndex = append(writeIndex, indices[size-1].Name)
		}
	}
	return writeIndex, nil
}

func isEligibleForDeletion(policy ISMPolicy, expectedPolicyMap map[string]bool) bool {
	return policy.Policy.Description == operatorManagedPolicy &&
		!expectedPolicyMap[*policy.ID]
}

// policyNeedsUpdate returns true if the policy document has changed
func policyNeedsUpdate(policy *ISMPolicy, existingPolicy *ISMPolicy) bool {
	newPolicyDocument := policy.Policy
	oldPolicyDocument := existingPolicy.Policy
	return newPolicyDocument.DefaultState != oldPolicyDocument.DefaultState ||
		newPolicyDocument.Description != oldPolicyDocument.Description ||
		diff.Diff(newPolicyDocument.States, oldPolicyDocument.States) != "" ||
		diff.Diff(newPolicyDocument.ISMTemplate, oldPolicyDocument.ISMTemplate) != ""
}

func createRolloverAction(rollover *vmcontrollerv1.RolloverPolicy) map[string]interface{} {
	rolloverAction := map[string]interface{}{}
	if rollover.MinDocCount != nil {
		rolloverAction["min_doc_count"] = *rollover.MinDocCount
	}
	if rollover.MinSize != nil {
		rolloverAction["min_size"] = *rollover.MinSize
	}
	var rolloverMinIndexAge = defaultRolloverIndexAge
	if rollover.MinIndexAge != nil {
		rolloverMinIndexAge = *rollover.MinIndexAge
	}
	rolloverAction[minIndexAgeKey] = rolloverMinIndexAge
	return rolloverAction
}

func serializeIndexManagementPolicy(policy *ISMPolicy) ([]byte, error) {
	return json.Marshal(policy)
}

func toISMPolicy(policy *vmcontrollerv1.IndexManagementPolicy) *ISMPolicy {
	rolloverAction := map[string]interface{}{
		"rollover": createRolloverAction(&policy.Rollover),
	}
	var minIndexAge = defaultMinIndexAge
	if policy.MinIndexAge != nil {
		minIndexAge = *policy.MinIndexAge
	}

	return &ISMPolicy{
		Policy: InlinePolicy{
			DefaultState: "ingest",
			Description:  operatorManagedPolicy,
			ISMTemplate: []ISMTemplate{
				{
					Priority: 1,
					IndexPatterns: []string{
						policy.IndexPattern,
					},
				},
			},
			States: []PolicyState{
				{
					Name: "ingest",
					Actions: []map[string]interface{}{
						rolloverAction,
					},
					Transitions: []PolicyTransition{
						{
							StateName: "delete",
							Conditions: &PolicyConditions{
								MinIndexAge: minIndexAge,
							},
						},
					},
				},
				{
					Name: "delete",
					Actions: []map[string]interface{}{
						{
							"delete": map[string]interface{}{},
						},
					},
					Transitions: []PolicyTransition{},
				},
			},
		},
	}
}

// getISMPolicyFromFile reads the given json file and return the ISMPolicy object after unmarshalling.
func getISMPolicyFromFile(policyFileName string) (*ISMPolicy, error) {
	policypath := filepath.Join(config.GetThirdPartyManifestsDir(), defaultPolicyPath)
	policyBytes, err := os.ReadFile(policypath + "/" + policyFileName)
	if err != nil {
		return nil, err
	}
	var policy ISMPolicy
	err = json.Unmarshal(policyBytes, &policy)
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (o *OSClient) isCustomPolicyExists(log vzlog.VerrazzanoLogger, searchPolicy *ISMPolicy, searchPolicyName string, policyList []ISMPolicy) bool {
	for _, policy := range policyList {
		if *policy.ID != searchPolicyName && policy.Policy.ISMTemplate[0].Priority == searchPolicy.Policy.ISMTemplate[0].Priority && isItemAlreadyExists(log, policy.Policy.ISMTemplate[0].IndexPatterns, searchPolicy.Policy.ISMTemplate[0].IndexPatterns) {
			log.Debugf("custom policy exists for policy %s", searchPolicyName)
			return true
		}
	}
	return false
}
func isItemAlreadyExists(log vzlog.VerrazzanoLogger, allListPolicyPatterns []string, subListPolicyPattern []string) bool {
	matched := false
	log.Debugf("searching for index pattern %s in all ISM policies %s", subListPolicyPattern, allListPolicyPatterns)
	for _, al := range allListPolicyPatterns {
		for _, sl := range subListPolicyPattern {
			if al == sl {
				matched = true
				break
			}
		}
	}
	return matched
}
