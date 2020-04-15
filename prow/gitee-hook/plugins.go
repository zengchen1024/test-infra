package hook

import (
	"fmt"

	pm "k8s.io/test-infra/prow/gitee-plugins"
)

type plugins struct {
	c  *pm.ConfigAgent
	ps pm.Plugins
}

func (p *plugins) GenericCommentHandlers(owner, repo string) map[string]pm.GenericCommentHandler {
	ps := p.getPlugins(owner, repo)
	hs := p.ps.GenericCommentHandlers()

	r := map[string]pm.GenericCommentHandler{}
	for _, p := range ps {
		if h, ok := hs[p]; ok {
			r[p] = h
		}
	}
	return r
}

func (p *plugins) IssueHandlers(owner, repo string) map[string]pm.IssueHandler {
	ps := p.getPlugins(owner, repo)
	hs := p.ps.IssueHandlers()

	r := map[string]pm.IssueHandler{}
	for _, p := range ps {
		if h, ok := hs[p]; ok {
			r[p] = h
		}
	}
	return r
}

func (p *plugins) IssueCommentHandlers(owner, repo string) map[string]pm.IssueCommentHandler {
	ps := p.getPlugins(owner, repo)
	hs := p.ps.IssueCommentHandlers()

	r := map[string]pm.IssueCommentHandler{}
	for _, p := range ps {
		if h, ok := hs[p]; ok {
			r[p] = h
		}
	}
	return r
}

func (p *plugins) PullRequestHandlers(owner, repo string) map[string]pm.PullRequestHandler {
	ps := p.getPlugins(owner, repo)
	hs := p.ps.PullRequestHandlers()

	r := map[string]pm.PullRequestHandler{}
	for _, p := range ps {
		if h, ok := hs[p]; ok {
			r[p] = h
		}
	}
	return r
}

func (p *plugins) PushEventHandlers(owner, repo string) map[string]pm.PushEventHandler {
	ps := p.getPlugins(owner, repo)
	hs := p.ps.PushEventHandlers()

	r := map[string]pm.PushEventHandler{}
	for _, p := range ps {
		if h, ok := hs[p]; ok {
			r[p] = h
		}
	}
	return r
}

func (p *plugins) ReviewEventHandlers(owner, repo string) map[string]pm.ReviewEventHandler {
	ps := p.getPlugins(owner, repo)
	hs := p.ps.ReviewEventHandlers()

	r := map[string]pm.ReviewEventHandler{}
	for _, p := range ps {
		if h, ok := hs[p]; ok {
			r[p] = h
		}
	}
	return r
}

func (p *plugins) ReviewCommentEventHandlers(owner, repo string) map[string]pm.ReviewCommentEventHandler {
	ps := p.getPlugins(owner, repo)
	hs := p.ps.ReviewCommentEventHandlers()

	r := map[string]pm.ReviewCommentEventHandler{}
	for _, p := range ps {
		if h, ok := hs[p]; ok {
			r[p] = h
		}
	}
	return r
}

func (p *plugins) StatusEventHandlers(owner, repo string) map[string]pm.StatusEventHandler {
	ps := p.getPlugins(owner, repo)
	hs := p.ps.StatusEventHandlers()

	r := map[string]pm.StatusEventHandler{}
	for _, p := range ps {
		if h, ok := hs[p]; ok {
			r[p] = h
		}
	}
	return r
}

func (p *plugins) getPlugins(owner, repo string) []string {
	var plugins []string

	c := p.c.Config()
	fullName := fmt.Sprintf("%s/%s", owner, repo)
	plugins = append(plugins, c.Plugins[owner]...)
	plugins = append(plugins, c.Plugins[fullName]...)

	return plugins
}

func (p *plugins) PluginConfig(name string) pm.PluginConfig {
	return p.c.Config().GetPluginConfig(name)
}
