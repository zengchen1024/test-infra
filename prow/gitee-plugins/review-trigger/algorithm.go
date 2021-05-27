package reviewtrigger

import (
	"regexp"
	"sort"
	"strings"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	cmdLGTM    = "LGTM"
	cmdLBTM    = "LBTM"
	cmdAPPROVE = "APPROVE"
	cmdReject  = "REJECT"
)

var (
	validCmds            = sets.NewString(cmdLGTM, cmdLBTM, cmdAPPROVE, cmdReject)
	negativeCmds         = sets.NewString(cmdReject, cmdLBTM)
	positiveCmds         = sets.NewString(cmdAPPROVE, cmdLGTM)
	cmdBelongsToApprover = sets.NewString(cmdAPPROVE, cmdReject)
	commandRegex         = regexp.MustCompile(`(?m)^/([^\s]+)[\t ]*([^\n\r]*)`)
)

type sComment struct {
	author  string
	t       time.Time
	comment string
}

func parseCommandFromComment(comment string) []string {
	r := []string{}
	for _, match := range commandRegex.FindAllStringSubmatch(comment, -1) {
		cmd := strings.ToUpper(match[1])
		if validCmds.Has(cmd) {
			r = append(r, cmd)
		}
	}
	return r
}

func canApplyCmd(cmd string, isPRAuthor, isApprover, allowSelfApprove bool) bool {
	switch cmd {
	case cmdReject:
		return isApprover && !isPRAuthor
	case cmdLGTM:
		return !isPRAuthor
	case cmdAPPROVE:
		return isApprover && (allowSelfApprove || !isPRAuthor)
	}
	return true
}

// first. filter comments and omit each one
// which is before the pr code update time
// or which is not a reviewer
// or which is commented by bot
//
// second sort the comments by updated time in aesc
func (rs reviewState) preTreatComments(comments []sdk.PullRequestComments, startTime time.Time) []sComment {
	r := make([]sComment, 0, len(comments))
	for i := range comments {
		c := &comments[i]

		if c.User == nil || c.User.Login == rs.botName {
			continue
		}

		ut, err := time.Parse(time.RFC3339, c.UpdatedAt)
		if err != nil || ut.Before(startTime) {
			continue
		}

		r = append(r, sComment{
			author:  c.User.Login,
			t:       ut,
			comment: c.Body,
		})
	}

	sort.SliceStable(r, func(i, j int) bool {
		return r[i].t.Before(r[j].t)
	})

	return r
}

func (rs reviewState) filterComments(comments []sdk.PullRequestComments, startTime time.Time) []*sComment {
	newComments := rs.preTreatComments(comments, startTime)

	done := map[string]bool{}
	n := len(newComments)
	validComments := make([]*sComment, 0, n)
	for i := n - 1; i >= 0; i-- {
		c := &newComments[i]
		if !rs.isReviewer(c.author) || done[c.author] {
			continue
		}

		if cmd, _ := rs.getCommands(c); cmd != "" {
			c.comment = cmd
			validComments = append(validComments, c)
			done[c.author] = true
		}
	}

	return validComments
}

func (rs reviewState) getCommands(c *sComment) (string, string) {
	cmds := parseCommandFromComment(c.comment)
	if len(cmds) == 0 {
		return "", ""
	}

	check := func(cmd string) bool {
		return canApplyCmd(
			cmd, rs.prAuthor == c.author,
			rs.isApprover(c.author), rs.cfg.AllowSelfApprove,
		)
	}

	lastCmd := ""
	invalidCmd := ""
	negatives := map[string]bool{}
	positives := map[string]bool{}
	for _, cmd := range cmds {
		if !check(cmd) {
			if invalidCmd == "" {
				invalidCmd = cmd
			}
			continue
		}

		lastCmd = cmd
		if negativeCmds.Has(cmd) {
			negatives[cmd] = true
		}
		if positiveCmds.Has(cmd) {
			positives[cmd] = true
		}
	}

	if len(negatives) == 0 && len(positiveCmds) == len(positives) {
		return cmdAPPROVE, invalidCmd
	}
	return lastCmd, invalidCmd
}

func (rs reviewState) applyComments(comments []*sComment) string {
	records := map[string]*record{}
	for dir := range rs.dirApproverMap {
		v := record{}
		records[dir] = &v
	}

	for _, c := range comments {
		cmd := c.comment
		if cmdBelongsToApprover.Has(cmd) {
			for k := range rs.dirsOfApprover(c.author) {
				records[k].update(cmd)
			}
		}
	}

	r := map[string]int{
		labelRequestChange: 0,
		labelApproved:      0,
	}
	for _, v := range records {
		l := v.inferLabel(rs.cfg.NumberOfApprovers)
		if l != "" {
			r[l] += 1
		}
	}

	if r[labelRequestChange] > 0 {
		return labelRequestChange
	}

	if r[labelApproved] == len(records) {
		return labelApproved
	}

	// At this point, the pr has not been approved, the label of pr
	// can be determind by the last command of reviewer.
	if positiveCmds.Has(comments[0].comment) {
		return labelLGTM
	}
	return labelRequestChange
}

type record struct {
	cmdAPPROVENum int
	cmdRejectNum  int
}

func (r *record) update(cmd string) {
	switch cmd {
	case cmdAPPROVE:
		r.cmdAPPROVENum += 1
	case cmdReject:
		r.cmdRejectNum += 1
	}
}

func (r *record) inferLabel(approveNum int) string {
	if r.cmdRejectNum > 0 {
		return labelRequestChange
	}

	if r.cmdAPPROVENum >= approveNum {
		return labelApproved
	}

	return ""
}
