package reviewtrigger

import (
	"fmt"
	"regexp"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
)

var (
	notificationTitle = "[Approval Notifier] This Pull-Request is"
	notificationRe    = regexp.MustCompile("^\\[Approval Notifier\\] This Pull-Request")
)

func (rs reviewState) applyLabel(label string, isCIPassed bool, reviewComments []*sComment, allComments []sdk.PullRequestComments) error {
	cls, err := rs.c.getPRCurrentLabels(rs.org, rs.repo, rs.prNumber)
	if err != nil {
		return err
	}

	desc := ""
	switch label {
	case labelRequestChange:
		rs.applyRequestChangeLabel(cls)
		desc = updateTips(label, reviewComments)
	case labelLGTM:
		rs.applyLGTMLabel(cls)
		if isCIPassed || cls[rs.cfg.LabelForCIPassed] {
			desc = createTips(reviewComments)
		}
	case labelApproved:
		rs.applyApprovedLabel(cls)
		desc = updateTips(label, reviewComments)
	}

	tips := findApproveTips(allComments)
	if tips != nil && tips.Body != desc {
		return rs.c.UpdatePRComment(rs.org, rs.repo, int(tips.Id), desc)
	}

	return rs.c.CreatePRComment(rs.org, rs.repo, rs.prNumber, desc)
}

func createTips(reviewComments []*sComment) string {
	approvers, _ := statApprover(reviewComments)

	s := ""
	if len(approvers) > 0 {
		s = fmt.Sprintf("It has been approved by: %s.\n", strings.Join(approvers, ", "))
	}

	return fmt.Sprintf(
		"%s **NOT APPROVED**\n\n%sIt needs approval from approvers to be merged.",
		notificationTitle, s,
	)

}

func updateTips(label string, reviewComments []*sComment) string {
	approvers, rejecter := statApprover(reviewComments)

	desc := ""
	switch label {
	case labelApproved:
		desc = fmt.Sprintf(
			"%s **APPROVED**\n\nIt has been approved by: %s",
			notificationTitle,
			strings.Join(approvers, ", "),
		)

	case labelRequestChange:
		if len(rejecter) > 0 {
			desc = fmt.Sprintf(
				"%s **NOT APPROVED**\n\nIt is rejected by: %s.\nPlease see the comments left by them and do more changes.\nThis pull-request will not be merged until these approvers comment /approve.",
				notificationTitle, strings.Join(rejecter, ", "),
			)
		}
	}

	return desc
}

func statApprover(reviewComments []*sComment) ([]string, []string) {
	r := map[string][]string{
		cmdReject:  []string{},
		cmdAPPROVE: []string{},
	}

	for _, c := range reviewComments {
		if cmdBelongsToApprover[c.comment] {
			r[c.comment] = append(r[c.comment], c.author)
		}
	}

	return r[cmdAPPROVE], r[cmdReject]
}

func findApproveTips(allComments []sdk.PullRequestComments) *sdk.PullRequestComments {
	for i := range allComments {
		tips := &allComments[i]
		if notificationRe.MatchString(tips.Body) {
			return tips
		}
	}
	return nil
}
