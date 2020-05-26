package main

import (
	"bytes"
	"fmt"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"

	"k8s.io/test-infra/prow/gitee"
	"k8s.io/test-infra/prow/github"
)

func (s *server) pushToGitee(destOrg, repo, localBranch, localDir string, spr *github.PullRequest) (string, error) {
	// Push to dest repo
	r, err := s.gegc.ClientFromDir(s.robot, repo, localDir)
	if err != nil {
		return "", err
	}

	if err := r.ForcePush(localBranch); err != nil {
		return "", err
	}

	// create or update pr
	head := fmt.Sprintf("%s:%s", s.robot, localBranch)
	return s.createOrUpdatePR(spr, destOrg, repo, head)
}

func (s *server) createOrUpdatePR(spr *github.PullRequest, destOrg, repo, head string) (string, error) {
	syncDesc, err := s.syncDescOnGitee(spr)
	if err != nil {
		return "", err
	}

	body := fmt.Sprintf("%s\n\n%s", spr.Body, syncDesc)

	dpr, err := s.queryPR(destOrg, repo, head)
	if err != nil {
		return "", err
	}

	if dpr == nil {
		pr, err := s.gec.CreatePullRequest(destOrg, repo, spr.Title, body, head, "master", false)
		if err != nil {
			return "", err
		}
		return s.syncDescOnGithub(&pr)
	}

	if strings.Compare(dpr.Body, body) != 0 {
		_, err = s.gec.UpdatePullRequest(destOrg, repo, dpr.Number, "", body, "", "")
		if err != nil {
			return "", err
		}
	}

	return s.syncDescOnGithub(dpr)
}

func (s *server) queryPR(org, repo, head string) (*sdk.PullRequest, error) {
	opt := gitee.ListPullRequestOpt{
		State: "open",
		Head:  head,
		Base:  "master",
	}

	prs, err := s.gec.GetPullRequests(org, repo, opt)
	if err != nil {
		return nil, err
	}

	switch len(prs) {
	case 0:
		return nil, nil
	case 1:
		return &prs[0], nil
	}

	return nil, fmt.Errorf("There are more than one prs in repo(%s/%s) which are open and created by %s", org, repo, opt.Head)
}

func (s *server) syncDescOnGitee(pr *github.PullRequest) (string, error) {
	var b bytes.Buffer
	err := s.config().GiteeCommentTemplate.Execute(&b, pr)

	if err != nil {
		return "", fmt.Errorf("error executing URL template: %v", err)
	}

	return b.String(), nil
}

func (s *server) syncDescOnGithub(pr *sdk.PullRequest) (string, error) {
	var b bytes.Buffer
	err := s.config().GithubCommentTemplate.Execute(&b, pr)

	if err != nil {
		return "", fmt.Errorf("error executing URL template: %v", err)
	}

	return b.String(), nil
}
