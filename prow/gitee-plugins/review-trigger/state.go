package reviewtrigger

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
	"k8s.io/apimachinery/pkg/util/sets"
)

type reviewState struct {
	org            string
	repo           string
	headSHA        string
	botName        string
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
	if err != nil {
		errs.addError(err)
	}

	if desc != "" {
		tips := findApproveTips(allComments, rs.botName)
		if tips != nil {
			if tips.Body != desc {
				err := rs.c.UpdatePRComment(rs.org, rs.repo, int(tips.Id), desc)
				if err != nil {
					errs.addError(err)
				}
			}
		} else {
			if err := rs.c.CreatePRComment(rs.org, rs.repo, rs.prNumber, desc); err != nil {
				errs.addError(err)
			}
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
		if err := rs.c.AddMultiPRLabel(rs.org, rs.repo, rs.prNumber, toAdd); err != nil {
			errs.addError(err)
		}
	}

	toRemove := []string{labelRequestChange, labelCanReview}
	for _, l := range toRemove {
		if !cls[l] {
			continue
		}
		if err := rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l); err != nil {
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
			if err != nil {
				errs.addError(err)
			}
		}
	}

	for _, l := range []string{labelApproved, labelRequestChange, labelCanReview} {
		if !cls[l] {
			continue
		}
		if err := rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l); err != nil {
			errs.addError(err)
		}
	}

	return errs.err()
}

func (rs reviewState) applyRequestChangeLabel(cls map[string]bool) error {
	errs := newErrors()

	l := labelRequestChange
	if !cls[l] {
		if err := rs.c.AddPRLabel(rs.org, rs.repo, rs.prNumber, l); err != nil {
			errs.addError(err)
		}
	}

	for _, l := range []string{labelApproved, labelLGTM, labelCanReview} {
		if !cls[l] {
			continue
		}
		if err := rs.c.RemovePRLabel(rs.org, rs.repo, rs.prNumber, l); err != nil {
			errs.addError(err)
		}
	}

	return errs.err()
}

func (rs reviewState) isApprover(author string) bool {
	_, b := rs.approverDirMap[author]
	return b
}
