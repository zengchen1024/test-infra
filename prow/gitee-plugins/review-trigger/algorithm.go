package reviewtrigger

import (
	"sort"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
)

type record struct {
	cmdLGTMNum    int
	cmdLBTMNum    int
	cmdAPPROVENum int
	cmdRejectNum  int
}

func (r *record) update(cmd string) {
	switch cmd {
	case cmdLBTM:
		r.cmdLBTMNum += 1
	case cmdLGTM:
		r.cmdLGTMNum += 1
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

	if r.cmdAPPROVENum > 0 || (r.cmdLGTMNum-r.cmdLBTMNum) > 0 {
		return labelLGTM
	}
	return labelRequestChange
}

func (rs reviewState) applyComments(comments []*sComment) string {
	records := map[string]*record{}
	for dir := range rs.dirApproverMap {
		v := record{}
		records[dir] = &v
	}

	m := map[string]string{
		cmdAPPROVE: cmdLGTM,
		cmdReject:  cmdLBTM,
	}

	for _, c := range comments {
		cmd := c.comment
		if cmdBelongsToApprover[cmd] {
			dirs := rs.approverDirMap[c.author]

			cmd1 := m[cmd]
			for k := range records {
				if dirs.Has(k) {
					records[k].update(cmd)
				} else {
					records[k].update(cmd1)
				}
			}
		} else {
			for k := range records {
				records[k].update(cmd)
			}
		}
	}

	r := map[string]int{
		labelRequestChange: 0,
		labelApproved:      0,
		labelLGTM:          0,
	}

	approveNum := rs.cfg.NumberOfApprovers
	for _, v := range records {
		r[v.inferLabel(approveNum)] += 1
	}

	if r[labelRequestChange] > 0 {
		return labelRequestChange
	}

	if r[labelLGTM] > 0 {
		return labelLGTM
	}

	return labelApproved
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

		if c.User == nil || c.User.Login == rs.botName || !rs.reviewers.Has(c.User.Login) {
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

	m := map[string]bool{}
	n := len(newComments)
	validComments := make([]*sComment, 0, n)
	for i := n - 1; i >= 0; i-- {
		c := &newComments[i]
		if m[c.author] {
			continue
		}
		if cmd := rs.getCommands(c); cmd != "" {
			c.comment = cmd
			m[c.author] = true
			validComments = append(validComments, c)
		}
	}

	return validComments
}
