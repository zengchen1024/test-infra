package reviewtrigger

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	sdk "gitee.com/openeuler/go-gitee/gitee"
)

var (
	notificationTitle   = "[Approval Notifier] This Pull-Request is"
	notificationRe      = regexp.MustCompile("^\\[Approval Notifier\\] This Pull-Request")
	notificationSpliter = "\n\n---\n\n"
)

func doesTipsHasPart2(comment string) bool {
	return strings.Contains(comment, notificationSpliter)
}

func part2(suggestedApprovers []string, oldComment string) string {
	if len(suggestedApprovers) > 0 {
		v := convertReviewers(suggestedApprovers)
		return fmt.Sprintf(
			"%sI suggest these approvers( %s ) to approve your PR.\nYou can assign the PR to them by writing a comment of `/assign @%s`.",
			notificationSpliter, strings.Join(v, ", "), strings.Join(suggestedApprovers, ", @"),
		)
	}

	v := strings.Split(oldComment, notificationSpliter)
	if len(v) == 2 {
		return notificationSpliter + v[1]
	}

	return ""
}

func lgtmTips(currentApprovers, suggestedApprovers []string, oldComment string) string {
	s := ""
	if len(currentApprovers) > 0 {
		v := convertReviewers(currentApprovers)
		s = fmt.Sprintf("\n\nIt has been approved by: %s.\nIt still needs approval from approvers to be merged.", strings.Join(v, ", "))
	}

	return fmt.Sprintf(
		"%s **NOT APPROVED**%s%s",
		notificationTitle, s, part2(suggestedApprovers, oldComment),
	)
}

func requestChangeTips(rejecter []string) string {
	if len(rejecter) == 0 {
		return ""
	}
	v := convertReviewers(rejecter)
	return fmt.Sprintf(
		"%s **Rejected**\n\nIt is rejected by: %s.\nPlease see the comments left by them and do more changes.",
		notificationTitle, strings.Join(v, ", "),
	)
}

func approvedTips(approvers []string) string {
	v := convertReviewers(approvers)
	return fmt.Sprintf(
		"%s **APPROVED**\n\nIt has been approved by: %s",
		notificationTitle,
		strings.Join(v, ", "),
	)
}

func convertReviewers(v []string) []string {
	rs := make([]string, 0, len(v))
	for _, item := range v {
		rs = append(rs, fmt.Sprintf("[*%s*](https://gitee.com/%s)", item, item))
	}
	return rs
}

func statApprover(reviewComments []*sComment) ([]string, []string) {
	rejecters := sets.NewString()
	approvers := sets.NewString()

	for _, c := range reviewComments {
		switch c.comment {
		case cmdReject:
			rejecters.Insert(c.author)
		case cmdAPPROVE:
			approvers.Insert(c.author)
		}
	}

	return approvers.List(), rejecters.List()
}

type approveTips struct {
	tipsID int
	body   string
}

func (a approveTips) exists() bool {
	return a.body != ""
}

func findApproveTips(allComments []sdk.PullRequestComments, botName string) approveTips {
	for i := range allComments {
		tips := &allComments[i]
		if tips.User == nil || tips.User.Login != botName {
			continue
		}
		if notificationRe.MatchString(tips.Body) {
			return approveTips{
				tipsID: int(tips.Id),
				body:   tips.Body,
			}
		}
	}
	return approveTips{}
}
