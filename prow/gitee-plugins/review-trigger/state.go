package reviewtrigger

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/test-infra/prow/github"
)

type reviewState struct {
	org            string
	repo           string
	headSHA        string
	botName        string
	prAuthor       string
	prNumber       int
	currentLabels  map[string]bool
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
	if len(validComments) == 0 {
		return nil
	}

	label := rs.applyComments(validComments)

	return rs.applyLabel(label, isCIPassed, validComments, comments)
}

func (rs reviewState) applyLabel(label string, isCIPassed bool, reviewComments []*sComment, allComments []sdk.PullRequestComments) error {
	cls := rs.currentLabels
	var err error
	desc := ""
	switch label {
	case labelRequestChange:
		err = rs.applyRequestChangeLabel(cls)
		desc = updateTips(label, reviewComments)
	case labelLGTM:
		err = rs.applyLGTMLabel(cls)
		if isCIPassed || cls[rs.cfg.LabelForCIPassed] {
			desc = createTips(reviewComments)
		}
	case labelApproved:
		err = rs.applyApprovedLabel(cls)
		desc = updateTips(label, reviewComments)
	}
	errs := newErrors()
	errs.addError(err)

	if desc != "" {
		tips := findApproveTips(allComments, rs.botName)
		if tips != nil {
			if tips.Body != desc {
				err := rs.c.UpdatePRComment(rs.org, rs.repo, int(tips.Id), desc)
				errs.addError(err)
			}
		} else {
			err := rs.c.CreatePRComment(rs.org, rs.repo, rs.prNumber, desc)
			errs.addError(err)
		}
	}
	return errs.err()
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
		if !cls[l] {
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
	_, b := rs.approverDirMap[github.NormLogin(author)]
	return b
}

func (rs reviewState) dirsOfApprover(author string) sets.String {
	v, b := rs.approverDirMap[github.NormLogin(author)]
	if b {
		return v
	}

	return sets.String{}
}

func (rs reviewState) isReviewer(author string) bool {
	return rs.reviewers.Has(github.NormLogin(author))
}
