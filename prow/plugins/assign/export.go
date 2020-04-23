package assign

import (
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
)

func Handle(e github.GenericCommentEvent, gc githubClient, log *logrus.Entry) error {
	err := handle(newAssignHandler(e, gc, log))
	if e.IsPR {
		err = combineErrors(err, handle(newReviewHandler(e, gc, log)))
	}
	return err
}
