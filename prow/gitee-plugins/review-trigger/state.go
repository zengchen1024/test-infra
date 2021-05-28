package reviewtrigger

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/test-infra/prow/repoowners"
)

type reviewState struct {
	org            string
	repo           string
	headSHA        string
	botName        string
	prAuthor       string
	prNumber       int
	filenames      []string
	currentLabels  map[string]bool
	assignees      []string
	c              ghclient
	cfg            *pluginConfig
	dirApproverMap map[string]sets.String
	approverDirMap map[string]sets.String
	reviewers      sets.String
	owner          repoowners.RepoOwner
	log            *logrus.Entry
}

func (rs reviewState) handle(isCIPassed bool, latestComment string) error {
	t, err := rs.c.getPRCodeUpdateTime(rs.org, rs.repo, rs.headSHA)
	if err != nil {
		return err
	}

	comments, err := rs.c.ListPRComments(rs.org, rs.repo, rs.prNumber)
	if err != nil {
		return err
	}

	validComments := rs.filterComments(comments, t)
	if len(validComments) == 0 {
		return nil
	}

	label := rs.applyComments(validComments)

	return rs.applyLabel(label, validComments, comments, isCIPassed, latestComment)
}

func (rs reviewState) applyLabel(label string, reviewComments []*sComment, allComments []sdk.PullRequestComments, isCIPassed bool, latestComment string) error {
	cls := rs.currentLabels
	tips := findApproveTips(allComments, rs.botName)
	approvers, rejecters := statApprover(reviewComments)

	switch label {
	case labelRequestChange:
		return rs.applyRequestChange(cls, rejecters, tips)
	case labelLGTM:
		return rs.applyLGTM(cls, approvers, isCIPassed, latestComment, tips)
	case labelApproved:
		return rs.applyApproved(cls, approvers, tips)
	}

	return nil
}

func (rs reviewState) applyLGTM(cls map[string]bool, approvers []string, isCIPassed bool, latestComment string, oldTips approveTips) error {
	errs := newErrors()
	err := rs.applyLGTMLabel(cls)
	errs.addError(err)

	if isCIPassed || cls[rs.cfg.LabelForCIPassed] {
		var sa []string
		if !oldTips.exists() || !doesTipsHasPart2(oldTips.body) || latestComment == cmdAPPROVE {
			sa = rs.suggestApprover(approvers)
		}

		desc := lgtmTips(approvers, sa, oldTips.body)
		err = rs.writeApproveTips(desc, oldTips)
		errs.addError(err)
	}

	return errs.err()
}

func (rs reviewState) applyRequestChange(cls map[string]bool, rejecters []string, oldTips approveTips) error {
	errs := newErrors()
	err := rs.applyRequestChangeLabel(cls)
	errs.addError(err)

	if len(rejecters) == 0 {
		if oldTips.exists() {
			err = rs.c.DeletePRComment(rs.org, rs.repo, oldTips.tipsID)
			errs.addError(err)
		}
	} else {
		err = rs.writeApproveTips(requestChangeTips(rejecters), oldTips)
		errs.addError(err)
	}
	return errs.err()
}

func (rs reviewState) applyApproved(cls map[string]bool, approvers []string, oldTips approveTips) error {
	errs := newErrors()
	err := rs.applyApprovedLabel(cls)
	errs.addError(err)

	err = rs.writeApproveTips(approvedTips(approvers), oldTips)
	errs.addError(err)

	return errs.err()
}

func (rs reviewState) writeApproveTips(desc string, oldTips approveTips) error {
	// can't check `if desc == oldTips.body` instead
	if desc == "" || desc == oldTips.body {
		return nil
	}

	if oldTips.exists() {
		return rs.c.UpdatePRComment(rs.org, rs.repo, oldTips.tipsID, desc)
	}
	return rs.c.CreatePRComment(rs.org, rs.repo, rs.prNumber, desc)
}

func (rs reviewState) applyApprovedLabel(cls map[string]bool) error {
	toAdd := []string{}

	if !cls[labelApproved] {
		toAdd = append(toAdd, labelApproved)
	}

	if !cls[labelLGTM] {
		toAdd = append(toAdd, labelLGTM)
	}

	errs := newErrors()

	if len(toAdd) > 0 {
		err := rs.c.AddMultiPRLabel(rs.org, rs.repo, rs.prNumber, toAdd)
		errs.addError(err)
	}

	toRemove := []string{labelRequestChange, labelCanReview}
	for _, l := range toRemove {
		if cls[l] {
			err := rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l)
			errs.addError(err)
		}
	}

	return errs.err()
}

func (rs reviewState) applyLGTMLabel(cls map[string]bool) error {
	errs := newErrors()

	l := labelLGTM
	if !cls[l] {
		if err := rs.c.AddPRLabel(rs.org, rs.repo, rs.prNumber, l); err != nil {
			errs.addError(err)
		} else {
			err := rs.c.CreatePRComment(
				rs.org, rs.repo, rs.prNumber, "lgtm label has been added.",
			)
			errs.addError(err)
		}
	}

	for _, l := range []string{labelApproved, labelRequestChange, labelCanReview} {
		if cls[l] {
			err := rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l)
			errs.addError(err)
		}
	}

	return errs.err()
}

func (rs reviewState) applyRequestChangeLabel(cls map[string]bool) error {
	errs := newErrors()

	l := labelRequestChange
	if !cls[l] {
		err := rs.c.AddPRLabel(rs.org, rs.repo, rs.prNumber, l)
		errs.addError(err)
	}

	for _, l := range []string{labelApproved, labelLGTM, labelCanReview} {
		if cls[l] {
			err := rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l)
			errs.addError(err)
		}
	}

	return errs.err()
}

func (rs reviewState) isApprover(author string) bool {
	_, b := rs.approverDirMap[author]
	return b
}

func (rs reviewState) dirsOfApprover(author string) sets.String {
	v, b := rs.approverDirMap[author]
	if b {
		return v
	}

	return sets.String{}
}

func (rs reviewState) isReviewer(author string) bool {
	return rs.reviewers.Has(author)
}

func (rs reviewState) suggestApprover(currentApprovers []string) []string {
	ah := approverHelper{
		currentApprovers:  currentApprovers,
		assignees:         rs.assignees,
		filenames:         rs.filenames,
		prNumber:          rs.prNumber,
		numberOfApprovers: rs.cfg.NumberOfApprovers,
		repoOwner:         rs.owner,
		prAuthor:          rs.prAuthor,
		allowSelfApprove:  rs.cfg.AllowSelfApprove,
		log:               rs.log,
	}
	return ah.suggestApprovers()
}
