package assign

import (
	"fmt"
	"strconv"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	prowConfig "k8s.io/test-infra/prow/config"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	originp "k8s.io/test-infra/prow/plugins"
	origina "k8s.io/test-infra/prow/plugins/assign"
)

type githubClient interface {
	AssignPR(owner, repo string, number int, logins []string) error
	UnassignPR(owner, repo string, number int, logins []string) error
	CreatePRComment(owner, repo string, number int, comment string) error
	AssignGiteeIssue(org, repo string, number int, login string) error
	UnassignGiteeIssue(org, repo string, number int, login string) error
	CreateGiteeIssueComment(owner, repo string, number int, comment string) error
}

type assign struct {
	getPluginConfig plugins.GetPluginConfig
	ghc             githubClient
}

type ghclient struct {
	githubClient
	isPR bool
}

func (c *ghclient) AssignIssue(owner, repo string, number int, logins []string) error {
	if c.isPR {
		return c.AssignPR(owner, repo, number, logins)
	}

	if len(logins) > 1 {
		return fmt.Errorf("can't assign more one persons to an issue at same time")
	}
	return c.AssignGiteeIssue(owner, repo, number, logins[0])
}

func (c *ghclient) UnassignIssue(owner, repo string, number int, logins []string) error {
	if c.isPR {
		return c.UnassignPR(owner, repo, number, logins)
	}

	if len(logins) > 1 {
		return fmt.Errorf("can't unassign more one persons from an issue at same time")
	}
	return c.UnassignGiteeIssue(owner, repo, number, logins[0])
}

func (c *ghclient) CreateComment(owner, repo string, number int, comment string) error {
	if c.isPR {
		return c.CreatePRComment(owner, repo, number, comment)
	}

	return c.CreateGiteeIssueComment(owner, repo, number, comment)
}

func (c *ghclient) RequestReview(org, repo string, number int, logins []string) error {
	return nil
}

func (c *ghclient) UnrequestReview(org, repo string, number int, logins []string) error {
	return nil
}

func NewAssign(f plugins.GetPluginConfig, ghc githubClient) plugins.Plugin {
	return &assign{
		getPluginConfig: f,
		ghc:             ghc,
	}
}

func (a *assign) HelpProvider(enabledRepos []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	h, ok := originp.HelpProviders()[a.PluginName()]
	if !ok {
		return nil, fmt.Errorf("can't find the assign's original helper method")
	}

	return h(nil, enabledRepos)
}

func (a *assign) PluginName() string {
	return "assign"
}

func (a *assign) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

func (a *assign) RegisterEventHandler(p plugins.Plugins) {
	name := a.PluginName()
	p.RegisterNoteEventHandler(name, a.handleNoteEvent)
}

func (a *assign) handleNoteEvent(e *sdk.NoteEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handleNoteEvent")
	}()

	if *(e.Action) != "comment" {
		log.Debug("Event is not a creation of a comment, skipping.")
		return nil
	}

	isPR := (*(e.NoteableType) == "PullRequest")
	n := e.PullRequest.Number
	if !isPR {
		v, _ := strconv.ParseInt(e.Issue.Number, 10, 32)
		n = int32(v)
	}

	ce := github.GenericCommentEvent{
		Repo: github.Repo{
			Owner: github.User{Login: e.Repository.Owner.Login},
			Name:  e.Repository.Name,
		},
		Body:    e.Comment.Body,
		User:    github.User{Login: e.Comment.User.Login},
		Number:  int(n),
		HTMLURL: e.Comment.HtmlUrl,
		IsPR:    false,
	}

	return origina.Handle(ce, &ghclient{githubClient: a.ghc, isPR: true}, log)
}
