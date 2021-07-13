package approvers

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

type approversHelper struct {
	fileApprovers map[string]sets.String
	approvers     sets.String
}

func newApproversHelper(o Owners, as []string) *approversHelper {
	return &approversHelper{
		fileApprovers: o.GetApprovers(),
		approvers:     sets.NewString(as...),
	}
}

func (ap *approversHelper) addApprover(a string) {
	ap.approvers.Insert(a)
}

func (ap approversHelper) requirementsMet() bool {
	return len(ap.fileApprovers) > 0 && ap.unapprovedFiles().Len() == 0
}

func (ap approversHelper) unapprovedFiles() sets.String {
	unapproved := sets.NewString()
	for fn, approvers := range ap.getFilesApprovers() {
		if approvers.Len() == 0 {
			unapproved.Insert(fn)
		}
	}
	return unapproved
}

func (ap approversHelper) getFilesApprovers() map[string]sets.String {
	currentApprovers := ap.getCurrentApproversSet()

	filesApprovers := map[string]sets.String{}
	for fn, potentialApprovers := range ap.fileApprovers {
		filesApprovers[fn] = currentApprovers.Intersection(potentialApprovers)
	}
	return filesApprovers
}

func (ap approversHelper) getCurrentApproversSet() sets.String {
	return ap.approvers
}
