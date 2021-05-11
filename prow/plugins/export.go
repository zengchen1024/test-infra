package plugins

import (
	"fmt"
	"regexp"
	"strings"
)

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
	m := re.FindStringSubmatch(url)
	if m != nil {
		return m[1]
	}
	return ""
}

// FormatResponseRaw nicely formats a response for one does not have an issue comment
func FormatResponseRaw1(body, bodyURL, login, reply string) string {
	format := `In response to [this](%s):

%s
`
	// Quote the user's comment by prepending ">" to each line.
	var quoted []string
	for _, l := range strings.Split(body, "\n") {
		quoted = append(quoted, ">"+l)
	}
	return formatResponse(login, reply, fmt.Sprintf(format, bodyURL, strings.Join(quoted, "\n")))
}

func formatResponse(to, message, reason string) string {
	format := `@%s: %s

<details>

%s
</details>`

	return fmt.Sprintf(format, to, message, reason)
}
