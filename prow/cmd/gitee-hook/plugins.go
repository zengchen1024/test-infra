package main

import (
	prowConfig "k8s.io/test-infra/prow/config"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/gitee-plugins/approve"
	"k8s.io/test-infra/prow/gitee-plugins/assign"
	"k8s.io/test-infra/prow/gitee-plugins/lgtm"
	"k8s.io/test-infra/prow/gitee-plugins/trigger"
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
