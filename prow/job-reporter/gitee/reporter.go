package gitee

import (
	"k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/crier/reporters/github"
	jobreporter "k8s.io/test-infra/prow/job-reporter"
)

type reporter struct {
	c   *github.Client
	ghc *ghclient
	gec giteeClient
}

func NewReporter(gec giteeClient, cfg config.Getter, reportAgent v1.ProwJobAgent) *reporter {
	ghc := &ghclient{giteeClient: gec}
	c := github.NewReporter(ghc, cfg, reportAgent)
	return &reporter{c: c, ghc: ghc, gec: gec}
}

// GetName returns the name of the reporter
func (r *reporter) GetName() string {
	return "gitee-reporter"
}

func (r *reporter) ShouldReport(pj *v1.ProwJob) bool {
	if pj.Annotations != nil {
		if v, ok := pj.Annotations[jobreporter.JobPlatformAnnotation]; ok && (v == "gitee") {
			// only report status for the newest commit
			if isForTheNewestCommit(r.ghc, pj) {
				return r.c.ShouldReport(pj)
			}
		}
	}

	return false
}

func (r *reporter) Report(pj *v1.ProwJob) ([]*v1.ProwJob, error) {
	if pj.Spec.Refs.Pulls != nil && len(pj.Spec.Refs.Pulls) == 1 {
		return r.c.Report1(pj, &ghclient{giteeClient: r.gec, prNumber: pj.Spec.Refs.Pulls[0].Number})
	}

	return r.c.Report(pj)
}

func isForTheNewestCommit(ghc *ghclient, pj *v1.ProwJob) bool {
	refs := pj.Spec.Refs
	if len(refs.Pulls) == 1 {
		sha := refs.Pulls[0].SHA
		prNumber := refs.Pulls[0].Number

		pr, err := ghc.GetGiteePullRequest(refs.Org, refs.Repo, prNumber)
		if err == nil {
			return sha == pr.Head.Sha
		}
	}
	return true
}
