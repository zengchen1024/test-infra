package trigger

import (
	"fmt"
	"sort"

	sdk "gitee.com/openeuler/go-gitee/gitee"

	"k8s.io/test-infra/prow/gitee"
	"k8s.io/test-infra/prow/github"
	reporter "k8s.io/test-infra/prow/job-reporter/gitee"
	"k8s.io/test-infra/prow/labels"
)

type giteeClient interface {
	AddPRLabel(org, repo string, number int, label string) error
	BotName() (string, error)
	IsCollaborator(org, repo, user string) (bool, error)
	IsMember(org, user string) (bool, error)
	GetRef(org, repo, ref string) (string, error)
	CreatePRComment(owner, repo string, number int, comment string) error
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
	GetGiteePullRequest(org, repo string, number int) (sdk.PullRequest, error)
	RemovePRLabel(org, repo string, number int, label string) error
	GetPRLabels(org, repo string, number int) ([]sdk.Label, error)
	ListPRComments(org, repo string, number int) ([]sdk.PullRequestComments, error)
	GetPullRequests(org, repo string, opts gitee.ListPullRequestOpt) ([]sdk.PullRequest, error)
}

type ghclient struct {
	giteeClient
	prNumber int
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
	var r []github.IssueComment

	v, err := c.ListPRComments(org, repo, number)
	if err != nil {
		return r, err
	}

	for _, i := range v {
		r = append(r, gitee.ConvertGiteePRComment(i))
	}

	sort.SliceStable(r, func(i, j int) bool {
		return r[i].CreatedAt.Before(r[j].CreatedAt)
	})

	return r, nil
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
	v, err := c.GetGiteePullRequest(org, repo, number)
	if err != nil {
		return nil, err
	}

	return gitee.ConvertGiteePR(&v), nil
}

func (c *ghclient) CreateStatus(owner, repo, ref string, status github.Status) error {
	return fmt.Errorf("CreateStatus is not used in original trigger")
}

func (c *ghclient) GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error) {
	comments, err := c.ListIssueComments(org, repo, c.prNumber)
	if err != nil {
		return nil, err
	}

	botname, err := c.BotName()
	if err != nil {
		return nil, err
	}

	status := reporter.ParseCombinedStatus(botname, ref, comments)

	return &github.CombinedStatus{Statuses: status}, nil
}

func (c *ghclient) DeleteStaleComments(org, repo string, number int, comments []github.IssueComment, isStale func(github.IssueComment) bool) error {
	return fmt.Errorf("DeleteStaleComments is not used in original trigger")
}

func (c *ghclient) hasApprovedPR(org, repo string) bool {
	opt := gitee.ListPullRequestOpt{
		State:  gitee.StatusOpen,
		Labels: []string{labels.Approved},
	}

	if v, err := c.GetPullRequests(org, repo, opt); err == nil {
		return len(v) > 0
	}

	return false
}
