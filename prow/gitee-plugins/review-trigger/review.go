package reviewtrigger

import (
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

func (rt *trigger) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The review_trigger plugin will trigger the whole review process to merge pull-request.
		It will handle comment of reviewer and approver, such as /lgtm, /lbtm, /approve and /reject.
		Also, it can add label of CI test cases.
		`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/lgtm",
		Description: "The code looks good. It will add 'lgtm' label if reviewer comment /lgtm",
		Featured:    true,
		WhoCanUse:   "Anyone",
		Examples:    []string{"/lgtm"},
	})
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/lbtm",
		Description: "The code looks bad. It will add 'request-change' label if reviewer comment /lbtm",
		Featured:    true,
		WhoCanUse:   "Anyone",
		Examples:    []string{"/lbtm"},
	})
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/approve",
		Description: "The code is ready to be merged. It may add 'approved' label if approver comment /approve",
		Featured:    true,
		WhoCanUse:   "approver",
		Examples:    []string{"/approve"},
	})
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/reject",
		Description: "The code can't be merged. It will add 'request-change' label if approver comment /reject",
		Featured:    true,
		WhoCanUse:   "approver",
		Examples:    []string{"/reject"},
	})

	return pluginHelp, nil
}

func (rt *trigger) PluginName() string {
	return "review_trigger"
}

func (rt *trigger) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

func (rt *trigger) orgRepoConfig(org, repo string) (*pluginConfig, error) {
	cfg, err := rt.pluginConfig()
	if err != nil {
		return nil, err
	}

	pc := cfg.TriggerFor(org, repo)
	if pc == nil {
		return nil, fmt.Errorf("no cla plugin config for this repo:%s/%s", org, repo)
	}

	return pc, nil
}

func (rt *trigger) pluginConfig() (*configuration, error) {
	c := rt.getPluginConfig(rt.PluginName())
	if c == nil {
		return nil, fmt.Errorf("can't find the configuration")
	}

	c1, ok := c.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}

	return c1, nil
}
func (rt *trigger) RegisterEventHandler(p plugins.Plugins) {
	name := rt.PluginName()
	p.RegisterNoteEventHandler(name, rt.handleNoteEvent)
	p.RegisterPullRequestHandler(name, rt.handlePREvent)
}

func (rt *trigger) handlePREvent(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	action := plugins.ConvertPullRequestAction(e)
	org, repo := gitee.GetOwnerAndRepoByPREvent(e)
	prNumber := int(e.PullRequest.Number)
	switch action {
	case github.PullRequestActionOpened:
		err := rt.client.AddPRLabel(org, repo, prNumber, labelCanReview)
		// suggest reviewer

		// no need to update local repo everytime when a pr is open.
		// repoowner will update it necessarily when suggesting reviewers.
		return err

	case github.PullRequestActionSynchronize:
		errs := newErrors()
		if err := rt.removeInvalidLabels(e, true); err != nil {
			errs.add(fmt.Sprintf("remove label when source code changed, err:%s", err.Error()))
		}
		// suggest reviewer

		if err := rt.deleteTips(org, repo, prNumber); err != nil {
			errs.add(fmt.Sprintf("delete tips, err:%s", err.Error()))
		}
		return errs.err()

	}
	return nil
}

func (rt *trigger) removeInvalidLabels(e *sdk.PullRequestEvent, canReview bool) error {
	org, repo := gitee.GetOwnerAndRepoByPREvent(e)
	cfg, err := rt.orgRepoConfig(org, repo)
	if err != nil {
		return err
	}

	rml := []string{labelApproved, labelRequestChange, labelLGTM}
	if cfg.EnableLabelForCI {
		rml = append(rml, cfg.labelsForCI()...)
	}
	if !canReview {
		rml = append(rml, labelCanReview)
	}

	number := int(e.PullRequest.Number)
	m := gitee.GetLabelFromEvent(e.PullRequest.Labels)

	rmls := make([]string, 0, len(rml))
	errs := newErrors()
	for _, l := range rml {
		if m[l] {
			if err := rt.client.RemovePRLabel(org, repo, number, l); err != nil {
				errs.add(fmt.Sprintf("remove label:%s, err:%v", l, err))
			}
			rmls = append(rmls, l)
		}
	}
	if len(rmls) > 0 {
		rt.client.CreatePRComment(
			org, repo, number, fmt.Sprintf(
				"New changes are detected. Remove the following labels: %s.",
				strings.Join(rmls, ", "),
			),
		)
	}

	l := labelCanReview
	if canReview && !m[l] {
		if err := rt.client.AddPRLabel(org, repo, number, l); err != nil {
			errs.add(fmt.Sprintf("add label:%s, err:%v", l, err))
		}
	}

	return errs.err()
}

func (rt *trigger) deleteTips(org, repo string, prNumber int) error {
	comments, err := rt.client.ListPRComments(org, repo, prNumber)
	if err != nil {
		return err
	}

	tips := findApproveTips(comments, rt.botName)
	if tips != nil {
		return rt.client.DeletePRComment(org, repo, int(tips.Id))
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
