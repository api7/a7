package selector

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Item struct {
	ID    string
	Label string
}

var ErrSelectionCanceled = fmt.Errorf("selection canceled")

func SelectOne(title string, items []Item) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items available")
	}
	if !isTerminalStdin() {
		return "", fmt.Errorf("interactive selection requires a terminal")
	}

	filtered := make([]Item, 0, len(items))
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		if item.Label == "" {
			item.Label = item.ID
		}
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 {
		return "", fmt.Errorf("no items available")
	}

	fmt.Fprintln(os.Stdout, title)
	for i, item := range filtered {
		fmt.Fprintf(os.Stdout, "  %d) %s\n", i+1, item.Label)
	}
	fmt.Fprint(os.Stdout, "Enter selection number (q to cancel): ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		if strings.TrimSpace(line) == "" {
			return "", ErrSelectionCanceled
		}
	}

	input := strings.TrimSpace(line)
	if input == "" {
		return "", ErrSelectionCanceled
	}
	if strings.EqualFold(input, "q") || strings.EqualFold(input, "quit") || strings.EqualFold(input, "exit") {
		return "", ErrSelectionCanceled
	}

	idx, err := strconv.Atoi(input)
	if err != nil {
		return "", fmt.Errorf("invalid selection: %s", input)
	}
	if idx < 1 || idx > len(filtered) {
		return "", fmt.Errorf("invalid selection: %d", idx)
	}

	selected := filtered[idx-1].ID
	if selected == "" {
		return "", fmt.Errorf("no item selected")
	}

	return selected, nil
}

func isTerminalStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
