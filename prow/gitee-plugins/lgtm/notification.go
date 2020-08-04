package lgtm

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	originl "k8s.io/test-infra/prow/plugins/lgtm"
)

const (
	consentientDesc = "**LGTM**"
	opposedDesc     = "**NOT LGTM**"
	separator       = ", "
	dirSepa         = "\n- "
)

var (
	notificationStr   = "LGTM NOTIFIER: This PR is %s.\n\nReviewers added `/lgtm` are: %s.\n\nReviewers added `/lgtm cancel` are: %s.\n\nIt still needs review for the codes in each of these directoris:%s\n<details>Git tree hash: %s</details>"
	notificationStrRe = regexp.MustCompile(fmt.Sprintf(notificationStr, "(.*)", "(.*)", "(.*)", "([\\s\\S]*)", "(.*)"))
)

type notification struct {
	consentors map[string]bool
	opponents  map[string]bool
	dirs       []string
	headSHA    string
	commentID  int
}

func (this *notification) GetConsentors() []string {
	return mapKeys(this.consentors)
}

func (this *notification) ResetConsentor() {
	this.consentors = map[string]bool{}
}

func (this *notification) ResetOpponents() {
	this.opponents = map[string]bool{}
}

func (this *notification) AddConsentor(consentor string) {
	this.consentors[consentor] = true
	if this.opponents[consentor] {
		delete(this.opponents, consentor)
	}
}

func (this *notification) AddOpponent(opponent string) {
	this.opponents[opponent] = true
	if this.consentors[opponent] {
		delete(this.consentors, opponent)
	}
}

func (this *notification) ResetDirs(s []string) {
	this.dirs = s
}

func (this *notification) WriteComment(gc *ghclient, org, repo string, prNumber int, ok bool) error {
	r := consentientDesc
	if !ok {
		r = opposedDesc
	}
	s := strings.Join(this.dirs, dirSepa)
	if s != "" {
		s = fmt.Sprintf("%s%s", dirSepa, s)
	}
	comment := fmt.Sprintf(
		notificationStr, r,
		strings.Join(mapKeys(this.consentors), separator),
		strings.Join(mapKeys(this.opponents), separator),
		s,
		this.headSHA)

	if this.commentID == 0 {
		return gc.CreateComment(org, repo, prNumber, comment)
	}
	return gc.UpdatePRComment(org, repo, this.commentID, comment)
}

func LoadLGTMnotification(gc *ghclient, org, repo string, prNumber int, sha string) (*notification, bool, error) {
	botname, err := gc.BotName()
	if err != nil {
		return nil, false, err
	}

	comments, err := gc.ListIssueComments(org, repo, prNumber)
	if err != nil {
		return nil, false, err
	}

	split := func(s, sep string) []string {
		if s != "" {
			return strings.Split(s, sep)
		}
		return nil
	}

	n := &notification{headSHA: sha}

	for _, comment := range comments {
		if comment.User.Login != botname {
			continue
		}

		m := notificationStrRe.FindStringSubmatch(comment.Body)
		if m != nil {
			n.commentID = comment.ID

			if m[5] == sha {
				n.consentors = sliceToMap(split(m[2], separator))
				n.opponents = sliceToMap(split(m[3], separator))
				n.dirs = split(m[4], dirSepa)

				return n, false, nil
			}
			break
		}
	}

	filenames, err := originl.GetChangedFiles(gc, org, repo, prNumber)
	if err != nil {
		return nil, false, err
	}

	n.ResetDirs(genDirs(filenames))
	n.ResetConsentor()
	n.ResetOpponents()
	return n, true, nil
}

func genDirs(filenames []string) []string {
	m := map[string]bool{}
	for _, n := range filenames {
		m[filepath.Dir(n)] = true
	}

	if m["."] {
		m["root directory"] = true
		delete(m, ".")
	}

	return mapKeys(m)
}

func mapKeys(m map[string]bool) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}

func sliceToMap(s []string) map[string]bool {
	m := map[string]bool{}
	for _, v := range s {
		m[v] = true
	}
	return m
}
