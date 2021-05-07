package reviewtrigger

import (
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/test-infra/prow/gitee"
)

func (rt *trigger) inferApproversReviewers(org, repo, branch string, prNumber int) (map[string]sets.String, sets.String, error) {
	ro, err := rt.oc.LoadRepoOwners(org, repo, branch)
	if err != nil {
		return nil, nil, err
	}

	filenames, err := rt.client.GetPullRequestChanges(org, repo, prNumber)
	if err != nil {
		return nil, nil, err
	}

	m := map[string]sets.String{}
	for i := range filenames {
		filename := filenames[i].Filename
		dir := filepath.Dir(filename)
		if _, ok := m[dir]; !ok {
			m[dir] = ro.Approvers(filename)
		}
	}

	return m, ro.AllReviewers(), nil
}

func (rt *trigger) newReviewState(ne gitee.PRNoteEvent) (reviewState, error) {
	org, repo := ne.GetOrgRep()

	dirApproverMap, reviewers, err := rt.inferApproversReviewers(
		org, repo, ne.PullRequest.Base.Ref, ne.GetPRNumber(),
	)
	if err != nil {
		return reviewState{}, err
	}
	approverDirMap := parseApprovers(dirApproverMap)

	cfg, err := rt.orgRepoConfig(org, repo)
	if err != nil {
		return reviewState{}, err
	}

	s := reviewState{
		org:            org,
		repo:           repo,
		headSHA:        ne.PullRequest.Head.Sha,
		botName:        rt.botName,
		prNumber:       ne.GetPRNumber(),
		c:              rt.client,
		cfg:            cfg,
		dirApproverMap: dirApproverMap,
		approverDirMap: approverDirMap,
		reviewers:      reviewers,
	}
	return s, nil

}
func (rt *trigger) handleReviewComment(ne gitee.PRNoteEvent, cmds []string) error {
	rs, err := rt.newReviewState(ne)
	if err != nil {
		return err
	}

	if !rs.reviewers.Has(ne.GetCommenter()) {
		return nil
	}

	_, isApprover := rs.approverDirMap[ne.GetCommenter()]

	for _, cmd := range cmds {
		if cmdBelongsToApprover[cmd] && !isApprover {
			// write comment
		}
	}

	return rs.handle(false)
}

func (rt *trigger) handleCIStatusComment(ne gitee.PRNoteEvent) error {
	org, repo := ne.GetOrgRep()
	cfg, err := rt.orgRepoConfig(org, repo)
	if err != nil {
		return err
	}

	status := parseCIStatus(cfg, ne.GetComment())
	if status == "" {
		return nil
	}

	if status == cfg.SuccessStatusOfJob {
		rs, err := rt.newReviewState(ne)
		if err != nil {
			return err
		}
		rs.handle(true)
	}

	if cfg.EnableLabelForCI {
		l := ""
		switch status {
		case cfg.SuccessStatusOfJob:
			l = cfg.LabelForCIPassed
		case cfg.runningStatusOfJob:
			l = cfg.LabelForCIRunning
		case cfg.FailureStatusOfJob:
			l = cfg.LabelForCIFailed
		}

		rt.client.AddPRLabel(org, repo, ne.GetPRNumber(), l)
	}

	return nil
}

func parseApprovers(dirApproverMap map[string]sets.String) map[string]sets.String {
	r := map[string]sets.String{}
	for dir, v := range dirApproverMap {
		for item := range v {
			if _, ok := r[item]; !ok {
				r[item] = sets.NewString(dir)
			} else {
				r[item].Insert(dir)
			}
		}
	}
	return r
}
