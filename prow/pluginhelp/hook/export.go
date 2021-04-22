package hook

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
)

func (ha *HelpAgent) GeneratePluginHelpRestrictedToConfig() *pluginhelp.Help {
	config := ha.pa.Config()
	orgToRepos := getOrgs(config)

	normalRevMap, externalRevMap := reversePluginMaps(config, orgToRepos)

	allPlugins, pluginHelp := ha.generateNormalPluginHelp(config, normalRevMap)

	allExternalPlugins, externalPluginHelp := ha.generateExternalPluginHelp(config, externalRevMap)

	// Load repo->plugins maps from config
	repoPlugins := map[string][]string{
		"": allPlugins,
	}
	for repo, plugins := range config.Plugins {
		repoPlugins[repo] = plugins
	}
	repoExternalPlugins := map[string][]string{
		"": allExternalPlugins,
	}
	for repo, exts := range config.ExternalPlugins {
		for _, ext := range exts {
			repoExternalPlugins[repo] = append(repoExternalPlugins[repo], ext.Name)
		}
	}

	return &pluginhelp.Help{
		AllRepos:            allRepos(config, orgToRepos),
		RepoPlugins:         repoPlugins,
		RepoExternalPlugins: repoExternalPlugins,
		PluginHelp:          pluginHelp,
		ExternalPluginHelp:  externalPluginHelp,
	}
}

func getOrgs(config *plugins.Configuration) map[string]sets.String {
	r := map[string]sets.String{}

	f := func(repo string) {
		if !strings.Contains(repo, "/") {
			if _, ok := r[repo]; !ok {
				r[repo] = sets.NewString(repo)
			}
		}
	}

	for repo := range config.Plugins {
		f(repo)
	}

	for repo := range config.ExternalPlugins {
		f(repo)
	}

	return r
}
