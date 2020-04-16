package plugins

import (
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

// HelpProvider defines the function type that construct a pluginhelp.PluginHelp for enabled
// plugins. It takes into account the plugins configuration and enabled repositories.
type HelpProvider func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error)

// IssueHandler defines the function contract for a github.IssueEvent handler.
type IssueHandler func(e *github.IssueEvent, log *logrus.Entry) error

// IssueCommentHandler defines the function contract for a github.IssueCommentEvent handler.
type IssueCommentHandler func(e *github.IssueCommentEvent, log *logrus.Entry) error

// PullRequestHandler defines the function contract for a github.PullRequestEvent handler.
type PullRequestHandler func(e *github.PullRequestEvent, log *logrus.Entry) error

// StatusEventHandler defines the function contract for a github.StatusEvent handler.
type StatusEventHandler func(e *github.StatusEvent, log *logrus.Entry) error

// PushEventHandler defines the function contract for a github.PushEvent handler.
type PushEventHandler func(e *github.PushEvent, log *logrus.Entry) error

// ReviewEventHandler defines the function contract for a github.ReviewEvent handler.
type ReviewEventHandler func(e *github.ReviewEvent, log *logrus.Entry) error

// ReviewCommentEventHandler defines the function contract for a github.ReviewCommentEvent handler.
type ReviewCommentEventHandler func(e *github.ReviewCommentEvent, log *logrus.Entry) error

// GenericCommentHandler defines the function contract for a github.GenericCommentEvent handler.
type GenericCommentHandler func(e *github.GenericCommentEvent, log *logrus.Entry) error

type Plugins interface {
	RegisterHelper(name string, fn HelpProvider)
	RegisterIssueHandler(name string, fn IssueHandler)
	RegisterIssueCommentHandler(name string, fn IssueCommentHandler)
	RegisterPullRequestHandler(name string, fn PullRequestHandler)
	RegisterStatusEventHandler(name string, fn StatusEventHandler)
	RegisterPushEventHandler(name string, fn PushEventHandler)
	RegisterReviewEventHandler(name string, fn ReviewEventHandler)
	RegisterReviewCommentEventHandler(name string, fn ReviewCommentEventHandler)
	RegisterGenericCommentHandler(name string, fn GenericCommentHandler)

	HelpProviders() map[string]HelpProvider
	GenericCommentHandlers() map[string]GenericCommentHandler
	IssueHandlers() map[string]IssueHandler
	IssueCommentHandlers() map[string]IssueCommentHandler
	PullRequestHandlers() map[string]PullRequestHandler
	PushEventHandlers() map[string]PushEventHandler
	ReviewEventHandlers() map[string]ReviewEventHandler
	ReviewCommentEventHandlers() map[string]ReviewCommentEventHandler
	StatusEventHandlers() map[string]StatusEventHandler
}

func NewPluginManager() Plugins {
	return &plugins{
		pluginHelp:                 map[string]HelpProvider{},
		genericCommentHandlers:     map[string]GenericCommentHandler{},
		issueHandlers:              map[string]IssueHandler{},
		issueCommentHandlers:       map[string]IssueCommentHandler{},
		pullRequestHandlers:        map[string]PullRequestHandler{},
		pushEventHandlers:          map[string]PushEventHandler{},
		reviewEventHandlers:        map[string]ReviewEventHandler{},
		reviewCommentEventHandlers: map[string]ReviewCommentEventHandler{},
		statusEventHandlers:        map[string]StatusEventHandler{},
	}
}

type plugins struct {
	pluginHelp                 map[string]HelpProvider
	genericCommentHandlers     map[string]GenericCommentHandler
	issueHandlers              map[string]IssueHandler
	issueCommentHandlers       map[string]IssueCommentHandler
	pullRequestHandlers        map[string]PullRequestHandler
	pushEventHandlers          map[string]PushEventHandler
	reviewEventHandlers        map[string]ReviewEventHandler
	reviewCommentEventHandlers map[string]ReviewCommentEventHandler
	statusEventHandlers        map[string]StatusEventHandler
}

// RegisterHelper registers a plugin's helper method.
func (p *plugins) RegisterHelper(name string, fn HelpProvider) {
	p.pluginHelp[name] = fn
}

// RegisterIssueHandler registers a plugin's github.IssueEvent handler.
func (p *plugins) RegisterIssueHandler(name string, fn IssueHandler) {
	p.issueHandlers[name] = fn
}

// RegisterIssueCommentHandler registers a plugin's github.IssueCommentEvent handler.
func (p *plugins) RegisterIssueCommentHandler(name string, fn IssueCommentHandler) {
	p.issueCommentHandlers[name] = fn
}

// RegisterPullRequestHandler registers a plugin's github.PullRequestEvent handler.
func (p *plugins) RegisterPullRequestHandler(name string, fn PullRequestHandler) {
	p.pullRequestHandlers[name] = fn
}

// RegisterStatusEventHandler registers a plugin's github.StatusEvent handler.
func (p *plugins) RegisterStatusEventHandler(name string, fn StatusEventHandler) {
	p.statusEventHandlers[name] = fn
}

// RegisterPushEventHandler registers a plugin's github.PushEvent handler.
func (p *plugins) RegisterPushEventHandler(name string, fn PushEventHandler) {
	p.pushEventHandlers[name] = fn
}

// RegisterReviewEventHandler registers a plugin's github.ReviewEvent handler.
func (p *plugins) RegisterReviewEventHandler(name string, fn ReviewEventHandler) {
	p.reviewEventHandlers[name] = fn
}

// RegisterReviewCommentEventHandler registers a plugin's github.ReviewCommentEvent handler.
func (p *plugins) RegisterReviewCommentEventHandler(name string, fn ReviewCommentEventHandler) {
	p.reviewCommentEventHandlers[name] = fn
}

// RegisterGenericCommentHandler registers a plugin's github.GenericCommentEvent handler.
func (p *plugins) RegisterGenericCommentHandler(name string, fn GenericCommentHandler) {
	p.genericCommentHandlers[name] = fn
}

func (p *plugins) HelpProviders() map[string]HelpProvider {
	return p.pluginHelp
}

func (p *plugins) GenericCommentHandlers() map[string]GenericCommentHandler {
	return p.genericCommentHandlers
}

func (p *plugins) IssueHandlers() map[string]IssueHandler {
	return p.issueHandlers
}

func (p *plugins) IssueCommentHandlers() map[string]IssueCommentHandler {
	return p.issueCommentHandlers
}

func (p *plugins) PullRequestHandlers() map[string]PullRequestHandler {
	return p.pullRequestHandlers
}

func (p *plugins) PushEventHandlers() map[string]PushEventHandler {
	return p.pushEventHandlers
}

func (p *plugins) ReviewEventHandlers() map[string]ReviewEventHandler {
	return p.reviewEventHandlers
}

func (p *plugins) ReviewCommentEventHandlers() map[string]ReviewCommentEventHandler {
	return p.reviewCommentEventHandlers
}

func (p *plugins) StatusEventHandlers() map[string]StatusEventHandler {
	return p.statusEventHandlers
}
