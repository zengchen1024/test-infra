package trigger

import (
	originp "k8s.io/test-infra/prow/plugins"
)

type configuration struct {
	Triggers []originp.Trigger `json:"triggers,omitempty"`
}

func (c *configuration) Validate() error {
	return nil
}

func (c *configuration) SetDefault() {
	// set JoinOrgURL
}

