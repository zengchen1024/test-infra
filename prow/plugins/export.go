package plugins

import "fmt"

func ResetPluginHelp(ph map[string]HelpProvider) {
	pluginHelp = ph
}

func GetBotCommandLink(url string) string {
	platform := parsePlatform(url)

	p := ""
	switch platform {
	case "gitee":
		p = "gitee-deck/"
	}

	return fmt.Sprintf("https://prow.osinfra.cn/%scommand-help", p)
}


func GetBotDesc(url string) string {
	return fmt.Sprintf(
		"%s I understand the commands that are listed [here](%s).",
		AboutThisBotWithoutCommands,
		GetBotCommandLink(url),
	)
}

func parsePlatform(url string) string {
        re := regexp.MustCompile(".*/(.*).com/")
        m := re.FindStringSubmatch(s)
	if m != nil {
		return m[1]
	}
	return ""
}
