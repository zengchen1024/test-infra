package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/labels"
	"k8s.io/test-infra/prow/pluginhelp"
)

const pluginName = "closepr"

func helpProvider(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The colsepr plugin will close PR automatically which has at least one LGTM and Approved labels.",
	}
	return pluginHelp, nil
}

type githubClient interface {
	ClosePR(org, repo string, number int) error
}

type server struct {
	tokenGenerator func() []byte

	ghc githubClient
	log *logrus.Entry

	// Tracks running handlers for graceful shutdown
	wg sync.WaitGroup
}

func (s *server) GracefulShutdown() {
	s.wg.Wait() // Handle remaining requests
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		return
	}
	fmt.Fprint(w, "Event received. Have a nice day.")

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

func (s *server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := s.log.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)

	switch eventType {
	case "pull_request":
		var pr github.PullRequestEvent
		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		pr.GUID = eventGUID
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			if err := s.handlePullRequestEvent(l, pr); err != nil {
				l.WithError(err).Info("Closing pr failed.")
			}
		}()
	default:
		logrus.Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *server) handlePullRequestEvent(l *logrus.Entry, pr github.PullRequestEvent) error {
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  pr.Repo.Owner.Login,
		github.RepoLogField: pr.Repo.Name,
		github.PrLogField:   pr.Number,
		"author":            pr.PullRequest.User.Login,
		"url":               pr.PullRequest.HTMLURL,
	})

	return s.handle(l, &pr)
}

func (s *server) handle(log *logrus.Entry, e *github.PullRequestEvent) error {
	if e.Action != github.PullRequestActionLabeled {
		return nil
	}

	bingo := map[string]bool{labels.LGTM: false, labels.Approved: false}
	for _, l := range e.PullRequest.Labels {
		if _, ok := bingo[l.Name]; ok {
			bingo[l.Name] = true
		}
	}

	if _, ok := bingo[e.Label.Name]; ok {
		bingo[e.Label.Name] = true
	}

	for _, v := range bingo {
		if !v {
			return nil
		}
	}

	pr := &(e.PullRequest)
	org := pr.Base.Repo.Owner.Login
	repo := pr.Base.Repo.Name
	number := pr.Number
	log.Info(fmt.Sprintf("Will close pr: %s/%s:%d", org, repo, number))
	return s.ghc.ClosePR(org, repo, number)
}
