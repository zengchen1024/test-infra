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
	reporter "k8s.io/test-infra/prow/job-reporter"
	"k8s.io/test-infra/prow/pluginhelp"
	originp "k8s.io/test-infra/prow/plugins"
	origint "k8s.io/test-infra/prow/plugins/trigger"
)

type prowJobClient interface {
	Create(*prowapi.ProwJob) (*prowapi.ProwJob, error)
	List(opts metav1.ListOptions) (*prowapi.ProwJobList, error)
	Update(*prowapi.ProwJob) (*prowapi.ProwJob, error)
}

type trigger struct {
	gec             giteeClient
	ghc             *ghclient
	pjc             prowJobClient
	gitClient       git.ClientFactory
	getProwConf     prowConfig.Getter
	getPluginConfig plugins.GetPluginConfig
}

func NewTrigger(f plugins.GetPluginConfig, f1 prowConfig.Getter, gec giteeClient, pjc prowJobClient, gitc git.ClientFactory) plugins.Plugin {
	return &trigger{
		gec:             gec,
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

	c, err := t.triggerFor(e.Repository.Namespace, e.Repository.Path)
	if err != nil {
		return err
	}

	ge := plugins.NoteEventToCommentEvent(e)

	cl := t.buildOriginClient(log)
	cl.GitHubClient = &ghclient{giteeClient: t.gec, prNumber: ge.Number}

	return origint.HandleGenericComment(
		cl, c, ge,
		func(m []prowConfig.Presubmit) {
			SetPresubmit(e.Repository.Namespace, e.Repository.Path, m)
		},
	)
}

func (t *trigger) handlePullRequestEvent(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handlePullRequest")
	}()

	c, err := t.triggerFor(e.Repository.Namespace, e.Repository.Path)
	if err != nil {
		return err
	}

	return origint.HandlePR(
		t.buildOriginClient(log),
		c,
		plugins.ConvertPullRequestEvent(e),
		func(m []prowConfig.Presubmit) {
			SetPresubmit(e.Repository.Namespace, e.Repository.Path, m)
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
			setPostsubmit(e.Repository.Namespace, e.Repository.Path, m)
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

func SetPresubmit(org, repo string, m []prowConfig.Presubmit) {
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
	job.Annotations[reporter.JobPlatformAnnotation] = "gitee"
}
