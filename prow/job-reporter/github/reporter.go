package github

import (
	"k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/crier/reporters/github"
	"k8s.io/test-infra/prow/github/report"
	jobreporter "k8s.io/test-infra/prow/job-reporter"
)

type reporter struct {
	c *github.Client
}

func NewReporter(gc report.GitHubClient, cfg config.Getter, reportAgent v1.ProwJobAgent) *reporter {
	c := github.NewReporter(gc, cfg, reportAgent)
	return &reporter{c: c}
}

func (r *reporter) GetName() string {
	return r.c.GetName()
}

func (r *reporter) ShouldReport(pj *v1.ProwJob) bool {
	if pj.Annotations != nil {
		if v, ok := pj.Annotations[jobreporter.JobPlatformAnnotation]; ok && (v != "github") {
			return false
		}
	}

	return r.c.ShouldReport(pj)
}

func (r *reporter) Report(pj *v1.ProwJob) ([]*v1.ProwJob, error) {
	return r.c.Report(pj)
}
