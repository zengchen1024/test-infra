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
}

func NewReporter(gc giteeClient, cfg config.Getter, reportAgent v1.ProwJobAgent) *reporter {
	ghc := &ghclient{giteeClient: gc}
	c := github.NewReporter(ghc, cfg, reportAgent)
	return &reporter{c: c, ghc: ghc}
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

			deleteOldJobsResultComment(r.ghc, pj)
		}
	}

	return false
}

func (r *reporter) Report(pj *v1.ProwJob) ([]*v1.ProwJob, error) {
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

func deleteOldJobsResultComment(ghc *ghclient, pj *v1.ProwJob) error {
	refs := pj.Spec.Refs
	if len(refs.Pulls) == 1 {
		org := refs.Org
		repo := refs.Repo
		sha := refs.Pulls[0].SHA
		prNumber := refs.Pulls[0].Number

		comments, err := ghc.ListIssueComments(org, repo, prNumber)
		if err != nil {
			return err
		}

		botname, err := ghc.BotName()
		if err != nil {
			return err
		}

		v, commentId := findCheckResultComment(botname, sha, comments)
		if v != "" {
			return ghc.DeletePRComment(org, repo, commentId)
		}
	}
	return nil
}
