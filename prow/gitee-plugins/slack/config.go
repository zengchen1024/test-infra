package slack

import "fmt"

type configuration struct {
	Slack slackConfig `json:"slack,omitempty"`
}

func (c *configuration) Validate() error {
	if c.Slack.RobotChannel == "" {
		return fmt.Errorf("robot_channel must be set")
	}

	if c.Slack.DevChannel == "" {
		return fmt.Errorf("dev_channel must be set")
	}

	return nil
}

func (c *configuration) SetDefault() {
}

// Sync holds configuration for the sync plugin
type slackConfig struct {
	// RobotChannel is the webhook endpoint of slack robot channel
	RobotChannel string `json:"robot_channel"`

	// DevChannel is the webhook endpoint of slack dev channel
	DevChannel string `json:"dev_channel"`
}
