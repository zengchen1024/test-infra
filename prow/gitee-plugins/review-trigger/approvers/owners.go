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
	"math/rand"
	"sort"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Repo allows querying and interacting with OWNERS information in a repo.
type Repo interface {
	Approvers(path string) sets.String
	LeafApprovers(path string) sets.String
}

// NewOwners consturcts a new Owners instance. filenames is the slice of files changed.
func NewOwners(log *logrus.Entry, filenames []string, r Repo, s int64) Owners {
	return Owners{filenames: filenames, repo: r, seed: s, log: log}
}

// Owners provides functionality related to owners of a specific code change.
type Owners struct {
	filenames []string
	repo      Repo
	seed      int64

	log *logrus.Entry
}

// GetApprovers returns a map from ownersFiles -> people that are approvers in them
func (o Owners) GetApprovers() map[string]sets.String {
	ownersToApprovers := map[string]sets.String{}
	for _, fn := range o.filenames {
		ownersToApprovers[fn] = o.repo.Approvers(fn)
	}
	return ownersToApprovers
}

// GetLeafApprovers returns a map from ownersFiles -> people that are approvers in them (only the leaf)
func (o Owners) GetLeafApprovers() map[string]sets.String {
	ownersToApprovers := map[string]sets.String{}
	for _, fn := range o.filenames {
		ownersToApprovers[fn] = o.repo.LeafApprovers(fn)
	}
	return ownersToApprovers
}

// GetAllPotentialApprovers returns the people from relevant owners files needed to get the PR approved
func (o Owners) GetAllPotentialApprovers() []string {
	approversOnly := []string{}
	for _, approverList := range o.GetLeafApprovers() {
		approversOnly = append(approversOnly, approverList.List()...)
	}

	if len(approversOnly) == 0 {
		o.log.Warn("No potential approvers exist. Does the repo have OWNERS files?")
	}

	sort.Strings(approversOnly)
	return approversOnly
}

// temporaryUnapprovedFiles returns the list of files that wouldn't be
// approved by the given set of approvers.
func (o Owners) temporaryUnapprovedFiles(approvers sets.String) sets.String {
	ap := NewApprovers(o)
	ap.AddApprovers(approvers.List())
	return ap.UnapprovedFiles()
}

// KeepCoveringApprovers finds who we should keep as suggested approvers given a pre-selection
// knownApprovers must be a subset of potentialApprovers.
func (o Owners) KeepCoveringApprovers(reverseMap map[string]sets.String, knownApprovers sets.String, potentialApprovers []string) sets.String {
	if len(potentialApprovers) == 0 {
		o.log.Debug("No potential approvers exist to filter for relevance. Does this repo have OWNERS files?")
	}

	unapproved := o.temporaryUnapprovedFiles(knownApprovers)

	keptApprovers := sets.NewString()
	for _, suggestedApprover := range o.GetSuggestedApprovers(reverseMap, potentialApprovers).List() {
		if reverseMap[suggestedApprover].Intersection(unapproved).Len() != 0 {
			keptApprovers.Insert(suggestedApprover)
		}
	}
	return keptApprovers
}

func findMostCoveringApprover(allApprovers []string, reverseMap map[string]sets.String, unapproved sets.String) string {
	maxCovered := 0
	var bestPerson string
	for _, approver := range allApprovers {
		filesCanApprove := reverseMap[approver]
		if n := filesCanApprove.Intersection(unapproved).Len(); n > maxCovered {
			maxCovered = n
			bestPerson = approver
		}
	}
	return bestPerson
}

// GetSuggestedApprovers solves the exact cover problem, finding an approver capable of
// approving every OWNERS file in the PR
func (o Owners) GetSuggestedApprovers(reverseMap map[string]sets.String, potentialApprovers []string) sets.String {
	ap := NewApprovers(o)
	for !ap.RequirementsMet() {
		newApprover := findMostCoveringApprover(potentialApprovers, reverseMap, ap.UnapprovedFiles())
		if newApprover == "" {
			o.log.Warnf("Couldn't find/suggest approvers for each files. Unapproved: %q", ap.UnapprovedFiles().List())
			return ap.GetCurrentApproversSet()
		}
		ap.AddApprover(newApprover)
	}
	return ap.GetCurrentApproversSet()
}

// GetShuffledApprovers shuffles the potential approvers so that we don't
// always suggest the same people.
func (o Owners) GetShuffledApprovers() []string {
	approversList := o.GetAllPotentialApprovers()
	order := rand.New(rand.NewSource(o.seed)).Perm(len(approversList))

	people := make([]string, 0, len(approversList))
	for _, i := range order {
		people = append(people, approversList[i])
	}
	return people
}

// GetReverseMap returns a map from people -> OWNERS files for which they are an approver
func GetReverseMap(approvers map[string]sets.String) map[string]sets.String {
	approverOwnersfiles := map[string]sets.String{}
	for ownersFile, approvers := range approvers {
		for approver := range approvers {
			if _, ok := approverOwnersfiles[approver]; ok {
				approverOwnersfiles[approver].Insert(ownersFile)
			} else {
				approverOwnersfiles[approver] = sets.NewString(ownersFile)
			}
		}
	}
	return approverOwnersfiles
}
