package cobol

import "strings"

// CommentText returns the text of comment lines without their decoration:
// the asterisks that close a banner's lines on the right, the lines that are
// only rules of =, - or *, and the indentation all lines share. Indentation
// beyond that is kept, for text aligned under other text. Blank lines
// separate paragraphs; blank lines at either end are dropped.
func CommentText(lines []string) []string {
	var text []string
	for _, line := range lines {
		s := strings.TrimRight(strings.TrimRight(strings.TrimRight(line, " "), "*"), " ")
		if t := strings.TrimSpace(s); t != "" && strings.Trim(t, "=-*") == "" {
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
	return dedent(text)
}

func dedent(lines []string) []string {
	indent := -1
	for _, line := range lines {
		if line != "" {
			n := len(line) - len(strings.TrimLeft(line, " "))
			if indent < 0 || n < indent {
				indent = n
			}
		}
	}
	for i, line := range lines {
		if line != "" {
			lines[i] = line[indent:]
		}
	}
	return lines
}
