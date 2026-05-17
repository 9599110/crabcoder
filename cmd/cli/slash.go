package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// interactiveSlash reads a slash command interactively with arrow-key selection.
// prefix is the initial filter text (already typed after /).
// Returns the selected command string, or "" if cancelled.
func interactiveSlash(prefix string) string {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return ""
	}
	defer term.Restore(fd, oldState)

	var input strings.Builder
	input.WriteString(prefix)
	selected := 0
	ch := make(chan readKey, 16)
	done := make(chan struct{})
	defer close(done)

	go readKeys(fd, ch, done)

	// Initial display
	matches := filterCommands(prefix)
	redraw(prefix, matches, selected)

	for k := range ch {
		switch k.key {
		case keyEscape:
			clearLines(len(matches) + 3)
			term.Restore(fd, oldState)
			fmt.Print("\r\n")
			return ""

		case keyEnter:
			clearLines(len(matches) + 3)
			term.Restore(fd, oldState)
			if selected >= 0 && selected < len(matches) {
				cmd := matches[selected].cmd
				fmt.Printf("\r\n> %s\n", cmd)
				return cmd
			}
			fmt.Print("\r\n")
			return ""

		case keyBackspace:
			if input.Len() > 0 {
				s := input.String()
				s = s[:len(s)-1]
				input.Reset()
				input.WriteString(s)
				selected = 0
			}

		case keyUp:
			if selected > 0 {
				selected--
			}

		case keyDown:
			if selected < len(matches)-1 {
				selected++
			}

		case keyRune:
			input.WriteRune(k.rune)
			selected = 0
		}

		matches = filterCommands(input.String())
		if len(matches) == 0 {
			continue
		}
		if selected >= len(matches) {
			selected = len(matches) - 1
		}
		redraw(input.String(), matches, selected)
	}

	return ""
}

type cmdEntry struct {
	cmd  string
	desc string
	tag  string
}

func filterCommands(prefix string) []cmdEntry {
	var matches []cmdEntry
	for _, c := range slashCommands {
		if strings.HasPrefix(c.cmd, "/"+prefix) {
			matches = append(matches, cmdEntry{c.cmd, c.desc, c.tag})
		}
	}
	return matches
}

func redraw(prefix string, matches []cmdEntry, selected int) {
	// Clear previous output: move cursor to start of our output area
	// We draw: header line + separator + one line per match + blank line
	clearLines(len(matches) + 3)

	fmt.Printf("\r\n❯ /%s", prefix)
	fmt.Printf("\r\n%s", strings.Repeat("─", 80))
	for i, m := range matches {
		indicator := " "
		if i == selected {
			indicator = ">"
		}
		tag := ""
		if m.tag != "" {
			tag = " (" + m.tag + ")"
		}
		if i == selected {
			fmt.Printf("\r\n%s \033[7m %-28s \033[0m %s%s", indicator, m.cmd, m.desc, tag)
		} else {
			fmt.Printf("\r\n%s  %-28s  %s%s", indicator, m.cmd, m.desc, tag)
		}
	}
	fmt.Printf("\r\n")
}

func clearLines(n int) {
	// Move up n lines and clear each
	for i := 0; i < n; i++ {
		fmt.Print("\033[F") // move up
	}
	fmt.Print("\033[J") // clear to end of screen
}

// Key types
const (
	keyRune     = iota
	keyEnter    = iota
	keyEscape   = iota
	keyBackspace = iota
	keyUp       = iota
	keyDown     = iota
)

type readKey struct {
	key  int
	rune rune
}

func readKeys(fd int, ch chan<- readKey, done <-chan struct{}) {
	buf := make([]byte, 8)
	for {
		select {
		case <-done:
			return
		default:
		}
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			close(ch)
			return
		}
		switch {
		case buf[0] == 3 || buf[0] == 4: // Ctrl-C, Ctrl-D
			ch <- readKey{key: keyEscape}
			close(ch)
			return
		case buf[0] == 13 || buf[0] == 10: // Enter
			ch <- readKey{key: keyEnter}
		case buf[0] == 27 && n >= 3 && buf[1] == '[':
			// ESC [ sequence
			switch buf[2] {
			case 'A':
				ch <- readKey{key: keyUp}
			case 'B':
				ch <- readKey{key: keyDown}
			}
		case buf[0] == 27:
			ch <- readKey{key: keyEscape}
		case buf[0] == 127 || buf[0] == 8: // Backspace
			ch <- readKey{key: keyBackspace}
		default:
			ch <- readKey{key: keyRune, rune: rune(buf[0])}
		}
	}
}


