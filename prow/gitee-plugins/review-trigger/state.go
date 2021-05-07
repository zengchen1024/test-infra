package reviewtrigger

import (
	"regexp"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	cmdLGTM    = "LGTM"
	cmdLBTM    = "LBTM"
	cmdAPPROVE = "APPROVE"
	cmdReject  = "REJECT"
)

var (
	validCmds = map[string]bool{
		cmdLGTM: true, cmdLBTM: true, cmdAPPROVE: true, cmdReject: true,
	}
	cmdBelongsToApprover = map[string]bool{
		cmdAPPROVE: true, cmdReject: true,
	}
	commandRegex = regexp.MustCompile(`(?m)^/([^\s]+)[\t ]*([^\n\r]*)`)
)

type sComment struct {
	author  string
	t       time.Time
	comment string
}

type reviewState struct {
	org            string
	repo           string
	headSHA        string
	botName        string
	prNumber       int
	c              ghclient
	cfg            *pluginConfig
	dirApproverMap map[string]sets.String
	approverDirMap map[string]sets.String
	reviewers      sets.String
}

func (rs reviewState) handle(isCIPassed bool) error {
	t, err := rs.c.getPRCodeUpdateTime(rs.org, rs.repo, rs.headSHA)
	if err != nil {
		return err
	}

	comments, err := rs.c.ListPRComments(rs.org, rs.repo, rs.prNumber)
	if err != nil {
		return err
	}

	validComments := rs.filterComments(comments, t)

	label := rs.applyComments(validComments)

	return rs.applyLabel(label, isCIPassed, validComments, comments)
}

func (rs reviewState) getCommands(c *sComment) string {
	cmds := parseCommandFromComment(c.comment)
	if len(cmds) == 0 {
		return ""
	}

	negatives := map[string]bool{
		cmdLBTM:   false,
		cmdReject: false,
	}
	positives := map[string]bool{
		cmdAPPROVE: false,
		cmdLGTM:    false,
	}

	lastCmd := ""
	negativeNum := 0
	positiveNum := 0
	_, isApprover := rs.approverDirMap[c.author]
	for _, cmd := range cmds {
		if cmdBelongsToApprover[cmd] && !isApprover {
			continue
		}
		lastCmd = cmd
		if v, ok := negatives[cmd]; ok {
			if !v {
				negatives[cmd] = true
				negativeNum += 1
			}
		} else {
			if !positives[cmd] {
				positives[cmd] = true
				positiveNum += 1
			}
		}
	}

	if negativeNum == 0 && positiveNum == len(positives) {
		return cmdAPPROVE
	}
	return lastCmd
}

func (rs reviewState) applyApprovedLabel(cls map[string]bool) error {
	toAdd := []string{}

	if !cls[labelApproved] {
		toAdd = append(toAdd, labelApproved)
	}

	if !cls[labelLGTM] {
		toAdd = append(toAdd, labelLGTM)
	}

	if len(toAdd) > 0 {
		rs.c.AddMultiPRLabel(rs.org, rs.repo, rs.prNumber, toAdd)
	}

	toRemove := []string{labelRequestChange, labelCanReview}
	for _, l := range toRemove {
		rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l)
	}

	return nil
}

func (rs reviewState) applyLGTMLabel(cls map[string]bool) error {
	l := labelLGTM
	if !cls[l] {
		rs.c.AddPRLabel(rs.org, rs.repo, rs.prNumber, l)
	}

	for _, l := range []string{labelApproved, labelRequestChange, labelCanReview} {
		rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l)
	}

	return nil
}

func (rs reviewState) applyRequestChangeLabel(cls map[string]bool) error {
	l := labelRequestChange
	if !cls[l] {
		rs.c.AddPRLabel(rs.org, rs.repo, rs.prNumber, l)
	}

	for _, l := range []string{labelApproved, labelLGTM, labelCanReview} {
		rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l)
	}

	return nil
}

func parseCommandFromComment(comment string) []string {
	r := []string{}
	for _, match := range commandRegex.FindAllStringSubmatch(comment, -1) {
		cmd := strings.ToUpper(match[1])
		if validCmds[cmd] {
			r = append(r, cmd)
		}
	}
	return r
}
