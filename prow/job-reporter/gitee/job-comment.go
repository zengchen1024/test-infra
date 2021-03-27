package gitee

import (
	"fmt"
	"regexp"

	"k8s.io/test-infra/prow/github"
)

var (
	jobResultNotification   = "| %s %s | %s | [details](%s) |"
	jobResultNotificationRe = regexp.MustCompile(fmt.Sprintf("\\| %s %s \\| %s \\| \\[details\\]\\(%s\\) \\|", "(.*)", "(.*)", "(.*)", "(.*)"))
)

type JobComment interface {
	GenJobComment(s *github.Status) string
	ParseJobComment(s string) (github.Status, error)
}

type jobComment struct{}

func (j jobComment) GenJobComment(s *github.Status) string {
	icon := stateToIcon(s.State)
	return fmt.Sprintf(jobResultNotification, icon, s.Context, s.Description, s.TargetURL)
}

func (j jobComment) ParseJobComment(s string) (github.Status, error) {
	m := jobResultNotificationRe.FindStringSubmatch(s)
	if m != nil {
		return github.Status{
			State:       iconToState(m[1]),
			Context:     m[2],
			Description: m[3],
			TargetURL:   m[4],
		}, nil
	}

	return github.Status{}, fmt.Errorf("invalid job comment")
}
