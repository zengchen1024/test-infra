/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package approvers

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

// NewApprovers create a new "Approvers" with no approval.
func NewApprovers(owners Owners) Approvers {
	return Approvers{
		owners:    owners,
		approvers: sets.NewString(),
		assignees: sets.NewString(),
	}
}

// Approvers is struct that provide functionality with regard to approvals of a specific
// code change.
type Approvers struct {
	owners    Owners
	approvers sets.String // The keys of this map are normalized to lowercase.
	assignees sets.String
}

// GetCCs gets the list of suggested approvers for a pull-request.  It
// now considers current assignees as potential approvers. Here is how
// it works:
// - We find suggested approvers from all potential approvers, but
// remove those that are not useful considering current approvers and
// assignees. This only uses leaf approvers to find the closest
// approvers to the changes.
// - We find a subset of suggested approvers from current
// approvers, suggested approvers and assignees, but we remove those
// that are not useful considering suggested approvers and current
// approvers. This uses the full approvers list, and will result in root
// approvers to be suggested when they are assigned.
// We return the union of the two sets: suggested and suggested
// assignees.
// The goal of this second step is to only keep the assignees that are
// the most useful.
func (ap Approvers) GetCCs() []string {
	currentApprovers := ap.GetCurrentApproversSet()

	approversAndAssignees := currentApprovers.Union(ap.assignees)
	randomizedApprovers := ap.owners.GetShuffledApprovers()
	leafReverseMap := GetReverseMap(ap.owners.GetLeafApprovers())
	suggested := ap.owners.KeepCoveringApprovers(leafReverseMap, approversAndAssignees, randomizedApprovers)

	approversAndSuggested := currentApprovers.Union(suggested)
	everyone := approversAndSuggested.Union(ap.assignees)
	fullReverseMap := GetReverseMap(ap.owners.GetApprovers())
	keepAssignees := ap.owners.KeepCoveringApprovers(fullReverseMap, approversAndSuggested, everyone.UnsortedList())

	return suggested.Union(keepAssignees).List()
}

// GetCurrentApproversSet returns the set of approvers (login only, normalized to lower case)
func (ap Approvers) GetCurrentApproversSet() sets.String {
	return ap.approvers
}

// AddApprover adds a new Approver
func (ap *Approvers) AddApprover(login string) {
	ap.approvers.Insert(login)
}

// AddApprovers adds approvers
func (ap *Approvers) AddApprovers(login []string) {
	ap.approvers.Insert(login...)
}

// UnapprovedFiles returns owners files that still need approval
func (ap Approvers) UnapprovedFiles() sets.String {
	unapproved := sets.NewString()
	for fn, approvers := range ap.GetFilesApprovers() {
		if approvers.Len() == 0 {
			unapproved.Insert(fn)
		}
	}
	return unapproved
}

// GetFilesApprovers returns a map from files -> list of current approvers.
func (ap Approvers) GetFilesApprovers() map[string]sets.String {
	currentApprovers := ap.GetCurrentApproversSetCased()

	filesApprovers := map[string]sets.String{}
	for fn, potentialApprovers := range ap.owners.GetApprovers() {
		filesApprovers[fn] = currentApprovers.Intersection(potentialApprovers)
	}
	return filesApprovers
}

// GetCurrentApproversSetCased returns the set of approvers logins with the original cases.
func (ap Approvers) GetCurrentApproversSetCased() sets.String {
	return ap.approvers
}

// AreFilesApproved returns a bool indicating whether or not OWNERS files associated with
// the PR are approved.  A PR with no OWNERS files is not considered approved.
func (ap Approvers) AreFilesApproved() bool {
	return len(ap.owners.filenames) != 0 && ap.UnapprovedFiles().Len() == 0
}

// RequirementsMet returns a bool indicating whether the PR has met all approval requirements:
// - all OWNERS files associated with the PR have been approved AND
func (ap Approvers) RequirementsMet() bool {
	return ap.AreFilesApproved()
}

// AddAssignees adds assignees to the list
func (ap *Approvers) AddAssignees(logins ...string) {
	ap.assignees.Insert(logins...)
}
