package main

import (
	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/gitee"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/gitee-plugins/approve"
	"k8s.io/test-infra/prow/gitee-plugins/assign"
	"k8s.io/test-infra/prow/gitee-plugins/lgtm"
	"k8s.io/test-infra/prow/gitee-plugins/trigger"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	originp "k8s.io/test-infra/prow/plugins"
)

func initPlugins(cfg prowConfig.Getter, agent *plugins.ConfigAgent, pm plugins.Plugins, cs *clients) {
	gpc := func(name string) plugins.PluginConfig {
		return agent.Config().GetPluginConfig(name)
	}

	var v []plugins.Plugin
	v = append(v, approve.NewApprove(gpc, cs.giteeClient, cs.ownersClient))
	v = append(v, assign.NewAssign(gpc, cs.giteeClient))
	v = append(v, lgtm.NewLGTM(gpc, agent.Config, cs.giteeClient, cs.ownersClient))
	v = append(v, trigger.NewTrigger(gpc, cfg, cs.giteeClient, cs.prowJobClient, cs.giteeGitClient))

	for _, i := range v {
		name := i.PluginName()

		i.RegisterEventHandler(pm)
		pm.RegisterHelper(name, i.HelpProvider)

		agent.RegisterPluginConfigBuilder(name, i.NewPluginConfig)
	}
}

func genHelpProvider(h plugins.HelpProvider) originp.HelpProvider {
	return func(_ *originp.Configuration, enabledRepos []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
		return h(enabledRepos)
	}
}

func resetPluginHelper(pm plugins.Plugins) {
	hs := pm.HelpProviders()
	ph := map[string]originp.HelpProvider{}
	for k, h := range hs {
		if h == nil {
			continue
		}
		ph[k] = genHelpProvider(h)
	}
	originp.ResetPluginHelp(ph)
}

type pluginHelperAgent struct {
	agent *plugins.ConfigAgent
}

func (p pluginHelperAgent) Config() *originp.Configuration {
	c := p.agent.Config()
	return &originp.Configuration{
		Plugins: c.Plugins,
	}
}

type pluginHelperClient struct {
	c gitee.Client
}

func (p pluginHelperClient) GetRepos(org string, _ bool) ([]github.Repo, error) {
	repos, err := p.c.GetRepos(org)
	if err != nil {
		return nil, err
	}

	r := make([]github.Repo, 0, len(repos))
	for _, item := range repos {
		r = append(r, github.Repo{FullName: item.FullName})
	}
	return r, nil
}
