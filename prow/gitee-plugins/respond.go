package plugins

import (
	originp "k8s.io/test-infra/prow/plugins"
)

func init() {
	originp.AboutThisBot = originp.GetBotDesc("https://gitee.com/")
}
