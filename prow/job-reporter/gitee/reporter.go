package gitee

import (
	"k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/crier/reporters/github"
	jobreporter "k8s.io/test-infra/prow/job-reporter"
)

type reporter struct {
	c *github.Client
}

func NewReporter(gc giteeClient, cfg config.Getter, reportAgent v1.ProwJobAgent) *reporter {
	c := github.NewReporter(&ghclient{giteeClient: gc}, cfg, reportAgent)
	return &reporter{c: c}
}

// GetName returns the name of the reporter
func (r *reporter) GetName() string {
	return "gitee-reporter"
}

func (r *reporter) ShouldReport(pj *v1.ProwJob) bool {
	if pj.Annotations != nil {
		if v, ok := pj.Annotations[jobreporter.JobPlatformAnnotation]; ok && (v == "gitee") {
			return r.c.ShouldReport(pj)
		}
	}

	return false
}

func (r *reporter) Report(pj *v1.ProwJob) ([]*v1.ProwJob, error) {
	return r.c.Report(pj)
}
