package tide

import (
	"k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/job-reporter/gitee"
)

func ConvertToSearchPR(r interface{}, prs []PullRequest) {
	nodes := make([]PRNode, 0, len(prs))
	for _, item := range prs {
		nodes = append(nodes, PRNode{PullRequest: item})
	}

	r1 := r.(*searchQuery)
	r1.Search.Nodes = nodes
}

func filterProwJob(pjs []v1.ProwJob) []v1.ProwJob {
	r := make([]v1.ProwJob, 0, len(pjs))

	for i := range pjs {
		pj := pjs[i]

		if gitee.IsGiteeJob(&pj) {
			r = append(r, pj)
		}
	}
	return r
}
