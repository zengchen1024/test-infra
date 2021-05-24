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

	filenames, err := rt.client.getPullRequestChanges(org, repo, prNumber)
	if err != nil {
		return nil, nil, err
	}

	m := map[string]sets.String{}
	for _, filename := range filenames {
		dir := filepath.Dir(filename)
		m[dir] = ro.Approvers(filename)
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
		prAuthor:       ne.PullRequest.User.Login,
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
	check := func(cmd string) bool {
		return canApplyCmd(
			cmd, rs.prAuthor == commenter,
			rs.isApprover(commenter), rs.cfg.AllowSelfApprove,
		)
	}

	for _, cmd := range cmds {
		if !check(cmd) {
			cfg, _ := rt.pluginConfig()
			s := fmt.Sprintf(
				"You can't use command of `/%s`. Please see the [*command usage*](%s) to get detail",
				strings.ToLower(cmd), cfg.Trigger.CommandsLink,
			)
			rt.client.CreatePRComment(
				rs.org, rs.repo, rs.prNumber,
				op.FormatResponseRaw1(ne.GetComment(), ne.Comment.HtmlUrl, commenter, s),
			)

			break
		}
	}

	if !rs.isReviewer(commenter) {
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
