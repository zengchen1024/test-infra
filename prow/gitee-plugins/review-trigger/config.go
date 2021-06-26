package reviewtrigger

import (
	"fmt"
	"regexp"

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
		if err := (&t[i]).validate(); err != nil {
			return err
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

		if item.TotalNumberOfApprovers <= 0 {
			item.TotalNumberOfApprovers = 2
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

		if sets.NewString(item.ExcludedRepos...).Has(fullName) {
			continue
		}

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

	// ExcludedRepos has the form of org/repo.
	ExcludedRepos []string `json:"excluded_repos,omitempty"`

	// AllowSelfApprove is the tag which indicate if the author
	// can appove his/her own pull-request.
	AllowSelfApprove bool `json:"allow_self_approve"`

	// NumberOfApprovers is the min number of approvers who commented
	// /approve at same time to merge the single module
	NumberOfApprovers int `json:"number_of_approvers"`

	// TotalNumberOfApprovers is the min number of approvers who commented
	// /approve at same time to merge the PR
	TotalNumberOfApprovers int `json:"total_number_of_approvers"`

	// MiddleLevel is one of levels which stand for the strictness of review level.
	// The corresponding algorithm will be used to infer label of PR and find the
	// candidate of approvers. The level can be `simple`, `middle` and `strict`.
	MiddleLevel bool `json:"middle_level"`

	// NoCI is the tag which indicates the repo is not set CI.
	// It can't be set with EnableLabelForCI at same time
	NoCI bool `json:"no_ci"`

	// TitleOfCITable is the title of ci comment for pr. The format of comment
	// must have 2 or more columns, and the second column must be job result.
	//
	//   | job name | result  | detail |
	//   | test     | success | link   |
	//
	// The value of TitleOfCITable for ci comment above is
	// `| job name | result | detail |`
	TitleOfCITable string `json:"title_of_ci_table"`

	// NumberOfTestCases is the number of test cases for PR
	NumberOfTestCases int `json:"number_of_test_cases"`

	// EnableLabelForCI is the tag which indicates whether enables
	// function to add ci status label for PR. If is true, the labels
	// which stand for running and fail must be set.
	// It can't be set with NoCI at same time
	EnableLabelForCI bool `json:"enable_label_for_ci"`

	// LabelForCIPassed is the label name for org/repos indicating
	// the CI test cases have passed
	LabelForCIPassed string `json:"label_for_ci_passed"`

	// LabelForCIFailed is the label name for org/repos indicating
	// the CI test cases have failed
	LabelForCIFailed string `json:"label_for_ci_failed"`

	// LabelForCIRunning is the label name for org/repos indicating
	// the CI test cases are running
	LabelForCIRunning string `json:"label_for_ci_running"`

	// SuccessStatusOfJob is the status desc when a single job is successful
	SuccessStatusOfJob string `json:"success_status_of_job"`

	// FailureStatusOfJob is the status desc when a single job is failed
	FailureStatusOfJob string `json:"failure_status_of_job"`

	runningStatusOfJob string

	Reviewers reviewerConfig `json:"reviewers"`

	// BranchWithoutOwners is a list of branches which have no OWNERS file
	// For these branch, collaborators will be work as the approvers
	// It can't be set with BranchWithOwners at same time
	BranchWithoutOwners   string `json:"branch_without_owners"`
	reBranchWithoutOwners *regexp.Regexp

	// BranchWithOwners is a list of branches which have OWNERS file
	// It can't be set with BranchWithoutOwners at same time
	BranchWithOwners []string `json:"branch_with_owners"`
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
	if len(p.BranchWithOwners) > 0 {
		return !sets.NewString(p.BranchWithOwners...).Has(branch)
	}

	return p.reBranchWithoutOwners != nil && p.reBranchWithoutOwners.MatchString(branch)
}

func (p *pluginConfig) validate() error {
	formatError := func(err error) error {
		return fmt.Errorf("Error for config of repo:%s, %v", p.Repos[0], err)
	}

	checkString := func(items map[string]bool, msg string) error {
		for k, v := range items {
			if !v {
				return fmt.Errorf("`%s` %s", k, msg)
			}
		}
		return nil
	}

	validStr := func(s string) bool {
		return s != ""
	}

	if !p.NoCI {
		err := checkString(
			map[string]bool{
				"title_of_ci_table":     validStr(p.TitleOfCITable),
				"label_for_ci_passed":   validStr(p.LabelForCIPassed),
				"success_status_of_job": validStr(p.SuccessStatusOfJob),
				"failure_status_of_job": validStr(p.FailureStatusOfJob),
				"number_of_test_cases":  p.NumberOfTestCases > 0,
			},
			"must be set when CI is set for repo",
		)
		if err != nil {
			return formatError(err)
		}

		if _, err := parseJobComment(p.TitleOfCITable); err != nil {
			return formatError(fmt.Errorf("the format of `title_of_ci_table` is not correct"))
		}
	}

	if p.EnableLabelForCI {
		if p.NoCI {
			return formatError(fmt.Errorf("both `enable_label_for_ci` and `no_ci` can not be set at same time"))
		}

		err := checkString(
			map[string]bool{
				"label_for_ci_failed":  validStr(p.LabelForCIFailed),
				"label_for_ci_running": validStr(p.LabelForCIRunning),
			},
			"must be set when adding label for ci status is enabled",
		)
		if err != nil {
			return formatError(err)
		}
	}

	if len(p.BranchWithOwners) > 0 && p.BranchWithoutOwners != "" {
		return formatError(fmt.Errorf("both `branch_with_owners` and `branch_without_owners` can not be set at same time"))
	}

	if p.BranchWithoutOwners != "" {
		r, err := regexp.Compile(p.BranchWithoutOwners)
		if err != nil {
			return formatError(fmt.Errorf("the value of `branch_without_owners` is not a valid regexp, err:%v", err))
		}
		p.reBranchWithoutOwners = r
	}

	return nil
}
