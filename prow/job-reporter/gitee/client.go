package gitee

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"k8s.io/test-infra/prow/gitee"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/report"
)

var (
	JobsResultNotification   = "Checks Results.\n%s\n  <details>Git tree hash: %s</details>"
	JobsResultNotificationRe = regexp.MustCompile(fmt.Sprintf(JobsResultNotification, "(.*)", "(.*)"))
	jobResultNotification    = "%s %s — %s [Details](%s)"
	jobResultNotificationRe  = regexp.MustCompile(fmt.Sprintf("%s %s — %s \\[Details\\]\\(%s\\)", ".*", "(.*)", ".*", ".*"))
	jobResultEachPartRe      = regexp.MustCompile(fmt.Sprintf("%s %s — %s \\[Details\\]\\(%s\\)", "(.*)", "(.*)", "(.*)", "(.*)"))
)

type giteeClient interface {
	BotName() (string, error)
	ListPRComments(org, repo string, number int) ([]sdk.PullRequestComments, error)
	CreatePRComment(org, repo string, number int, comment string) error
	DeletePRComment(org, repo string, ID int) error
	UpdatePRComment(org, repo string, commentID int, comment string) error
	GetGiteePullRequest(org, repo string, number int) (sdk.PullRequest, error)
}

var _ report.GitHubClient = (*ghclient)(nil)

type ghclient struct {
	giteeClient
	prNumber int
}

func (c *ghclient) ListIssueComments(org, repo string, number int) ([]github.IssueComment, error) {
	var r []github.IssueComment

	v, err := c.ListPRComments(org, repo, number)
	if err != nil {
		return r, err
	}

	for _, i := range v {
		r = append(r, gitee.ConvertGiteePRComment(i))
	}

	sort.SliceStable(r, func(i, j int) bool {
		return r[i].CreatedAt.Before(r[j].CreatedAt)
	})

	return r, nil
}

func (c *ghclient) CreateComment(owner, repo string, number int, comment string) error {
	return c.CreatePRComment(owner, repo, number, comment)
}

func (c *ghclient) DeleteComment(org, repo string, id int) error {
	return c.DeletePRComment(org, repo, id)
}

func (c *ghclient) EditComment(org, repo string, ID int, comment string) error {
	return c.UpdatePRComment(org, repo, ID, comment)
}

func (c *ghclient) CreateStatus(org, repo, ref string, s github.Status) error {
	prNumber := c.prNumber
	var err error
	if prNumber <= 0 {
		prNumber, err = parsePRNumber(org, repo, s)
		if err != nil {
			return err
		}
	}

	comments, err := c.ListIssueComments(org, repo, prNumber)
	if err != nil {
		return err
	}

	botname, err := c.BotName()
	if err != nil {
		return err
	}

	jobsOldComment, commentId := findCheckResultComment(botname, ref, comments)

	desc := genJobResultComment(jobsOldComment, ref, s)

	if jobsOldComment == "" {
		return c.CreatePRComment(org, repo, prNumber, desc)
	}
	return c.UpdatePRComment(org, repo, commentId, desc)
}

func parsePRNumber(org, repo string, s github.Status) (int, error) {
	re := regexp.MustCompile(fmt.Sprintf("http.*/%s_%s/(.*)/%s/.*", org, repo, s.Context))
	m := re.FindStringSubmatch(s.TargetURL)
	if m != nil {
		return strconv.Atoi(m[1])
	}
	return 0, fmt.Errorf("Can't parse pr number from url:%s", s.TargetURL)
}

func findCheckResultComment(botname, sha string, comments []github.IssueComment) (string, int) {
	for i := len(comments) - 1; i >= 0; i-- {
		comment := comments[i]
		if comment.User.Login != botname {
			continue
		}

		m := JobsResultNotificationRe.FindStringSubmatch(comment.Body)
		if m != nil && m[2] == sha {
			return m[1], comment.ID
		}
	}

	return "", -1
}

func buildJobResultComment(s github.Status) string {
	icon := stateToIcon(s.State)
	return fmt.Sprintf(jobResultNotification, icon, s.Context, s.Description, s.TargetURL)
}

func stateToIcon(state string) string {
	icon := ""
	switch state {
	case github.StatusPending:
		icon = ":large_blue_circle:"
	case github.StatusSuccess:
		icon = ":white_check_mark:"
	case github.StatusFailure:
		icon = ":x:"
	case github.StatusError:
		icon = ":heavy_minus_sign:"
	}
	return icon
}

func iconToState(icon string) string {
	switch icon {
	case ":large_blue_circle:":
		return github.StatusPending
	case ":white_check_mark:":
		return github.StatusSuccess
	case ":x:":
		return github.StatusFailure
	case ":heavy_minus_sign:":
		return github.StatusError
	}
	return ""
}

func genJobResultComment(jobsOldComment, sha string, jobStatus github.Status) string {
	jobComment := buildJobResultComment(jobStatus)

	if jobsOldComment == "" {
		return fmt.Sprintf(JobsResultNotification, jobComment, sha)
	}

	jobName := jobStatus.Context
	spliter := "\n"
	js := strings.Split(jobsOldComment, spliter)
	bingo := false
	for i, s := range js {
		m := jobResultNotificationRe.FindStringSubmatch(s)
		if m != nil && m[1] == jobName {
			js[i] = jobComment
			bingo = true
			break
		}
	}
	if !bingo {
		js = append(js, jobComment)
	}

	return fmt.Sprintf(JobsResultNotification, strings.Join(js, spliter), sha)
}

func ParseCombinedStatus(botname, sha string, comments []github.IssueComment) []github.Status {
	jobsComment, _ := findCheckResultComment(botname, sha, comments)
	if jobsComment == "" {
		return []github.Status{}
	}

	js := strings.Split(jobsComment, "\n")
	r := make([]github.Status, 0, len(js))
	for _, s := range js {
		m := jobResultEachPartRe.FindStringSubmatch(s)
		if m != nil {
			r = append(r, github.Status{
				State:       iconToState(m[1]),
				Context:     m[2],
				Description: m[3],
				TargetURL:   m[4],
			})
		}
	}
	return r
}
