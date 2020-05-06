package main

import (
	"fmt"

	"k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/cmd/job-reporter/gitee"
	"k8s.io/test-infra/prow/cmd/job-reporter/github"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/config/secret"
)

const jobPlatformAnnotation = ""

type reportClient interface {
	Report(pj *v1.ProwJob) ([]*v1.ProwJob, error)
	GetName() string
	ShouldReport(pj *v1.ProwJob) bool
}

func buildReporter(o *options, cfg config.Getter) ([]reportClient, error) {
	var rs []reportClient

	var secretAgent secret.Agent
	if err := secretAgent.Start([]string{}); err != nil {
		return rs, fmt.Errorf("Error starting secret agent: %w", err)
	}

	githubReporter, err := newGithubReporter(o, &secretAgent, cfg)
	if err != nil {
		return rs, err
	}
	if githubReporter != nil {
		rs = append(rs, githubReporter)
	}

	giteeReporter, err := newGiteeReporter(o, &secretAgent, cfg)
	if err != nil {
		return rs, err
	}
	if giteeReporter != nil {
		rs = append(rs, giteeReporter)
	}

	return rs, nil
}

func newGithubReporter(o *options, secretAgent *secret.Agent, cfg config.Getter) (reportClient, error) {
	if o.github.TokenPath == "" {
		return nil, nil
	}

	if err := secretAgent.Add(o.github.TokenPath); err != nil {
		return nil, fmt.Errorf("Error reading GitHub credentials: %w", err)
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		return nil, fmt.Errorf("github client: %w", err)
	}

	return github.NewReporter(githubClient, cfg, v1.ProwJobAgent(""), jobPlatformAnnotation), nil
}

func newGiteeReporter(o *options, secretAgent *secret.Agent, cfg config.Getter) (reportClient, error) {
	if o.gitee.TokenPath == "" {
		return nil, nil
	}

	if err := secretAgent.Add(o.gitee.TokenPath); err != nil {
		return nil, fmt.Errorf("Error reading Gitee credentials: %w", err)
	}

	giteeClient, err := o.gitee.GiteeClient(secretAgent, o.dryRun)
	if err != nil {
		return nil, fmt.Errorf("gitee client: %w", err)
	}

	return gitee.NewReporter(giteeClient, cfg, v1.ProwJobAgent(""), jobPlatformAnnotation), nil
}
