package main

import (
	"fmt"
	"strings"
)

func renderPanel(title string, lines []string) {
	width := calculateWidth(lines)
	borderWidth := max(width, len(title)+2)

	fmt.Printf("┌─%s %s┐\n", title, strings.Repeat("─", borderWidth-len(title)-2))
	for _, line := range lines {
		fmt.Printf("  %s  \n", line)
	}
	fmt.Printf("└%s┘\n", strings.Repeat("─", borderWidth))
}

func calculateWidth(lines []string) int {
	maxLen := 0
	for _, line := range lines {
		stripped := stripANSI(line)
		if len(stripped) > maxLen {
			maxLen = len(stripped)
		}
	}
	return maxLen + 4
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
