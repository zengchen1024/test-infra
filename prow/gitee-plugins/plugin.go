package plugins

import (
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/pluginhelp"
)

type Plugin interface {
	//PluginConfigBuilder() PluginConfigBuilder
	PluginName() string
	NewPluginConfig() PluginConfig
	RegisterEventHandler(p Plugins)
	HelpProvider(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error)
}

type GetPluginConfig func(string) PluginConfig
