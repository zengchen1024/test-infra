package lifecycle

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/gitee"
)

func reopenIssue(gc giteeClient, log *logrus.Entry, e *sdk.NoteEvent) error {
	ne := gitee.NewIssueNoteEvent(e)
	org, repo := ne.GetOrgRep()
	commenter := ne.GetCommenter()
	number := ne.GetIssueNumber()

	if ne.GetIssueAuthor() != commenter && !isCollaborator(org, repo, commenter, gc, log) {
		resp := response(
			ne.NoteEventWrapper,
			"You can't reopen an issue unless you are the author of it or a collaborator.",
		)
		return gc.CreateGiteeIssueComment(org, repo, number, resp)
	}

	if err := gc.ReopenIssue(org, repo, number); err != nil {
		return err
	}

	// Add a comment after reopening the pr to leave an audit trail of who asked to reopen it.
	return gc.CreateGiteeIssueComment(org, repo, number, response(ne.NoteEventWrapper, "Reopened this issue."))
}
