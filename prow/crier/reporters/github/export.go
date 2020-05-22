package github

import (
	"fmt"

	v1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/github/report"
)

func (c *Client) Report1(pj *v1.ProwJob, gc report.GitHubClient) ([]*v1.ProwJob, error) {
	// The github comment create/update/delete done for presubmits
	// needs pr-level locking to avoid racing when reporting multiple
	// jobs in parallel.
	if pj.Spec.Type == v1.PresubmitJob {
		key, err := lockKeyForPJ(pj)
		if err != nil {
			return nil, fmt.Errorf("failed to get lockkey for job: %v", err)
		}
		lock := c.prLocks.getLock(*key)
		lock.Lock()
		defer lock.Unlock()
	}

	// TODO(krzyzacy): ditch ReportTemplate, and we can drop reference to config.Getter
	return []*v1.ProwJob{pj}, report.Report(gc, c.config().Plank.ReportTemplateForRepo(pj.Spec.Refs), *pj, c.config().GitHubReporter.JobTypesToReport)
}
