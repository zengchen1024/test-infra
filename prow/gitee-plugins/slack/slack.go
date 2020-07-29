package slack

import (
	"fmt"
	"regexp"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	prowConfig "k8s.io/test-infra/prow/config"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/pluginhelp"
)

type getAllConf func() *plugins.Configurations

type slack struct {
	getPluginConfig plugins.GetPluginConfig

	// robot is the account of robot on gitee
	robot string

	c client
}

func NewSlack(f plugins.GetPluginConfig, robot string) plugins.Plugin {
	return &slack{
		getPluginConfig: f,
		robot:           robot,
	}
}

func (this *slack) HelpProvider(enabledRepos []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The slack plugin will transfer events of Gitee to slack. it sends comments of Gitee to different chanels by the author of comment",
	}
	return pluginHelp, nil
}

func (this *slack) PluginName() string {
	return "slack"
}

func (this *slack) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

func (this *slack) RegisterEventHandler(p plugins.Plugins) {
	name := this.PluginName()
	p.RegisterNoteEventHandler(name, this.handleNoteEvent)
}

func (this *slack) handleNoteEvent(e *sdk.NoteEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handleNoteEvent")
	}()

	m := genCommentMessage(e)
	if m == "" {
		return nil
	}

	c, err := this.pluginConfig()
	if err != nil {
		return err
	}

	if e.Comment.User.Login == this.robot {
		return this.c.dispatch(c.Slack.RobotChannel, m)
	}
	return this.c.dispatch(c.Slack.DevChannel, m)
}

func (this *slack) pluginConfig() (*configuration, error) {
	c := this.getPluginConfig(this.PluginName())
	if c == nil {
		return nil, fmt.Errorf("can't find the slack's configuration")
	}

	c1, ok := c.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to slack's configuration")
	}
	return c1, nil
}

func genCommentMessage(e *sdk.NoteEvent) string {
	if (*e.Action) != "comment" && (*e.Action) != "edited" {
		return ""
	}

	switch *(e.NoteableType) {
	case "PullRequest":
		return fmt.Sprintf(
			"<%s|%s> left a <%s|comment> for pr:<%s|%s> of repo:<%s|%s>.\n\"\n%s\n\"\n",
			e.Comment.User.HtmlUrl,
			e.Comment.User.Name,
			e.Comment.HtmlUrl,
			e.PullRequest.HtmlUrl,
			e.PullRequest.Title,
			e.Repository.HtmlUrl,
			e.Repository.FullName,
			newComment(e.Comment.Body),
		)

	case "Issue":
		return fmt.Sprintf(
			"<%s|%s> left a <%s|comment> for issue:<%s|%s> of repo:<%s|%s>.\n\"\n%s\n\"\n",
			e.Comment.User.HtmlUrl,
			e.Comment.User.Name,
			e.Comment.HtmlUrl,
			e.Issue.HtmlUrl,
			e.Issue.Title,
			e.Repository.HtmlUrl,
			e.Repository.FullName,
			newComment(e.Comment.Body),
		)
	}

	return ""
}

func newComment(comment string) string {
	re := regexp.MustCompile("\\[([^[]*)\\]\\(([^[]*)\\)")

	return re.ReplaceAllString(comment, `<$2|$1>`)
}
