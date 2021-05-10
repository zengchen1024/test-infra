package reviewtrigger

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/test-infra/prow/gitee"
	op "k8s.io/test-infra/prow/plugins"
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
		currentLabels:  gitee.GetLabelFromEvent(ne.PullRequest.Labels),
		c:              rt.client,
		cfg:            cfg,
		dirApproverMap: dirApproverMap,
		approverDirMap: parseApprovers(dirApproverMap),
		reviewers:      reviewers,
	}
	return s, nil

}
func (rt *trigger) handleReviewComment(ne gitee.PRNoteEvent, cmds []string) error {
	rs, err := rt.newReviewState(ne)
	if err != nil {
		return err
	}

	commenter := ne.GetCommenter()
	notApprover := !rs.isApprover(commenter)
	for _, cmd := range cmds {
		if cmdBelongsToApprover.Has(cmd) && notApprover {
			rt.client.CreatePRComment(
				rs.org, rs.repo, rs.prNumber,
				op.FormatResponseRaw(
					ne.GetComment(), ne.Comment.HtmlUrl, commenter,
					fmt.Sprintf(
						"These commands such as %s are restricted to approvers in OWNERS files.",
						strings.Join(cmdBelongsToApprover.List(), ", "),
					),
				),
			)

			break
		}
	}

	if !rs.reviewers.Has(commenter) {
		return nil
	}
	return rs.handle(false)
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
