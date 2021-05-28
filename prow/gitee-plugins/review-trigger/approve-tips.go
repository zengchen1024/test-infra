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

func createTips(currentApprovers, suggestedApprovers []string) string {
	s := ""
	if len(currentApprovers) > 0 {
		s = fmt.Sprintf("\nIt has been approved by: %s.", strings.Join(currentApprovers, ", "))
	}

	s1 := ""
	if len(suggestedApprovers) > 0 {
		rs := convertReviewers(suggestedApprovers)
		s1 = fmt.Sprintf(
			"\nI suggests these approvers( %s ) to approve your PR.\nYou can assign the PR to them through the command `assign`, for example `/assign @%s`.",
			strings.Join(rs, ", "), suggestedApprovers[0],
		)
	}

	return fmt.Sprintf(
		"%s **NOT APPROVED**\n%s\nIt still needs approval from approvers to be merged.%s",
		notificationTitle, s, s1,
	)

}

func updateTips(label string, approvers, rejecter []string) string {
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

func convertReviewers(v []string) []string {
	rs := make([]string, 0, len(v))
	for _, item := range v {
		rs = append(rs, fmt.Sprintf("[*%s*](https://gitee.com/%s)", item, item))
	}
	return rs
}

func statApprover(reviewComments []*sComment) ([]string, []string) {
	r := map[string][]string{
		cmdReject:  {},
		cmdAPPROVE: {},
	}

	for _, c := range reviewComments {
		if cmdBelongsToApprover.Has(c.comment) {
			r[c.comment] = append(r[c.comment], c.author)
		}
	}

	return r[cmdAPPROVE], r[cmdReject]
}

func findApproveTips(allComments []sdk.PullRequestComments, botName string) *sdk.PullRequestComments {
	for i := range allComments {
		tips := &allComments[i]
		if tips.User == nil || tips.User.Login != botName {
			continue
		}
		if notificationRe.MatchString(tips.Body) {
			return tips
		}
	}
	return nil
}
