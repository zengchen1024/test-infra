package approvers

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

type StaticApprovers struct {
	fileApprovers map[string]sets.String
	approvers     sets.String
}

func NewStaticApprovers(o Owners, as []string) *StaticApprovers {
	return &StaticApprovers{
		fileApprovers: o.GetApprovers(),
		approvers:     sets.NewString(as...),
	}
}

func (ap *StaticApprovers) AddApprover(as ...string) {
	ap.approvers.Insert(as...)
}

func (ap StaticApprovers) requirementsMet() bool {
	return len(ap.fileApprovers) > 0 && ap.unapprovedFiles().Len() == 0
}

func (ap StaticApprovers) unapprovedFiles() sets.String {
	unapproved := sets.NewString()
	for fn, approvers := range ap.GetFilesApprovers() {
		if approvers.Len() == 0 {
			unapproved.Insert(fn)
		}
	}
	return unapproved
}

func (ap StaticApprovers) GetFilesApprovers() map[string]sets.String {
	currentApprovers := ap.getCurrentApproversSet()

	filesApprovers := map[string]sets.String{}
	for fn, potentialApprovers := range ap.fileApprovers {
		filesApprovers[fn] = currentApprovers.Intersection(potentialApprovers)
	}
	return filesApprovers
}

func (ap StaticApprovers) getCurrentApproversSet() sets.String {
	return ap.approvers
}
