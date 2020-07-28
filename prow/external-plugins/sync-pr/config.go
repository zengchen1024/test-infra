package main

import (
	"fmt"
	"io/ioutil"
	"text/template"

	"sigs.k8s.io/yaml"
)

// Sync holds configuration for the sync plugin
type syncPRConfig struct {
	// GithubComment is the comment template to show where the pr is synchronized to
	GithubComment         string             `json:"github_comment_template,omitempty"`
	GithubCommentTemplate *template.Template `json:"-"`

	// GiteeComment is the comment template to show where the pr is synchronized from
	GiteeComment         string             `json:"gitee_comment_template,omitempty"`
	GiteeCommentTemplate *template.Template `json:"-"`

	// DestOrg is the destination organization to which the pr is synchronized. It is
	// the map between source org/repo to org.
	DestOrg map[string]string `json:"dest_org,omitempty"`

	// MiddleRepo is the name of repo which the robot of gitee forked from the dest
	// gitee repo. It is the map between source(github) org/repo to repo.
	MiddleRepo map[string]string `json:"middle_repo,omitempty"`
}

// syncPRConfigFor finds the dest org for a repo
func (s syncPRConfig) syncPRFor(org, repo string) string {
	if s.DestOrg == nil {
		return ""
	}

	if v, ok := s.DestOrg[org]; ok {
		return v
	}

	orgRepo := fmt.Sprintf("%s/%s", org, repo)
	if v, ok := s.DestOrg[orgRepo]; ok {
		return v
	}

	return ""
}

// middleRepo finds the middle repo
func (s syncPRConfig) middleRepo(org, repo string) string {
	if s.MiddleRepo == nil {
		return repo
	}

	orgRepo := fmt.Sprintf("%s/%s", org, repo)
	if v, ok := s.MiddleRepo[orgRepo]; ok {
		return v
	}

	return repo
}

func (s *syncPRConfig) validate() error {
	if s.GithubComment == "" {
		return fmt.Errorf("github_comment_template must be set")
	}

	tmpl, err := template.New("github").Parse(s.GithubComment)
	if err != nil {
		return fmt.Errorf("parsing template: %v", err)
	}
	s.GithubCommentTemplate = tmpl

	// gitee
	if s.GiteeComment == "" {
		return fmt.Errorf("gitee_comment_template must be set")
	}

	tmpl, err = template.New("gitee").Parse(s.GiteeComment)
	if err != nil {
		return fmt.Errorf("parsing template: %v", err)
	}
	s.GiteeCommentTemplate = tmpl

	return nil
}

func load(path string) (syncPRConfig, error) {
	var s struct {
		SyncPR syncPRConfig `json:"syncpr,omitempty"`
	}
	c := s.SyncPR

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return c, err
	}

	if err := yaml.Unmarshal(b, &s); err != nil {
		return c, err
	}

	if err := (&s.SyncPR).validate(); err != nil {
		return c, err
	}

	return s.SyncPR, nil
}
