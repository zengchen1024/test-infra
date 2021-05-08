package reviewtrigger

import (
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"k8s.io/test-infra/prow/github"
)

type giteeClient interface {
	AddPRLabel(owner, repo string, number int, label string) error
	AddMultiPRLabel(org, repo string, number int, label []string) error
	RemovePRLabel(owner, repo string, number int, label string) error
	GetPRCommit(org, repo, SHA string) (sdk.RepoCommit, error)
	ListPRComments(org, repo string, number int) ([]sdk.PullRequestComments, error)
	GetPRLabels(org, repo string, number int) ([]sdk.Label, error)
	CreatePRComment(owner, repo string, number int, comment string) error
	DeletePRComment(org, repo string, ID int) error
	UpdatePRComment(org, repo string, commentID int, comment string) error
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
}

type ghclient struct {
	giteeClient
}

func (c ghclient) getPRCodeUpdateTime(org, repo, headSHA string) (time.Time, error) {
	v, err := c.GetPRCommit(org, repo, headSHA)
	if err != nil {
		return time.Time{}, err
	}

	return v.Commit.Committer.Date, nil
}

func (c ghclient) getPRCurrentLabels(org, repo string, number int) (map[string]bool, error) {
	labels, err := c.GetPRLabels(org, repo, number)
	if err != nil {
		return nil, err
	}

	m := map[string]bool{}
	for i := range labels {
		m[labels[i].Name] = true
	}
	return m, nil
}
