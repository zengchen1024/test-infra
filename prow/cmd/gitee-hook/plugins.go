package main

import (
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/gitee-plugins/approve"
	"k8s.io/test-infra/prow/gitee-plugins/assign"
)

func initPlugins(agent *plugins.ConfigAgent, pm plugins.Plugins, cs *clients) {
	gpc := func(name string) plugins.PluginConfig {
		return agent.Config().GetPluginConfig(name)
	}

	var v []plugins.Plugin
	v = append(v, approve.NewApprove(gpc, cs.giteeClient, cs.ownersClient))
	v = append(v, assign.NewAssign(gpc, cs.giteeClient))

	for _, i := range v {
		name := i.PluginName()

		i.RegisterEventHandler(pm)
		pm.RegisterHelper(name, i.HelpProvider)

		agent.RegisterPluginConfigBuilder(name, i.NewPluginConfig)
	}
}
