package reviewtrigger

import (
	"errors"
	"fmt"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/gitee"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

const (
	labelCanReview = "can-review"
	labelLGTM      = "lgtm"
	labelLBTM      = "lbtm"
	labelApproved  = "approved"
)

type cloneRepo struct {
	gitClient git.ClientFactory
	client    giteeClient
}

func NewPlugin(c git.ClientFactory, gc giteeClient) plugins.Plugin {
	return &cloneRepo{gitClient: c, client: gc}
}

func (cr *cloneRepo) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The review_trigger plugin will trigger the whole review process. first, it will clone repo in advance and
		save it at local disk when the pr is created in order to same the time for other operation, such as lgtm, approve. second
		it will handle comment of reviewer and approver, such as /lgtm, /lbtm, /approve and /reject.
		`,
	}
	return pluginHelp, nil
}

func (cr *cloneRepo) PluginName() string {
	return "review_trigger"
}

func (cr *cloneRepo) NewPluginConfig() plugins.PluginConfig {
	return nil
}

func (cr *cloneRepo) RegisterEventHandler(p plugins.Plugins) {
	p.RegisterPullRequestHandler(cr.PluginName(), cr.clone)
}

func (cr *cloneRepo) clone(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	action := plugins.ConvertPullRequestAction(e)
	switch action {
	case github.PullRequestActionOpened:
		org, repo := gitee.GetOwnerAndRepoByPREvent(e)

		err := cr.client.AddPRLabel(org, repo, int(e.PullRequest.Number), labelCanReview)
		// suggest reviewer

		if c, err := cr.gitClient.ClientFor(org, repo); err == nil {
			c.Clean()
		}

		return err

	case github.PullRequestActionSynchronize:
		err := removeInvalidLabels(e, cr.client, true)
		// suggest reviewer
		return err

	}
	return nil
}

func removeInvalidLabels(e *sdk.PullRequestEvent, c giteeClient, canReview bool) error {
	labels := e.PullRequest.Labels
	m := map[string]bool{}
	for i := range labels {
		m[labels[i].Name] = true
	}

	rml := []string{labelApproved, labelLBTM, labelLGTM}
	if !canReview {
		rml = append(rml, labelCanReview)
	}

	org, repo := gitee.GetOwnerAndRepoByPREvent(e)
	number := int(e.PullRequest.Number)

	errs := []string{}
	for _, l := range rml {
		if m[l] {
			if err := c.RemovePRLabel(org, repo, number, l); err != nil {
				errs = append(errs, fmt.Sprintf("remove label:%s, err:%v", l, err))
			}
		}
	}

	l := labelCanReview
	if canReview && !m[l] {
		if err := c.AddPRLabel(org, repo, number, l); err != nil {
			errs = append(errs, fmt.Sprintf("add label:%s, err:%v", l, err))
		}
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, ". "))
	}
	return nil
}
