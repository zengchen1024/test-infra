package reviewtrigger

import (
	"fmt"
	"regexp"
	"strings"
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
	for _, item := range r {
		if strings.Contains(item, cfg.FailureStatusOfJob) {
			return cfg.FailureStatusOfJob
		}

		if !strings.Contains(item, cfg.SuccessStatusOfJob) {
			return cfg.runningStatusOfJob
		}
	}

	if len(r) == cfg.NumberOfTestCases {
		return cfg.SuccessStatusOfJob
	}

	return cfg.runningStatusOfJob
}
