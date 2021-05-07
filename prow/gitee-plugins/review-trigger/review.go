package reviewtrigger

import (
	"errors"
	"fmt"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/gitee"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/repoowners"
)

const (
	labelCanReview     = "can-review"
	labelLGTM          = "lgtm"
	labelApproved      = "approved"
	labelRequestChange = "request-change"
)

type trigger struct {
	client          ghclient
	botName         string
	oc              repoowners.Interface
	getPluginConfig plugins.GetPluginConfig
}

func NewPlugin(f plugins.GetPluginConfig, gc giteeClient, botName string, oc repoowners.Interface) plugins.Plugin {
	return &trigger{
		getPluginConfig: f,
		oc:              oc,
		botName:         botName,
		client:          ghclient{giteeClient: gc},
	}
}

func (cr *trigger) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The review_trigger plugin will trigger the whole review process. first, it will clone repo in advance and
		save it at local disk when the pr is created in order to same the time for other operation, such as lgtm, approve. second
		it will handle comment of reviewer and approver, such as /lgtm, /lbtm, /approve and /reject.
		`,
	}
	return pluginHelp, nil
}

func (cr *trigger) PluginName() string {
	return "review_trigger"
}

func (cr *trigger) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

func (cr *trigger) orgRepoConfig(org, repo string) (*pluginConfig, error) {
	cfg, err := cr.pluginConfig()
	if err != nil {
		return nil, err
	}

	pc := cfg.TriggerFor(org, repo)
	if pc == nil {
		return nil, fmt.Errorf("no cla plugin config for this repo:%s/%s", org, repo)
	}

	return pc, nil
}

func (cr *trigger) pluginConfig() (*configuration, error) {
	c := cr.getPluginConfig(cr.PluginName())
	if c == nil {
		return nil, fmt.Errorf("can't find the configuration")
	}

	c1, ok := c.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}

	return c1, nil
}
func (cr *trigger) RegisterEventHandler(p plugins.Plugins) {
	name := cr.PluginName()
	p.RegisterNoteEventHandler(name, cr.handleNoteEvent)
	p.RegisterPullRequestHandler(name, cr.clone)
}

func (cr *trigger) clone(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	action := plugins.ConvertPullRequestAction(e)
	switch action {
	case github.PullRequestActionOpened:
		org, repo := gitee.GetOwnerAndRepoByPREvent(e)

		err := cr.client.AddPRLabel(org, repo, int(e.PullRequest.Number), labelCanReview)
		// suggest reviewer

		// no need to update local repo everytime when a pr is open.
		// repoowner will update it necessarily when suggesting reviewers.
		return err

	case github.PullRequestActionSynchronize:
		err := removeInvalidLabels(e, cr.client, true)
		// suggest reviewer
		// delete the approver tips comment
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

	rml := []string{labelApproved, labelRequestChange, labelLGTM}
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

func (rt *trigger) handleNoteEvent(e *sdk.NoteEvent, log *logrus.Entry) error {
	ne := gitee.NewPRNoteEvent(e)
	if !ne.IsPullRequest() || !ne.IsPROpen() {
		return nil
	}

	if ne.IsCreatingCommentEvent() && ne.GetCommenter() != rt.botName {
		cmds := parseCommandFromComment(ne.GetComment())
		if len(cmds) > 0 {
			return rt.handleReviewComment(ne, cmds)
		}
	}

	return rt.handleCIStatusComment(ne)
}
