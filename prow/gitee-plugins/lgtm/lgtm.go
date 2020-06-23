package lgtm

import (
	"fmt"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/commentpruner"
	prowConfig "k8s.io/test-infra/prow/config"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	originp "k8s.io/test-infra/prow/plugins"
	originl "k8s.io/test-infra/prow/plugins/lgtm"
	"k8s.io/test-infra/prow/repoowners"
)

type getAllConf func() *plugins.Configurations

type lgtm struct {
	getPluginConfig plugins.GetPluginConfig
	ghc             *ghclient
	getAllConf      getAllConf
	oc              repoowners.Interface
}

func NewLGTM(f plugins.GetPluginConfig, f1 getAllConf, gec giteeClient, oc repoowners.Interface) plugins.Plugin {
	return &lgtm{
		getPluginConfig: f,
		ghc:             &ghclient{giteeClient: gec},
		getAllConf:      f1,
		oc:              oc,
	}
}

func (lg *lgtm) HelpProvider(enabledRepos []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	c, err := lg.pluginConfig()
	if err != nil {
		return nil, err
	}

	c1 := originp.Configuration{Lgtm: c.Lgtm}

	return originl.HelpProvider(&c1, enabledRepos)
}

func (lg *lgtm) PluginName() string {
	return originl.PluginName
}

func (lg *lgtm) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

func (lg *lgtm) RegisterEventHandler(p plugins.Plugins) {
	name := lg.PluginName()
	p.RegisterNoteEventHandler(name, lg.handleNoteEvent)
	p.RegisterPullRequestHandler(name, lg.handlePullRequestEvent)
}

func (lg *lgtm) handleNoteEvent(e *sdk.NoteEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handleNoteEvent")
	}()

	if *(e.NoteableType) != "PullRequest" {
		log.Debug("Event is not a creation of a comment on a PR, skipping.")
		return nil
	}

	if *(e.Action) != "comment" {
		log.Debug("Event is not a creation of a comment on an open PR, skipping.")
		return nil
	}

	toAdd, toRemove := doWhat(e.Comment.Body)
	if !(toAdd || toRemove) {
		return nil
	}

	c, err := lg.buildOriginConfig()
	if err != nil {
		return err
	}

	pr := e.PullRequest
	assignees := make([]github.User, len(pr.Assignees))
	for i, item := range pr.Assignees {
		assignees[i] = github.User{Login: item.Login}
	}

	var repo github.Repo
	repo.Owner.Login = e.Repository.Namespace
	repo.Name = e.Repository.Name

	comment := e.Comment
	rc := originl.NewReviewCtx(
		comment.User.Login, pr.Head.User.Login, comment.Body,
		comment.HtmlUrl, repo, assignees, int(pr.Number))

	cp := commentpruner.NewEventClient(
		lg.ghc, log.WithField("client", "commentpruner"),
		repo.Owner.Login, repo.Name, int(pr.Number),
	)

	return originl.Handle(toAdd, c, lg.oc, rc, lg.ghc, log, cp)
}

func (lg *lgtm) handlePullRequestEvent(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handlePullRequest")
	}()

	if e.PullRequest.State != "open" {
		log.Debug("Pull request state is not open, skipping...")
		return nil
	}

	if *(e.Action) != "update" {
		log.Debug("Pull request event is not update , skipping...")
		return nil
	}

	c, err := lg.buildOriginConfig()
	if err != nil {
		return err
	}

	pr := e.PullRequest
	var pe github.PullRequestEvent
	pe.Action = plugins.ConvertPullRequestAction(e)
	pe.PullRequest.Base.Repo.Owner.Login = pr.Base.Repo.Namespace
	pe.PullRequest.Base.Repo.Name = pr.Base.Repo.Name
	pe.PullRequest.User.Login = pr.Head.User.Login
	pe.PullRequest.Number = int(pr.Number)
	pe.PullRequest.Head.SHA = pr.Head.Sha

	return originl.HandlePullRequest(log, lg.ghc, c, &pe)
}

func (lg *lgtm) pluginConfig() (*configuration, error) {
	c := lg.getPluginConfig(lg.PluginName())
	if c == nil {
		return nil, fmt.Errorf("can't find the lgtm's configuration")
	}

	c1, ok := c.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to lgtm's configuration")
	}
	return c1, nil
}

func (lg *lgtm) buildOriginConfig() (*originp.Configuration, error) {
	c, err := lg.pluginConfig()
	if err != nil {
		return nil, err
	}

	return &originp.Configuration{
		Lgtm:   c.Lgtm,
		Owners: lg.getAllConf().Owners,
	}, nil
}

func doWhat(comment string) (bool, bool) {
	// If we create an "/lgtm" comment, add lgtm if necessary.
	if originl.LGTMRe.MatchString(comment) {
		return true, false
	}

	// If we create a "/lgtm cancel" comment, remove lgtm if necessary.
	if originl.LGTMCancelRe.MatchString(comment) {
		return false, true
	}

	return false, false
}
