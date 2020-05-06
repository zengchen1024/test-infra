package gitee

import (
	"k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/crier/reporters/github"
)

type reporter struct {
	c                      *github.Client
	platformAnnotationName string
}

func NewReporter(gc giteeClient, cfg config.Getter, reportAgent v1.ProwJobAgent, platformAnnotationName string) *reporter {
	c := github.NewReporter(&ghclient{giteeClient: gc}, cfg, reportAgent)
	return &reporter{c: c, platformAnnotationName: platformAnnotationName}
}

// GetName returns the name of the reporter
func (r *reporter) GetName() string {
	return "gitee-reporter"
}

func (r *reporter) ShouldReport(pj *v1.ProwJob) bool {
	if v, ok := pj.Annotations[r.platformAnnotationName]; !(ok && v == "gitee") {
		return false
	}

	return r.c.ShouldReport(pj)
}

func (r *reporter) Report(pj *v1.ProwJob) ([]*v1.ProwJob, error) {
	return r.c.Report(pj)
}
