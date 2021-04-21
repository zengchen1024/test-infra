package clonerepo

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/gitee"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

type cloneRepo struct {
	gitClient git.ClientFactory
}

func NewPlugin(c git.ClientFactory) plugins.Plugin {
	return &cloneRepo{gitClient: c}
}

func (cr *cloneRepo) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The clone_repo plugin will clone repo and save it at local disk when the pr is created
		in order to same the time for other plugin, such as lgtm, approve.`,
	}
	return pluginHelp, nil
}

func (cr *cloneRepo) PluginName() string {
	return "clone_repo"
}

func (cr *cloneRepo) NewPluginConfig() plugins.PluginConfig {
	return nil
}

func (cr *cloneRepo) RegisterEventHandler(p plugins.Plugins) {
	p.RegisterPullRequestHandler(cr.PluginName(), cr.clone)
}

func (cr *cloneRepo) clone(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	action := plugins.ConvertPullRequestAction(e)
	if action != github.PullRequestActionOpened {
		return nil
	}

	org, repo := gitee.GetOwnerAndRepoByPREvent(e)
	c, err := cr.gitClient.ClientFor(org, repo)
	c.Clean()
	return err
}
