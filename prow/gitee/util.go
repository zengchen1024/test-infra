package gitee

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
)

func GetOwnerAndRepoByEvent(e interface{}) (string, string) {
	var repository *sdk.ProjectHook

	switch t := e.(type) {
	case *sdk.PullRequestEvent:
		repository = t.Repository
	case *sdk.NoteEvent:
		repository = t.Repository
	case *sdk.IssueEvent:
		repository = t.Repository
	case *sdk.PushEvent:
		repository = t.Repository
	default:
		return "", ""
	}

	return repository.Namespace, repository.Path
}
