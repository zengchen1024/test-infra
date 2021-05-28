package approvers

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
)

func SuggestApprovers(currentApprovers, assignees, filenames []string, prNumber, numberOfApprovers int, repoOwner Repo, prAuthor string, allowSelfApprove bool, log *logrus.Entry) []string {
	currentApprovers1 := sets.NewString(currentApprovers...)
	assignees1 := sets.NewString(assignees...)
	if !allowSelfApprove {
		if currentApprovers1.Has(prAuthor) {
			currentApprovers1.Delete(prAuthor)
		}
		if assignees1.Has(prAuthor) {
			assignees1.Delete(prAuthor)
		}
	}

	owner := NewOwners(log, filenames, repoOwner, int64(prNumber))
	if numberOfApprovers == 1 {
		ap := NewApprovers(owner)
		ap.AddAssignees(assignees1.List()...)
		for item := range currentApprovers1 {
			ap.AddApprover(item, "", false)
		}
		return ap.GetCCs()
	}

	approversAndAssignees := currentApprovers1.Union(assignees1)
	randomizedApprovers := owner.GetShuffledApprovers()
	leafReverseMap := owner.GetReverseMap(owner.GetLeafApprovers())
	if !allowSelfApprove {
		if _, ok := leafReverseMap[prAuthor]; ok {
			delete(leafReverseMap, prAuthor)
			randomizedApprovers = removeSliceElement(randomizedApprovers, prAuthor)
		}

	}
	suggested := keepCoveringApprovers(
		owner, leafReverseMap, approversAndAssignees,
		randomizedApprovers, numberOfApprovers,
	)

	approversAndSuggested := currentApprovers1.Union(suggested)
	everyone := approversAndSuggested.Union(assignees1)
	fullReverseMap := owner.GetReverseMap(owner.GetApprovers())
	if !allowSelfApprove {
		if _, ok := fullReverseMap[prAuthor]; ok {
			delete(fullReverseMap, prAuthor)
		}
	}
	keepAssignees := keepCoveringApprovers(
		owner, fullReverseMap, approversAndSuggested,
		everyone.List(), numberOfApprovers,
	)

	return suggested.Union(keepAssignees).List()
}

func removeSliceElement(v []string, target string) []string {
	for i, item := range v {
		if item == target {
			n := len(v) - 1
			v[i] = v[n]
			return v[:n]
		}
	}
	return v
}

func keepCoveringApprovers(owner Owners, reverseMap map[string]sets.String, knownApprovers sets.String, potentialApprovers []string, numberOfApprovers int) sets.String {
	f := func(ap Approvers) sets.String {
		excludedApprovers := sets.String{}
		unapproved := sets.String{}
		files := ap.GetFilesApprovers()
		for f, v := range files {
			if len(v) < numberOfApprovers {
				unapproved.Insert(f)
				for k := range v {
					excludedApprovers.Insert(k)
				}
			}
		}
		if len(unapproved) == 0 {
			return sets.NewString()
		}

		candidates := []string{}
		for _, item := range potentialApprovers {
			if !excludedApprovers.Has(item) {
				candidates = append(candidates, item)
			}
		}

		keptApprovers := sets.NewString()
		for _, suggestedApprover := range owner.GetSuggestedApprovers(reverseMap, candidates).List() {
			if reverseMap[suggestedApprover].Intersection(unapproved).Len() != 0 {
				keptApprovers.Insert(suggestedApprover)
			}
		}

		return keptApprovers
	}

	ap := NewApprovers(owner)
	for k := range knownApprovers {
		ap.AddApprover(k, "", false)
	}

	r := sets.NewString()
	for i := 0; i < numberOfApprovers; i++ {
		v := f(ap)
		if len(v) == 0 {
			break
		}

		r = r.Union(v)

		for k := range v {
			ap.AddApprover(k, "", false)
		}
	}

	return r
}
