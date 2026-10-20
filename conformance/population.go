package main

import (
	"bufio"
	"io"
	"strings"
)

// member is a member of the suite's population file: a program, a library
// text that programs copy, or data, as its *HEADER line names it, and its
// lines.
type member struct {
	kind  string // COBOL, CLBRY or DATA*
	name  string
	lines []string
}

// selected are the option letters EXEC85 selects when no control card
// changes them: the optional code a line flags with one of them in column
// 7 is kept, and other optional code is made a comment.
const selected = "AEHLTY"

// readPopulation splits the population file into its members, each line
// of a program or library text with EXEC85's options applied, and data as
// it is. A header is
// *HEADER,kind,name, or *HEADER,COBOL,main,SUBRTN,name for a program a
// main program calls; a member ends at *END-OF.
func readPopulation(r io.Reader) ([]member, error) {
	var members []member
	var current *member
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " ")
		switch {
		case strings.HasPrefix(line, "*HEADER,"):
			fields := strings.Split(line, ",")
			members = append(members, member{kind: fields[1], name: fields[len(fields)-1]})
			current = &members[len(members)-1]
		case strings.HasPrefix(line, "*END-OF,"):
			current = nil
		case current != nil && current.kind == "DATA*":
			current.lines = append(current.lines, line)
		case current != nil:
			current.lines = append(current.lines, withOptions(line))
		}
	}
	return members, scanner.Err()
}

// withOptions applies EXEC85's options to a line: a letter in column 7,
// other than D for debugging, flags optional code, which a selected option
// keeps by blanking the letter, and an option not selected makes a
// comment, the letter moved to column 8.
func withOptions(line string) string {
	if len(line) < 7 || line[6] == 'D' || line[6] < 'A' || line[6] > 'Z' {
		return line
	}
	if strings.IndexByte(selected, line[6]) >= 0 {
		return line[:6] + " " + line[7:]
	}
	rest := ""
	if len(line) > 8 {
		rest = line[8:]
	}
	return line[:6] + "*" + line[6:7] + rest
}
