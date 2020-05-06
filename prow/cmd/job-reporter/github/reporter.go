package github

import (
	"k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/crier/reporters/github"
	"k8s.io/test-infra/prow/github/report"
)

type reporter struct {
	c                      *github.Client
	platformAnnotationName string
}

func NewReporter(gc report.GitHubClient, cfg config.Getter, reportAgent v1.ProwJobAgent, platformAnnotationName string) *reporter {
	c := github.NewReporter(gc, cfg, reportAgent)
	return &reporter{c: c, platformAnnotationName: platformAnnotationName}
}

func (r *reporter) GetName() string {
	return r.c.GetName()
}

func (r *reporter) ShouldReport(pj *v1.ProwJob) bool {
	if v, ok := pj.Annotations[r.platformAnnotationName]; !(ok && v == "github") {
		return false
	}

	return r.c.ShouldReport(pj)
}

func (r *reporter) Report(pj *v1.ProwJob) ([]*v1.ProwJob, error) {
	return r.c.Report(pj)
}
