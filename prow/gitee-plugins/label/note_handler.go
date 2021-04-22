package label

type noteHandler interface {
	addLabel(label []string) error
	addComment(comment string) error
	removeLabel(label string) error
	getLabels() (map[string]string, error)
}

type issueNoteHandle struct {
	client giteeClient
	org    string
	repo   string
	number string
}

func (inh *issueNoteHandle) addLabel(label []string) error {
	return inh.client.AddMultiIssueLabel(inh.org, inh.repo, inh.number, label)
}

func (inh *issueNoteHandle) addComment(comment string) error {
	return inh.client.CreateIssueComment(inh.org, inh.repo, inh.number, comment)
}

func (inh *issueNoteHandle) removeLabel(label string) error {
	return inh.client.RemoveIssueLabel(inh.org, inh.repo, inh.number, label)
}

func (inh *issueNoteHandle) getLabels() (map[string]string, error) {
	labels, err := inh.client.GetIssueLabels(inh.org, inh.repo, inh.number)
	if err != nil {
		return nil, err
	}
	return labelsTransformMap(labels), nil
}

type prNoteHandle struct {
	client giteeClient
	org    string
	repo   string
	number int
}

func (pnh *prNoteHandle) addLabel(label []string) error {
	return pnh.client.AddMultiPRLabel(pnh.org, pnh.repo, pnh.number, label)
}

func (pnh *prNoteHandle) addComment(comment string) error {
	return pnh.client.CreatePRComment(pnh.org, pnh.repo, pnh.number, comment)
}

func (pnh *prNoteHandle) removeLabel(label string) error {
	return pnh.client.RemovePRLabel(pnh.org, pnh.repo, pnh.number, label)
}

func (pnh *prNoteHandle) getLabels() (map[string]string, error) {
	labels, err := pnh.client.GetPRLabels(pnh.org, pnh.repo, pnh.number)
	if err != nil {
		return nil, err
	}
	return labelsTransformMap(labels), nil
}
