package gitee

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/test-infra/prow/github"
)

var (
	jobsResultNotification   = "| Check Name | Result | Details |\n| --- | --- | --- |\n%s\n  <details>base sha:%s\nhead sha: %s</details>"
	jobsResultNotificationRe = regexp.MustCompile(fmt.Sprintf("\\| Check Name \\| Result \\| Details \\|\n\\| --- \\| --- \\| --- \\|\n%s\n  <details>base sha:%s\nhead sha: %s</details>", "([\\s\\S]*)", "(.*)", "(.*)"))
	jobStatusLabelRe         = regexp.MustCompile(`^ci/test-(error|failure|pending|success)$`)
)

type helper struct {
	comment *github.IssueComment
	jsc     *JobsComment
	labels  []string
	baseSHA string
	headSHA string
}

func newHelper(c *ghclient, org, repo string, prNumber int) (*helper, error) {
	pr, err := c.GetGiteePullRequest(org, repo, prNumber)
	if err != nil {
		return nil, err
	}

	comments, err := c.ListIssueComments(org, repo, prNumber)
	if err != nil {
		return nil, err
	}

	jsc := &JobsComment{
		JobsResultNotificationRe: jobsResultNotificationRe,
		JobComment:               jobComment{},
	}

	labels := make([]string, 0, len(pr.Labels))
	for i := range pr.Labels {
		labels = append(labels, pr.Labels[i].Name)
	}

	return &helper{
		comment: jsc.FindComment(c.botname, comments),
		jsc:     jsc,
		labels:  labels,
		baseSHA: pr.Base.Sha,
		headSHA: pr.Head.Sha,
	}, nil
}

func (h *helper) genComment(baseSHA, headSHA string, status *github.Status) string {
	if h.comment == nil {
		s := ""
		if h.shaMatched(baseSHA, headSHA) {
			s = h.jsc.UpdateComment("", status)
		}
		return h.buildComment(s)
	}

	m := jobsResultNotificationRe.FindStringSubmatch(h.comment.Body)

	if h.shaMatched(m[2], m[3]) {
		if h.shaMatched(baseSHA, headSHA) {
			return h.buildComment(h.jsc.UpdateComment(m[1], status))
		}
		return ""
	}

	s := ""
	if h.shaMatched(baseSHA, headSHA) {
		s = h.jsc.UpdateComment("", status)
	}
	return h.buildComment(s)
}

func (h *helper) shaMatched(baseSHA, headSHA string) bool {
	return h.baseSHA == baseSHA && h.headSHA == headSHA
}

func (h *helper) commentID() int {
	if h.comment == nil {
		return -1
	}
	return h.comment.ID
}

func (h *helper) buildComment(s string) string {
	return fmt.Sprintf(jobsResultNotification, s, h.baseSHA, h.headSHA)
}

func (h *helper) genLabel(comment string) string {
	if comment == "" {
		return ""
	}

	m := jobsResultNotificationRe.FindStringSubmatch(comment)
	if m == nil {
		return ""
	}

	ss := h.jsc.ParseComment(m[1])
	if len(ss) == 0 {
		return ""
	}

	statusSet := sets.String{}
	for _, item := range ss {
		statusSet.Insert(item.State)
	}
	return genLabelByJobStatus(statusSet)
}

func (h *helper) updatePRLabel(comment string) ([]string, bool) {
	jobLabels := sets.String{}
	labelSet := sets.String{}
	for _, v := range h.labels {
		if jobStatusLabelRe.MatchString(v) {
			jobLabels.Insert(v)
		} else {
			labelSet.Insert(v)
		}
	}

	if newLabel := h.genLabel(comment); newLabel != "" {
		if jobLabels.Has(newLabel) {
			return nil, false
		}
		labelSet.Insert(newLabel)
		return labelSet.List(), true
	}

	if len(jobLabels) > 0 {
		return labelSet.List(), true
	}
	return nil, false
}

func genLabelByJobStatus(statusSet sets.String) string {
	if statusSet.Has(github.StatusError) {
		return "ci/test-error"
	}
	if statusSet.Has(github.StatusFailure) {
		return "ci/test-failure"
	}
	if statusSet.Has(github.StatusPending) {
		return "ci/test-pending"
	}
	return "ci/test-success"
}
