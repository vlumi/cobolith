package layout

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/vlumi/cobolith/cobol"
)

// A report's lines are its file's records: one per LINE of each group,
// named after the group, with the columns at their COLUMN positions.
func TestReportLines(t *testing.T) {
	source := `>>SOURCE FORMAT IS FREE
IDENTIFICATION DIVISION.
PROGRAM-ID. RWTEST.
ENVIRONMENT DIVISION.
INPUT-OUTPUT SECTION.
FILE-CONTROL.
    SELECT RPT ASSIGN TO "RPT" ORGANIZATION LINE SEQUENTIAL.
DATA DIVISION.
FILE SECTION.
FD  RPT REPORT IS PREMIUMS.
WORKING-STORAGE SECTION.
01  WS-PRODUCT PIC X(2).
01  WS-AMOUNT  PIC 9(7).
REPORT SECTION.
RD  PREMIUMS CONTROLS ARE FINAL WS-PRODUCT PAGE LIMIT 60 LINES HEADING 1 FIRST DETAIL 4.
01  TYPE IS PAGE HEADING.
    05  LINE 1.
        10  COLUMN 1  PIC X(12) VALUE "PREMIUMS DUE".
        10  COLUMN 70 PIC ZZ9   SOURCE PAGE-COUNTER.
    05  LINE 3.
        10  COLUMN 1  PIC X(7)  VALUE "PRODUCT".
01  PREMIUM-DETAIL TYPE IS DETAIL LINE PLUS 1.
    05  COLUMN 1  PIC X(2)  SOURCE WS-PRODUCT.
    05  COLUMN 5  PIC Z,ZZZ,ZZ9 SOURCE WS-AMOUNT.
    05  PIC X(3) VALUE "JPY".
01  TYPE IS CONTROL FOOTING FINAL.
    05  LINE PLUS 2.
        10  COLUMN 5  PIC ZZZ,ZZZ,ZZ9 SUM WS-AMOUNT.
PROCEDURE DIVISION.
MAIN.
    STOP RUN.
`
	tokens, err := cobol.Lex("RWTEST.cbl", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := cobol.Parse("RWTEST.cbl", tokens)
	if err != nil {
		t.Fatal(err)
	}
	laidOut, err := Program(p, cobol.IBM)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range laidOut.Files[0].Records {
		got = append(got, fmt.Sprintf("%s %d bytes", r.Name, r.Length))
		for _, c := range r.Children {
			got = append(got, fmt.Sprintf("  %s %d-%d %s %s source %q sum %v", c.Name, c.Start, c.Start+c.Length-1, c.Picture, c.Category, c.Source, c.Sum))
		}
	}
	want := []string{
		"PAGE-HEADING-LINE-1 72 bytes",
		`  PAGE-HEADING-LINE-1-COL-1 1-12 X(12) alphanumeric source "" sum []`,
		`  PAGE-HEADING-LINE-1-COL-70 70-72 ZZ9 numeric-edited source "PAGE-COUNTER" sum []`,
		"PAGE-HEADING-LINE-3 7 bytes",
		`  PAGE-HEADING-LINE-3-COL-1 1-7 X(7) alphanumeric source "" sum []`,
		"PREMIUM-DETAIL 16 bytes",
		`  PREMIUM-DETAIL-COL-1 1-2 X(2) alphanumeric source "WS-PRODUCT" sum []`,
		`  PREMIUM-DETAIL-COL-5 5-13 Z,ZZZ,ZZ9 numeric-edited source "WS-AMOUNT" sum []`,
		`  PREMIUM-DETAIL-COL-14 14-16 X(3) alphanumeric source "" sum []`,
		"CONTROL-FOOTING-FINAL 15 bytes",
		`  CONTROL-FOOTING-FINAL-COL-5 5-15 ZZZ,ZZZ,ZZ9 numeric-edited source "" sum [WS-AMOUNT]`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("report lines\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
