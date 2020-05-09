package trigger

import (
	"fmt"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/v2"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/job-reporter"
	"k8s.io/test-infra/prow/pluginhelp"
	originp "k8s.io/test-infra/prow/plugins"
	origint "k8s.io/test-infra/prow/plugins/trigger"
)

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
	BotName() (string, error)
	IsCollaborator(org, repo, user string) (bool, error)
	IsMember(org, user string) (bool, error)
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetRef(org, repo, ref string) (string, error)
	CreateComment(owner, repo string, number int, comment string) error
	ListIssueComments(owner, repo string, issue int) ([]github.IssueComment, error)
	CreateStatus(owner, repo, ref string, status github.Status) error
	GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error)
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
	RemoveLabel(org, repo string, number int, label string) error
	DeleteStaleComments(org, repo string, number int, comments []github.IssueComment, isStale func(github.IssueComment) bool) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
}

type prowJobClient interface {
	Create(*prowapi.ProwJob) (*prowapi.ProwJob, error)
	List(opts metav1.ListOptions) (*prowapi.ProwJobList, error)
	Update(*prowapi.ProwJob) (*prowapi.ProwJob, error)
}

type trigger struct {
	ghc             githubClient
	pjc             prowJobClient
	gitClient       git.ClientFactory
	getProwConf     prowConfig.Getter
	getPluginConfig plugins.GetPluginConfig
}

func NewTrigger(f plugins.GetPluginConfig, f1 prowConfig.Getter, gec giteeClient, pjc prowJobClient, gitc git.ClientFactory) plugins.Plugin {
	return &trigger{
		ghc:             &ghclient{giteeClient: gec},
		pjc:             pjc,
		gitClient:       gitc,
		getProwConf:     f1,
		getPluginConfig: f,
	}
}

func (t *trigger) HelpProvider(enabledRepos []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	c, err := t.buildOriginPluginConfig()
	if err != nil {
		return nil, err
	}

	return origint.HelpProvider(&c, enabledRepos)
}

func (t *trigger) PluginName() string {
	return origint.PluginName
}

func (t *trigger) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

func (t *trigger) RegisterEventHandler(p plugins.Plugins) {
	name := t.PluginName()
	p.RegisterNoteEventHandler(name, t.handleNoteEvent)
	p.RegisterPullRequestHandler(name, t.handlePullRequestEvent)
	p.RegisterPushEventHandler(name, t.handlePushEvent)
}

func (t *trigger) handleNoteEvent(e *sdk.NoteEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handleNoteEvent")
	}()

	c, err := t.triggerFor(e.Repository.Owner.Login, e.Repository.Name)
	if err != nil {
		return err
	}

	return origint.HandleGenericComment(
		t.buildOriginClient(log),
		c,
		plugins.NoteEventToCommentEvent(e))
}

func (t *trigger) handlePullRequestEvent(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handlePullRequest")
	}()

	c, err := t.triggerFor(e.Repository.Owner.Login, e.Repository.Name)
	if err != nil {
		return err
	}

	return origint.HandlePR(
		t.buildOriginClient(log),
		c,
		plugins.ConvertPullRequestEvent(e),
		func(m []prowConfig.Presubmit) {
			setPresubmit(e.Repository.Owner.Login, e.Repository.Name, m)
		},
	)
}

func (t *trigger) handlePushEvent(e *sdk.PushEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handlePushEvent")
	}()

	return origint.HandlePE(
		t.buildOriginClient(log),
		plugins.ConvertPushEvent(e),
		func(m []prowConfig.Postsubmit) {
			setPostsubmit(e.Repository.Owner.Login, e.Repository.Name, m)
		},
	)
}

func (t *trigger) buildOriginPluginConfig() (originp.Configuration, error) {
	r := originp.Configuration{}

	c := t.getPluginConfig(t.PluginName())
	if c == nil {
		return r, fmt.Errorf("can't find the trigger's configuration")
	}

	c1, ok := c.(*configuration)
	if !ok {
		return r, fmt.Errorf("can't convert to trigger's configuration")
	}

	r.Triggers = c1.Triggers
	return r, nil
}

func (t *trigger) triggerFor(org, repo string) (originp.Trigger, error) {
	c, err := t.buildOriginPluginConfig()
	if err != nil {
		return originp.Trigger{}, err
	}

	return c.TriggerFor(org, repo), nil
}

func (t *trigger) buildOriginClient(log *logrus.Entry) origint.Client {
	return origint.Client{
		GitHubClient:  t.ghc,
		Config:        t.getProwConf(),
		ProwJobClient: t.pjc,
		Logger:        log,
		GitClient:     t.gitClient,
	}
}

func setPresubmit(org, repo string, m []prowConfig.Presubmit) {
	/* can't write as this, or the JobBase can't be changed
	for _, i := range m {
		setJob(org, repo, &i.JobBase)
	}*/

	for i, _ := range m {
		setJob(org, repo, &m[i].JobBase)
	}
}

func setPostsubmit(org, repo string, m []prowConfig.Postsubmit) {
	for i, _ := range m {
		setJob(org, repo, &m[i].JobBase)
	}
}

func setJob(org, repo string, job *prowConfig.JobBase) {
	job.CloneURI = fmt.Sprintf("https://gitee.com/%s/%s", org, repo)

	if job.Annotations == nil {
		job.Annotations = make(map[string]string)
	}
	job.Annotations[jobreporter.JobPlatformAnnotation] = "gitee"
}
