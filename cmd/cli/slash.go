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
	return s
}
