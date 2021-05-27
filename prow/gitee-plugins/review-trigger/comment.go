package reviewtrigger

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/test-infra/prow/gitee"
	"k8s.io/test-infra/prow/github"
	op "k8s.io/test-infra/prow/plugins"
)

func (rt *trigger) newReviewState(ne gitee.PRNoteEvent, log *logrus.Entry) (*reviewState, error) {
	org, repo := ne.GetOrgRep()

	ro, err := rt.oc.LoadRepoOwners(org, repo, ne.PullRequest.Base.Ref)
	if err != nil {
		return nil, err
	}

	filenames, err := rt.client.getPullRequestChanges(org, repo, ne.GetPRNumber())
	if err != nil {
		return nil, err
	}

	dirApproverMap := map[string]sets.String{}
	for _, filename := range filenames {
		dirApproverMap[filename] = ro.Approvers(filename)
	}

	cfg, err := rt.orgRepoConfig(org, repo)
	if err != nil {
		return nil, err
	}

	v := ne.PullRequest.Assignees
	as := make([]string, 0, len(v))
	for i := range v {
		as = append(as, github.NormLogin(v[i].Login))
	}

	s := reviewState{
		org:            org,
		repo:           repo,
		headSHA:        ne.PullRequest.Head.Sha,
		botName:        rt.botName,
		prAuthor:       ne.PullRequest.User.Login,
		prNumber:       ne.GetPRNumber(),
		filenames:      filenames,
		currentLabels:  gitee.GetLabelFromEvent(ne.PullRequest.Labels),
		assignees:      as,
		c:              rt.client,
		cfg:            cfg,
		dirApproverMap: dirApproverMap,
		approverDirMap: parseApprovers(dirApproverMap),
		reviewers:      ro.AllReviewers(),
		owner:          ro,
		log:            log,
	}
	return &s, nil

}

func (rt *trigger) handleReviewComment(ne gitee.PRNoteEvent, log *logrus.Entry) error {
	rs, err := rt.newReviewState(ne, log)
	if err != nil {
		return err
	}

	commenter := github.NormLogin(ne.GetCommenter())
	c := sComment{
		comment: ne.GetComment(),
		author:  commenter,
	}
	cmd, invalidCmd := rs.getCommands(&c)
	if invalidCmd != "" {
		cfg, _ := rt.pluginConfig()
		s := fmt.Sprintf(
			"You can't use command of `/%s`. Please see the [*command usage*](%s) to get detail",
			strings.ToLower(invalidCmd), cfg.Trigger.CommandsLink,
		)
		rt.client.CreatePRComment(
			rs.org, rs.repo, rs.prNumber,
			op.FormatResponseRaw1(c.comment, ne.Comment.HtmlUrl, commenter, s),
		)
	}
	if cmd == "" || !rs.isReviewer(commenter) {
		return nil
	}

	return rs.handle(false, cmd)
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
