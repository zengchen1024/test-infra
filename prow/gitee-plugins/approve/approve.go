package approve

import (
	"fmt"

	"github.com/sirupsen/logrus"
	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	originp "k8s.io/test-infra/prow/plugins"
	origina "k8s.io/test-infra/prow/plugins/approve"
	"k8s.io/test-infra/prow/repoowners"
)

type approve struct {
	getPluginConfig plugins.GetPluginConfig
	ghc             githubClient
	oc              ownersClient
}

type githubClient interface {
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	ListIssueComments(org, repo string, number int) ([]github.IssueComment, error)
	ListReviews(org, repo string, number int) ([]github.Review, error)
	ListPullRequestComments(org, repo string, number int) ([]github.ReviewComment, error)
	DeleteComment(org, repo string, ID int) error
	CreateComment(org, repo string, number int, comment string) error
	BotName() (string, error)
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
	ListIssueEvents(org, repo string, num int) ([]github.ListedIssueEvent, error)
}

type ownersClient interface {
	LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error)
}

func NewApprove(f plugins.GetPluginConfig, ghc githubClient, oc ownersClient) plugins.Plugin {
	return &approve{
		getPluginConfig: f,
		ghc:             ghc,
		oc:              oc,
	}
}

func (a *approve) HelpProvider(enabledRepos []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	c := a.getPluginConfig(a.PluginName())
	if c == nil {
		return nil, fmt.Errorf("can't find the approve's configuration")
	}

	c1, ok := c.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to approve's configuration")
	}

	c2 := originp.Configuration{Approve: c1.Approve}

	h, ok := originp.HelpProviders()[a.PluginName()]
	if !ok {
		return nil, fmt.Errorf("can't find the approve's original helper method")
	}

	return h(&c2, enabledRepos)
}

func (a *approve) PluginName() string {
	return origina.PluginName
}

/*
func (a *approve) PluginConfigBuilder() PluginConfigBuilder {
	return func() plugins.PluginConfig {
		return &configuration{}
	}
}
*/

func (a *approve) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

func (a *approve) RegisterEventHandler(p plugins.Plugins) {
	name := a.PluginName()
	p.RegisterGenericCommentHandler(name, a.handleGenericCommentEvent)
	p.RegisterReviewEventHandler(name, a.handleReviewEvent)
	p.RegisterPullRequestHandler(name, a.handlePullRequestEvent)
}

func (a *approve) handleGenericCommentEvent(e *github.GenericCommentEvent, log *logrus.Entry) error {
	return nil
}

func (a *approve) handleReviewEvent(e *github.ReviewEvent, log *logrus.Entry) error {
	return nil
}

func (a *approve) handlePullRequestEvent(e *github.PullRequestEvent, log *logrus.Entry) error {
	return nil
}
