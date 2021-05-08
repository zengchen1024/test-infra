package reviewtrigger

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/test-infra/prow/gitee"
)

const spliter = "\n"

var (
	jobResultNotificationRe = regexp.MustCompile(fmt.Sprintf("\\|%s\\|%s\\|", "([^|]*)", "([^|]*)"))
)

// parseJobComment return the single job result
// The format of job comment is "| job name | result | detail |"
func parseJobComment(s string) (string, error) {
	m := jobResultNotificationRe.FindStringSubmatch(s)
	if m != nil {
		return m[2], nil
	}

	return "", fmt.Errorf("invalid job comment")
}

type ciCommentParser struct {
	jobsResultNotificationRe *regexp.Regexp
}

func (j ciCommentParser) IsCIComment(comment string) bool {
	return j.jobsResultNotificationRe.MatchString(comment)
}

func (j ciCommentParser) ParseComment(comment string) []string {
	m := j.jobsResultNotificationRe.FindStringSubmatch(comment)
	if m == nil {
		return nil
	}

	cs := strings.Split(m[3], spliter)
	n := len(cs)
	// the first row must be `| --- | --- |`, and omit it.
	if n <= 1 {
		return nil
	}

	r := make([]string, 0, n)
	for i := 1; i < n; i++ {
		if status, err := parseJobComment(cs[i]); err == nil {
			r = append(r, status)
		}
	}
	return r
}

func newCICommentParser(title string) ciCommentParser {
	s := strings.ReplaceAll(title, "|", "\\|")
	s = "(.*)" + s + "(.*)\n([\\s\\S]*)"

	return ciCommentParser{
		jobsResultNotificationRe: regexp.MustCompile(s),
	}
}

func parseCIStatus(cfg *pluginConfig, comment string) string {
	parser := newCICommentParser(cfg.TitleOfCITable)
	if !parser.IsCIComment(comment) {
		return ""
	}

	r := parser.ParseComment(comment)
	running := false
	for _, item := range r {
		if strings.Contains(item, cfg.FailureStatusOfJob) {
			return cfg.FailureStatusOfJob
		}

		if !strings.Contains(item, cfg.SuccessStatusOfJob) {
			running = true
		}
	}

	if running {
		return cfg.runningStatusOfJob
	}

	if len(r) == cfg.NumberOfTestCases {
		return cfg.SuccessStatusOfJob
	}
	return cfg.runningStatusOfJob
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

	errs := newErrors()
	if status == cfg.SuccessStatusOfJob {
		if rs, err := rt.newReviewState(ne); err != nil {
			errs.add(fmt.Sprintf("new review state, err:%s", err.Error()))
		} else {
			if err := rs.handle(true); err != nil {
				errs.add(fmt.Sprintf("working on CI success, err:%s", err.Error()))
			}
		}
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

		if err := updatePRCILabel(ne, l, cfg, rt.client); err != nil {
			errs.add(err.Error())
		}
	}

	return errs.err()
}

func updatePRCILabel(ne gitee.PRNoteEvent, label string, cfg *pluginConfig, client ghclient) error {
	m := gitee.GetLabelFromEvent(ne.PullRequest.Labels)
	if m[label] {
		return nil
	}

	org, repo := ne.GetOrgRep()
	prNumber := ne.GetPRNumber()
	errs := newErrors()
	for _, item := range cfg.labelsForCI() {
		if m[item] {
			if err := client.RemovePRLabel(org, repo, prNumber, item); err != nil {
				errs.add(err.Error())
			}
		}
	}

	if err := client.AddPRLabel(org, repo, prNumber, label); err != nil {
		errs.add(err.Error())
	}
	return errs.err()
}
