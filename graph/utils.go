package graph

import (
	"regexp"
	"unicode/utf8"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

func stripAnsi(str string) string {
	re := regexp.MustCompile(ansi)
	return re.ReplaceAllString(str, "")
}

// maxRuneCountInStrings returns the rune count of the longest string in the
// slice.
func maxRuneCountInStrings(slice []string) int {
	var max, count int
	for _, s := range slice {
		count = utf8.RuneCountInString(s)
		if count > max {
			max = count
		}
	}
	return max
}
