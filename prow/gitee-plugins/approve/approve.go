package approve

import (
	"fmt"
	"net/url"

	"gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	prowConfig "k8s.io/test-infra/prow/config"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
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
	c, err := a.pluginConfig()
	if err != nil {
		return nil, err
	}

	c1 := originp.Configuration{Approve: c.Approve}

	h, ok := originp.HelpProviders()[a.PluginName()]
	if !ok {
		return nil, fmt.Errorf("can't find the approve's original helper method")
	}

	return h(&c1, enabledRepos)
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
	p.RegisterNoteEventHandler(name, a.handleNoteEvent)
	//p.RegisterPullRequestHandler(name, a.handlePullRequestEvent)
}

func (a *approve) handleNoteEvent(e *gitee.NoteEvent, log *logrus.Entry) error {
	pr := e.PullRequest
	org := e.Repository.Owner.Login
	repo := e.Repository.Name

	repoc, err := a.oc.LoadRepoOwners(org, repo, pr.Base.Ref)
	if err != nil {
		return err
	}

	c, err := a.approveFor(org, repo)
	if err != nil {
		return err
	}

	assignees := make([]github.User, 0, len(pr.Assignees))
	for i, item := range pr.Assignees {
		assignees[i] = github.User{Login: item.Login}
	}

	return origina.Handle(
		log,
		a.ghc,
		repoc,
		getGiteeOption(),
		c,
		origina.NewState(org, repo, pr.Base.Ref, pr.Body, pr.User.Login, pr.HtmlUrl, int(pr.Number), assignees),
	)
}

func (a *approve) handlePullRequestEvent(e *gitee.PullRequestEvent, log *logrus.Entry) error {
	return nil
}

func getGiteeOption() prowConfig.GitHubOptions {
	s := "https://gitee.com"
	linkURL, _ := url.Parse(s)
	return prowConfig.GitHubOptions{LinkURLFromConfig: s, LinkURL: linkURL}
}

func (a *approve) pluginConfig() (*configuration, error) {
	c := a.getPluginConfig(a.PluginName())
	if c == nil {
		return nil, fmt.Errorf("can't find the approve's configuration")
	}

	c1, ok := c.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to approve's configuration")
	}
	return c1, nil
}

func (a *approve) approveFor(org, repo string) (*originp.Approve, error) {
	c, err := a.pluginConfig()
	if err != nil {
		return nil, err
	}

	c1 := originp.Configuration{Approve: c.Approve}
	return c1.ApproveFor(org, repo), nil
}
