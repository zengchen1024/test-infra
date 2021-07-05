package label

import (
	"fmt"
	"time"

	"github.com/huaweicloud/golangsdk"
	"k8s.io/apimachinery/pkg/util/sets"
)

type labelCfg struct {
	//Repos is either of the form org/repos or just org.
	Repos []string `json:"repos" required:"true"`

	//ClearLabels specifies labels that should be removed when the codes of PR are changed.
	ClearLabels []string `json:"clear_labels,omitempty"`

	//LabelsToValidate specifies config of label that will be validated
	LabelsToValidate []configOfValidatingLabel `json:"labels_to_validate,omitempty"`
}

type configOfValidatingLabel struct {
	// Label is the label name to be validated
	Label string `json:"label" required:"true"`

	// ActiveTime is the time in hours that the label becomes invalid after it from created
	ActiveTime int `json:"expiry_time" required:"true"`
}

func (c configOfValidatingLabel) isExpiry(t time.Time) bool {
	return t.Add(time.Duration(c.ActiveTime) * time.Hour).Before(time.Now())
}

type configuration struct {
	Label []labelCfg `json:"label,omitempty"`
}

func (cfg *configuration) Validate() error {
	_, err := golangsdk.BuildRequestBody(cfg, "")
	return err
}

func (cfg *configuration) SetDefault() {

}

func (cfg *configuration) labelFor(org, repo string) *labelCfg {
	fullName := fmt.Sprintf("%s/%s", org, repo)

	index := -1
	for i := range cfg.Label {
		item := &(cfg.Label[i])

		s := sets.NewString(item.Repos...)
		if s.Has(fullName) {
			return item
		}

		if s.Has(org) {
			index = i
		}
	}

	if index >= 0 {
		return &(cfg.Label[index])
	}

	return nil
}
