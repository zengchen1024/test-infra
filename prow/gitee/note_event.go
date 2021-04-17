package gitee

import sdk "gitee.com/openeuler/go-gitee/gitee"

const (
	//StatusOpen gitee issue or pr status is open
	StatusOpen = "open"
	//StatusOpen gitee issue or pr status is closed
	StatusClosed = "closed"
)

type NoteEventWrapper struct {
	*sdk.NoteEvent
}

//IsCreatingCommentEvent Determine whether an note event is create a comment
func (ne NoteEventWrapper) IsCreatingCommentEvent() bool {
	return *(ne.Action) == "comment"
}

//GetCommenter Return to the author of the comment
func (ne NoteEventWrapper) GetCommenter() string {
	return ne.Comment.User.Login
}

//GetComment Return to the content of the comment
func (ne NoteEventWrapper) GetComment() string {
	return ne.Comment.Body
}

//GetOrgRepo Return to the org and repo
func (ne NoteEventWrapper) GetOrgRep() (string, string) {
	return ne.Repository.Namespace, ne.Repository.Path
}

//IsPullRequest Determine whether it is a PullRequest
func (ne NoteEventWrapper) IsPullRequest() bool {
	return *(ne.NoteableType) == "PullRequest"
}

//IsIssue Determine whether it is a issue
func (ne NoteEventWrapper) IsIssue() bool {
	return *(ne.NoteableType) == "Issue"
}

type IssueNoteEvent struct {
	NoteEventWrapper
}

//IsIssueClosed whether the status is close  of the issue
func (ne IssueNoteEvent) IsIssueClosed() bool {
	return ne.Issue.State == StatusClosed
}

//IsIssueOpen whether the status is open  of the issue
func (ne IssueNoteEvent) IsIssueOpen() bool {
	return ne.Issue.State == StatusOpen
}

//GetIssueAuthor Return to the author of the issue
func (ne IssueNoteEvent) GetIssueAuthor() string {
	return ne.Issue.User.Login
}

//GetIssueNumber Return to the number of the issue
func (ne IssueNoteEvent) GetIssueNumber() string {
	return ne.Issue.Number
}

type PRNoteEvent struct {
	NoteEventWrapper
}

//GetPRNumber Return to the number of the PR
func (ne PRNoteEvent) GetPRNumber() int {
	return int(ne.PullRequest.Number)
}

//GetPRAuthor Return to the author of the PR
func (ne PRNoteEvent) GetPRAuthor() string {
	return ne.PullRequest.User.Login
}

//IsPROpen whether the status is open  of the pull request
func (ne PRNoteEvent) IsPROpen() bool {
	return ne.PullRequest.State == StatusOpen
}

func NewNoteEventWrapper(e *sdk.NoteEvent) NoteEventWrapper {
	return NoteEventWrapper{NoteEvent: e}
}

func NewIssueNoteEvent(e *sdk.NoteEvent) IssueNoteEvent {
	return IssueNoteEvent{
		NoteEventWrapper: NoteEventWrapper{NoteEvent: e},
	}
}

func NewPRNoteEvent(e *sdk.NoteEvent) PRNoteEvent {
	return PRNoteEvent{
		NoteEventWrapper: NoteEventWrapper{NoteEvent: e},
	}
}
