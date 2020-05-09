package trigger

import (
	"fmt"

	"gitee.com/openeuler/go-gitee/gitee"
	"k8s.io/test-infra/prow/github"
)

type giteeClient interface {
	AddPRLabel(org, repo string, number int, label string) error
	BotName() (string, error)
	IsCollaborator(org, repo, user string) (bool, error)
	IsMember(org, user string) (bool, error)
	GetRef(org, repo, ref string) (string, error)
	CreatePRComment(owner, repo string, number int, comment string) error
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
	RemovePRLabel(org, repo string, number int, label string) error
	GetPRLabels(org, repo string, number int) ([]gitee.Label, error)
}

var _ githubClient = (*ghclient)(nil)

type ghclient struct {
	giteeClient
}

func (c *ghclient) GetIssueLabels(org, repo string, number int) ([]github.Label, error) {
	var r []github.Label

	v, err := c.GetPRLabels(org, repo, number)
	if err != nil {
		return r, err
	}

	for _, i := range v {
		r = append(r, github.Label{Name: i.Name})
	}
	return r, nil
}

func (c *ghclient) ListIssueComments(org, repo string, number int) ([]github.IssueComment, error) {
	return nil, fmt.Errorf("ListIssueComments is not used in original trigger")
}

func (c *ghclient) CreateComment(org, repo string, number int, comment string) error {
	return c.CreatePRComment(org, repo, number, comment)
}

func (c *ghclient) AddLabel(org, repo string, number int, label string) error {
	return c.AddPRLabel(org, repo, number, label)
}

func (c *ghclient) RemoveLabel(org, repo string, number int, label string) error {
	return c.RemovePRLabel(org, repo, number, label)
}

func (c *ghclient) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	return nil, fmt.Errorf("GetPullRequest is not used in original trigger")
}

func (c *ghclient) CreateStatus(owner, repo, ref string, status github.Status) error {
	return fmt.Errorf("CreateStatus is not used in original trigger")
}

func (c *ghclient) GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error) {
	return nil, fmt.Errorf("GetCombinedStatus is not implemented")
}

func (c *ghclient) DeleteStaleComments(org, repo string, number int, comments []github.IssueComment, isStale func(github.IssueComment) bool) error {
	return fmt.Errorf("DeleteStaleComments is not used in original trigger")
}
