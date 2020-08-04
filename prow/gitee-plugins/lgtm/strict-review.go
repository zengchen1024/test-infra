package lgtm

import (
	"fmt"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/plugins"
	originl "k8s.io/test-infra/prow/plugins/lgtm"
	"k8s.io/test-infra/prow/repoowners"
)

func HandleStrictLGTMPREvent(gc *ghclient, e *github.PullRequestEvent) error {
	pr := e.PullRequest
	sha := pr.Head.SHA
	org := pr.Base.Repo.Owner.Login
	repo := pr.Base.Repo.Name
	prNumber := pr.Number

	var n *notification
	needRemoveLabel := false

	switch e.Action {
	case github.PullRequestActionOpened:
		n = &notification{
			headSHA: sha,
		}

	case github.PullRequestActionSynchronize:
		v, prChanged, err := LoadLGTMnotification(gc, org, repo, prNumber, sha)
		if err != nil {
			return err
		}

		if !prChanged {
			return nil
		}

		n = v
		needRemoveLabel = true

	default:
		return nil
	}

	filenames, err := originl.GetChangedFiles(gc, org, repo, prNumber)
	if err != nil {
		return err
	}

	n.ResetDirs(genDirs(filenames))

	if err := n.WriteComment(gc, org, repo, prNumber, false); err != nil {
		return err
	}

	if needRemoveLabel {
		return gc.RemoveLabel(org, repo, prNumber, originl.LGTMLabel)
	}
	return nil
}

// skipCollaborators && strictReviewer
func HandleStrictLGTMComment(gc *ghclient, oc repoowners.Interface, log *logrus.Entry, wantLGTM bool, e *sdk.NoteEvent) error {
	pr := e.PullRequest
	s := &strictReview{
		gc:  gc,
		oc:  oc,
		log: log,

		org:      e.Repository.Namespace,
		repo:     e.Repository.Name,
		headSHA:  pr.Head.Sha,
		prAuthor: pr.Head.User.Login,
		prNumber: int(pr.Number),
	}

	/*
		commenter := e.Comment.User.Login
		if commenter != s.prAuthor {
			bingo := false
			for _, v := range pr.Assignees {
				if v.Login == commenter {
					bingo = true
					break
				}
			}
			if !bingo {
				err := gc.AssignIssue(s.org, s.repo, s.prNumber, []string{commenter})
				if err != nil {
					return err
				}
			}
		}
	*/

	noti, _, err := LoadLGTMnotification(gc, s.org, s.repo, s.prNumber, s.headSHA)
	if err != nil {
		return err
	}

	validReviewers, err := s.fileReviewers()
	if err != nil {
		return err
	}

	hasLGTM, err := s.hasLGTMLabel()
	if err != nil {
		return err
	}

	if !wantLGTM {
		return s.handleLGTMCancel(noti, validReviewers, e, hasLGTM)
	}

	return s.handleLGTM(noti, validReviewers, e, hasLGTM)
}

type strictReview struct {
	log *logrus.Entry
	gc  *ghclient
	oc  repoowners.Interface

	org      string
	repo     string
	headSHA  string
	prAuthor string
	prNumber int
}

func (this *strictReview) handleLGTMCancel(noti *notification, validReviewers map[string]sets.String, e *sdk.NoteEvent, hasLabel bool) error {
	commenter := e.Comment.User.Login

	ok := canRemoveLgtmLabel(validReviewers, commenter)
	this.log.Infof("commenter=%s, ok=%v, reviewers=%#v", commenter, ok, validReviewers)
	if commenter != this.prAuthor && !ok {
		noti.AddOpponent(commenter)

		return noti.WriteComment(this.gc, this.org, this.repo, this.prNumber, hasLabel)
	}

	if commenter == this.prAuthor {
		noti.ResetConsentor()
		noti.ResetOpponents()
	} else {
		noti.AddOpponent(commenter)
	}

	filenames := make([]string, 0, len(validReviewers))
	for k := range validReviewers {
		filenames = append(filenames, k)
	}
	noti.ResetDirs(genDirs(filenames))

	err := noti.WriteComment(this.gc, this.org, this.repo, this.prNumber, false)
	if err != nil {
		return err
	}

	if hasLabel {
		return this.gc.RemovePRLabel(this.org, this.repo, this.prNumber, originl.LGTMLabel)
	}
	return nil
}

func (this *strictReview) handleLGTM(noti *notification, validReviewers map[string]sets.String, e *sdk.NoteEvent, hasLabel bool) error {
	comment := e.Comment
	commenter := comment.User.Login

	if commenter == this.prAuthor {
		resp := "you cannot LGTM your own PR."
		this.log.Infof("Commenting with \"%s\".", resp)
		return this.gc.CreateComment(
			this.org, this.repo, this.prNumber,
			plugins.FormatResponseRaw(comment.Body, comment.HtmlUrl, commenter, resp))
	}

	noti.AddConsentor(commenter)

	isReviewer, canAdd, dirs := canAddLgtmLabel(validReviewers, commenter, noti.GetConsentors())
	if !isReviewer {
		return noti.WriteComment(this.gc, this.org, this.repo, this.prNumber, hasLabel)
	}

	if canAdd {
		noti.ResetDirs([]string{})
		if err := noti.WriteComment(this.gc, this.org, this.repo, this.prNumber, true); err != nil {
			return err
		}
		// add label
		if !hasLabel {
			if err := this.gc.AddLabel(this.org, this.repo, this.prNumber, originl.LGTMLabel); err != nil {
				return err
			}
		}
		return nil
	}

	noti.ResetDirs(dirs)
	return noti.WriteComment(this.gc, this.org, this.repo, this.prNumber, false)
}

func (this *strictReview) fileReviewers() (map[string]sets.String, error) {
	ro, err := originl.LoadRepoOwners(this.gc, this.oc, this.org, this.repo, this.prNumber)
	if err != nil {
		return nil, err
	}

	filenames, err := originl.GetChangedFiles(this.gc, this.org, this.repo, this.prNumber)
	if err != nil {
		return nil, err
	}

	m := map[string]sets.String{}
	for _, filename := range filenames {
		m[filename] = ro.Approvers(filename).Union(ro.Reviewers(filename))
	}

	return m, nil
}

func (this *strictReview) hasLGTMLabel() (bool, error) {
	labels, err := this.gc.GetIssueLabels(this.org, this.repo, this.prNumber)
	if err != nil {
		return false, fmt.Errorf("Failed to get pr labels, err:%s", err.Error())
	}
	return github.HasLabel(originl.LGTMLabel, labels), nil
}

func canAddLgtmLabel(validReviewers map[string]sets.String, commenter string, reviewers []string) (bool, bool, []string) {
	isReviewer := false
	needReview := map[string]bool{}
	commenter = github.NormLogin(commenter)

	reviewers1 := make([]string, 0, len(reviewers))
	for _, v := range reviewers {
		reviewers1 = append(reviewers1, github.NormLogin(v))
	}

	for filename, rs := range validReviewers {
		if rs.Has(commenter) {
			isReviewer = true
		} else if !rs.HasAny(reviewers1...) {
			needReview[filename] = true
		}
	}

	if !isReviewer {
		// not reviewer
		return false, false, nil
	}

	if len(needReview) == 0 {
		// can add label
		return true, true, nil
	}

	//return dir to find reviewer
	return true, false, genDirs(mapKeys(needReview))
}

func canRemoveLgtmLabel(validReviewers map[string]sets.String, commenter string) bool {
	commenter = github.NormLogin(commenter)

	for _, rs := range validReviewers {
		if rs.Has(commenter) {
			return true
		}
	}

	return false
}
