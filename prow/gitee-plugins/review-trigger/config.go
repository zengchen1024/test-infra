package reviewtrigger

import (
	"fmt"

	"github.com/huaweicloud/golangsdk"
	"k8s.io/apimachinery/pkg/util/sets"
)

type configuration struct {
	Trigger *confTrigger `json:"review_trigger,omitempty"`
}

type confTrigger struct {
	CommandsLink string         `json:"commands_link" required:"true"`
	Trigger      []pluginConfig `json:"trigger,omitempty"`
}

func (c confTrigger) commandsLink(org, repo string) string {
	return fmt.Sprintf("%s%s%%2F%s", c.CommandsLink, org, repo)
}

func (c *configuration) Validate() error {
	if c.Trigger == nil {
		return nil
	}

	if _, err := golangsdk.BuildRequestBody(c, ""); err != nil {
		return err
	}

	t := c.Trigger.Trigger
	for i := range t {
		item := &t[i]

		if _, err := parseJobComment(item.TitleOfCITable); err != nil {
			return fmt.Errorf("the format of `title_of_ci_table` is not correct")
		}

		if item.EnableLabelForCI {
			m := map[string]string{
				"label_for_ci_failed":  item.LabelForCIFailed,
				"label_for_ci_running": item.LabelForCIRunning,
			}

			for k, v := range m {
				if v == "" {
					return fmt.Errorf("`%s` must be set when adding label for ci status is enabled", k)
				}
			}
		}
	}

	return nil
}

func (c *configuration) SetDefault() {
	if c.Trigger == nil {
		return
	}

	t := c.Trigger.Trigger
	for i := range t {
		item := &t[i]
		if item.NumberOfApprovers <= 0 {
			item.NumberOfApprovers = 1
		}

		item.runningStatusOfJob = "running"
		if item.Reviewers.ReviewerCount == 0 {
			item.Reviewers.ReviewerCount = 2
		}
	}
}

func (c *configuration) TriggerFor(org, repo string) *pluginConfig {
	if c.Trigger == nil {
		return nil
	}

	fullName := fmt.Sprintf("%s/%s", org, repo)
	index := -1
	t := c.Trigger.Trigger
	for i := range t {
		item := &(t[i])

		s := sets.NewString(item.Repos...)
		if s.Has(fullName) {
			return item
		}

		if s.Has(org) {
			index = i
		}
	}

	if index >= 0 {
		return &(t[index])
	}

	return nil
}

type reviewerConfig struct {
	// ReviewerCount is the minimum number of reviewers to request
	// reviews from. Defaults to requesting reviews from 2 reviewers
	ReviewerCount int `json:"request_count,omitempty"`
	// ExcludeApprovers controls whether approvers are considered to be
	// reviewers. By default, approvers are considered as reviewers if
	// insufficient reviewers are available. If ExcludeApprovers is true,
	// approvers will never be considered as reviewers.
	ExcludeApprovers bool `json:"exclude_approvers,omitempty"`
}

type pluginConfig struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos" required:"true"`

	// AllowSelfApprove is the tag which indicate if the author
	// can appove his/her own pull-request.
	AllowSelfApprove bool `json:"allow_self_approve"`

	// NumberOfApprovers is the min number of approvers who commented
	// /approve at same time to merge the single module
	NumberOfApprovers int `json:"number_of_approvers"`

	// TitleOfCITable is the title of ci comment for pr. The format of comment
	// must have 2 or more columns, and the second column must be job result.
	//
	//   | job name | result  | detail |
	//   | test     | success | link   |
	//
	// The value of TitleOfCITable for ci comment above is
	// `| job name | result | detail |`
	TitleOfCITable string `json:"title_of_ci_table" required:"true"`

	// NumberOfTestCases is the number of test cases for PR
	NumberOfTestCases int `json:"number_of_test_cases" required:"true"`

	// EnableLabelForCI is the tag which indicates whether enables
	// function to add ci status label for PR. If is true, the labels
	// which stand for running and fail must be set.
	EnableLabelForCI bool `json:"enable_label_for_ci"`

	// LabelForCIPassed is the label name for org/repos indicating
	// the CI test cases have passed
	LabelForCIPassed string `json:"label_for_ci_passed" required:"true"`

	// LabelForCIFailed is the label name for org/repos indicating
	// the CI test cases have failed
	LabelForCIFailed string `json:"label_for_ci_failed"`

	// LabelForCIRunning is the label name for org/repos indicating
	// the CI test cases are running
	LabelForCIRunning string `json:"label_for_ci_running"`

	// SuccessStatusOfJob is the status desc when a single job is successful
	SuccessStatusOfJob string `json:"success_status_of_job" required:"true"`

	// FailureStatusOfJob is the status desc when a single job is failed
	FailureStatusOfJob string `json:"failure_status_of_job" required:"true"`

	runningStatusOfJob string

	Reviewers reviewerConfig `json:"reviewers"`

	// BranchWithoutOwners is a list of branches which have no OWNERS file
	// For these branch, collaborators will be work as the approvers
	BranchWithoutOwners []string `json:"branch_without_owners"`
}

func (p pluginConfig) labelsForCI() []string {
	return []string{
		p.LabelForCIFailed, p.LabelForCIPassed, p.LabelForCIRunning,
	}
}

func (p pluginConfig) statusToLabel(status string) string {
	l := ""
	switch status {
	case p.SuccessStatusOfJob:
		l = p.LabelForCIPassed
	case p.runningStatusOfJob:
		l = p.LabelForCIRunning
	case p.FailureStatusOfJob:
		l = p.LabelForCIFailed
	}
	return l
}

func (p pluginConfig) isBranchWithoutOwners(branch string) bool {
	for _, i := range p.BranchWithoutOwners {
		if i == branch {
			return true
		}
	}
	return false
}
