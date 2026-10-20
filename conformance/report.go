package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// result is how reading one program went: its error, if it failed.
type result struct {
	program string
	failure string
}

// modules name the suite's modules by the first two letters of their
// programs.
var modules = map[string]string{
	"CM": "Communication",
	"DB": "Debugging",
	"EX": "EXEC85, the suite's own program",
	"IC": "Inter-program communication",
	"IF": "Intrinsic functions",
	"IX": "Indexed I-O",
	"NC": "Nucleus",
	"OB": "Obsolete elements",
	"RL": "Relative I-O",
	"RW": "Report writer",
	"SG": "Segmentation",
	"SM": "Source text manipulation",
	"SQ": "Sequential I-O",
	"ST": "Sort-merge",
}

// report writes the results as Markdown: what was run and how, how many
// programs of each module were read, and the reasons the rest were not,
// most common first, each with the programs it stopped.
func report(results []result) string {
	var b strings.Builder
	read := 0
	for _, r := range results {
		if r.failure == "" {
			read++
		}
	}
	b.WriteString("# Conformance: NIST's COBOL-85 test suite\n\n")
	fmt.Fprintf(&b, "Cobolith reads %d of the %d programs of CCVS85, the COBOL Compiler Validation System for COBOL-85 of the National Institute of Standards and Technology. A program is read when it is lexed, its COPY statements expanded from the suite's library, parsed, and its records laid out in IBM COBOL. Whether it would compile, or run as the suite expects, is not tested.\n\n", read, len(results))
	b.WriteString("The suite was created by NIST employees and is in the public domain; cobolith's tests use it with that acknowledgment, and do not redistribute it. `make conformance` fetches it from the Internet Archive's copy of NIST's address, checks it against the hash GNU Guix pins it by, and writes this report. Its programs are prepared as the suite's own program, EXEC85, prepares them with no control cards: the optional code a letter in column 7 flags is kept for the options it selects by default, A, E, H, L, T and Y, and made a comment for the rest. Its X-cards, the implementor's names it would substitute for XXXXXnnn, are left as written: most stand where any name may.\n\n")
	b.WriteString("| Module | Programs | Read |\n| --- | --: | --: |\n")
	byModule := map[string][2]int{}
	for _, r := range results {
		n := byModule[r.program[:2]]
		n[0]++
		if r.failure == "" {
			n[1]++
		}
		byModule[r.program[:2]] = n
	}
	for _, m := range slices.Sorted(mapKeys(byModule)) {
		fmt.Fprintf(&b, "| %s, %s | %d | %d |\n", m, cmp.Or(modules[m], "?"), byModule[m][0], byModule[m][1])
	}
	b.WriteString("\n## Why the rest are not read\n\nEach program stops at its first error. The reasons, most common first, with the programs each stops:\n\n| Reason | Programs |\n| --- | --- |\n")
	stopped := map[string][]string{}
	for _, r := range results {
		if r.failure != "" {
			reason := reasonOf(r.failure)
			stopped[reason] = append(stopped[reason], r.program)
		}
	}
	reasons := slices.Sorted(mapKeys(stopped))
	slices.SortStableFunc(reasons, func(a, b string) int { return len(stopped[b]) - len(stopped[a]) })
	for _, reason := range reasons {
		fmt.Fprintf(&b, "| %s | %s |\n", escape(reason), strings.Join(stopped[reason], ", "))
	}
	return b.String()
}

// reasonOf is an error without where it happened: the text after the
// last file position, paragraph or item it names.
func reasonOf(failure string) string {
	if i := strings.LastIndex(failure, ": "); i >= 0 {
		return strings.TrimSpace(failure[i+2:])
	}
	return failure
}

// escape keeps text from breaking a Markdown table cell.
func escape(s string) string {
	return "`" + strings.ReplaceAll(s, "|", "\\|") + "`"
}

func mapKeys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}
