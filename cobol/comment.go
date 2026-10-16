package cobol

import "strings"

// CommentText returns the text of comment lines without their decoration:
// the asterisks that close a banner's lines on the right and the lines that
// are only rules of =, - or *. Blank lines separate paragraphs; blank lines
// at either end are dropped.
func CommentText(lines []string) []string {
	var text []string
	for _, line := range lines {
		s := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(line), "*"))
		if s != "" && strings.Trim(s, "=-*") == "" {
			continue
		}
		if s == "" && (len(text) == 0 || text[len(text)-1] == "") {
			continue
		}
		text = append(text, s)
	}
	for len(text) > 0 && text[len(text)-1] == "" {
		text = text[:len(text)-1]
	}
	return text
}
