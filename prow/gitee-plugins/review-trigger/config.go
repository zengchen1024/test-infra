package reviewtrigger

import (
	"fmt"

	"github.com/huaweicloud/golangsdk"
	"k8s.io/apimachinery/pkg/util/sets"
)

type configuration struct {
	Trigger []pluginConfig `json:"review_trigger,omitempty"`
}

func (c *configuration) Validate() error {
	if _, err := golangsdk.BuildRequestBody(c, ""); err != nil {
		return err
	}

	for i := range c.Trigger {
		item := &c.Trigger[i]

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
	for i := range c.Trigger {
		item := &c.Trigger[i]
		if item.NumberOfApprovers <= 0 {
			item.NumberOfApprovers = 1
		}

		item.runningStatusOfJob = "running"
	}
}

func (c *configuration) TriggerFor(org, repo string) *pluginConfig {
	fullName := fmt.Sprintf("%s/%s", org, repo)

	index := -1
	for i := range c.Trigger {
		item := &(c.Trigger[i])

		s := sets.NewString(item.Repos...)
		if s.Has(fullName) {
			return item
		}

		if s.Has(org) {
			index = i
		}
	}

	if index >= 0 {
		return &(c.Trigger[index])
	}

	return nil
}

type pluginConfig struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos" required:"true"`

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
}

func (p pluginConfig) labelsForCI() []string {
	return []string{
		p.LabelForCIFailed, p.LabelForCIPassed, p.LabelForCIRunning,
	}
}
