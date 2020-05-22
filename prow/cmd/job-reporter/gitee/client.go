package gitee

import (
	"sort"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"k8s.io/test-infra/prow/gitee"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/report"
)

type giteeClient interface {
	BotName() (string, error)
	ListPRComments(org, repo string, number int) ([]sdk.PullRequestComments, error)
	CreatePRComment(org, repo string, number int, comment string) error
	DeletePRComment(org, repo string, ID int) error
	UpdatePRComment(org, repo string, commentID int, comment string) error
}

var _ report.GitHubClient = (*ghclient)(nil)

type ghclient struct {
	giteeClient
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

func (c *ghclient) CreateComment(owner, repo string, number int, comment string) error {
	return c.CreatePRComment(owner, repo, number, comment)
}

func (c *ghclient) DeleteComment(org, repo string, id int) error {
	return c.DeletePRComment(org, repo, id)
}

func (c *ghclient) EditComment(org, repo string, ID int, comment string) error {
	return c.UpdatePRComment(org, repo, ID, comment)
}

func (c *ghclient) CreateStatus(org, repo, ref string, s github.Status) error {
	return nil
}
