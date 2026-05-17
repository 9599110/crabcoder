package main

import (
	"strings"

	"github.com/c-bata/go-prompt"
)

// slashCompleter returns matching slash command suggestions when input starts with /.
func slashCompleter(d prompt.Document) []prompt.Suggest {
	text := d.TextBeforeCursor()
	if !strings.HasPrefix(text, "/") {
		return nil
	}
	var s []prompt.Suggest
	for _, c := range slashCommands {
		if strings.HasPrefix(c.cmd, text) {
			s = append(s, prompt.Suggest{
				Text:        c.cmd,
				Description: c.desc,
			})
		}
	}

	// Model name completions after /model
	if after, ok := strings.CutPrefix(text, "/model "); ok {
		for _, m := range knownModels {
			if strings.HasPrefix(m, after) {
				s = append(s, prompt.Suggest{
					Text:        m,
					Description: "switch to " + m,
				})
			}
		}
	}
	return s
}

var knownModels = []string{
	"claude-sonnet-4-6",
	"claude-opus-4-7",
	"claude-haiku-4-5",
	"deepseek-chat",
	"deepseek-reasoner",
	"gpt-4o",
	"gpt-4o-mini",
}
