package label

import (
	"fmt"
	"strings"
	"time"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/gitee"
)

func (l *label) handleValidatingLabel(e *sdk.PullRequestEvent, log *logrus.Entry) error {
	org, repo := gitee.GetOwnerAndRepoByPREvent(e)
	cfg, err := l.orgRepoCfg(org, repo)
	if err != nil {
		return err
	}

	v := cfg.LabelsToValidate
	if len(v) == 0 {
		return nil
	}

	m := gitee.GetLabelFromEvent(e.PullRequest.Labels)
	toValidates := map[string]*configOfValidatingLabel{}
	for i := range v {
		item := &v[i]
		if m[item.Label] {
			toValidates[item.Label] = item
		}
	}
	if len(toValidates) == 0 {
		return nil
	}

	prNumber := int(e.PullRequest.Number)

	ops, err := l.ghc.ListPROperationLogs(org, repo, prNumber)
	if err != nil {
		return err
	}

	errs := make([]string, 0, len(toValidates))
	for k, item := range toValidates {
		if t := getTimeOfAddingLabel(ops, k, log); t != nil && item.isExpiry(*t) {
			if err = l.ghc.RemovePRLabel(org, repo, prNumber, k); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, ". "))
}

func getTimeOfAddingLabel(ops []sdk.OperateLog, k string, log *logrus.Entry) *time.Time {
	var t *time.Time
	for i := range ops {
		op := &ops[i]

		if !strings.Contains(op.Content, k) {
			continue
		}

		ut, err := time.Parse(time.RFC3339, op.CreatedAt)
		if err != nil {
			log.Warnf("parse time:%s failed", op.CreatedAt)
			continue
		}

		if t == nil || ut.After(*t) {
			t = &ut
		}
	}
	return t
}
