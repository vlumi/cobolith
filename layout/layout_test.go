package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vlumi/copal/bedrock/cobol"
)

func entries(t *testing.T, source string) []*cobol.DataEntry {
	t.Helper()
	tokens, err := cobol.Lex("T.cpy", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var out []*cobol.DataEntry
	start := 0
	for i, tok := range tokens {
		if tok.Kind == cobol.Period {
			e, err := cobol.ParseDataEntry(tokens[start:i])
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, e)
			start = i + 1
		}
	}
	return out
}

// outline lists every field as "level name start length", indented by depth.
func outline(fields []*Field, depth int, b *strings.Builder) {
	for _, f := range fields {
		fmt.Fprintf(b, "%s%02d %s %d %d\n", strings.Repeat("  ", depth), f.Level, f.Name, f.Start, f.Length)
		outline(f.Children, depth+1, b)
	}
}

func TestRecords(t *testing.T) {
	source := `
       01  HOLD-RECORD.
           05  HR-TYPE                 PIC X.
               88  HR-OPEN             VALUE 'O'.
           05  HR-BODY                 PIC X(19).
           05  HR-DETAIL REDEFINES HR-BODY.
               10  HR-ACCT-NO          PIC 9(10).
               10  HR-AMOUNT           PIC S9(7)V99 COMP-3.
               10  FILLER              PIC X(4).
           05  HR-HISTORY              OCCURS 3 TIMES.
               10  HR-DATE             PIC 9(8).
               10  HR-SUM              PIC S9(5) COMP.
           05  HR-END                  PIC X.
       01  WS-PACKED                   COMP-3.
           05  WS-A                    PIC S9(5).
           05  WS-B                    PIC 9(2).
       77  WS-I                        PIC 99.
`
	records, err := Records(entries(t, source), cobol.IBM)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	outline(records, 0, &b)
	want := `01 HOLD-RECORD 1 57
  05 HR-TYPE 1 1
  05 HR-BODY 2 19
  05 HR-DETAIL 2 19
    10 HR-ACCT-NO 2 10
    10 HR-AMOUNT 12 5
    10 FILLER 17 4
  05 HR-HISTORY 21 12
    10 HR-DATE 21 8
    10 HR-SUM 29 4
  05 HR-END 57 1
01 WS-PACKED 1 5
  05 WS-A 1 3
  05 WS-B 4 2
77 WS-I 1 2
`
	if b.String() != want {
		t.Errorf("got\n%s\nwant\n%s", b.String(), want)
	}
	if c := records[0].Children[0].Conditions; len(c) != 1 || c[0].Name != "HR-OPEN" {
		t.Errorf("HR-TYPE conditions %+v, want HR-OPEN", c)
	}
	amount := records[0].Children[2].Children[1]
	if amount.Digits != 9 || amount.Scale != 2 || !amount.Signed || amount.Category != Numeric {
		t.Errorf("HR-AMOUNT %+v, want 9 signed digits, 2 after the point", amount)
	}
}

func TestRecordsOccursDependingOn(t *testing.T) {
	source := `
       01  WS-LIST.
           05  WS-COUNT                PIC 9(2).
           05  WS-ITEM                 PIC X(10)
                   OCCURS 0 TO 20 TIMES DEPENDING ON WS-COUNT.
`
	records, err := Records(entries(t, source), cobol.IBM)
	if err != nil {
		t.Fatal(err)
	}
	item := records[0].Children[1]
	if records[0].Length != 202 || item.Length != 10 || item.Occurs != 20 || item.DependingOn != "WS-COUNT" {
		t.Errorf("WS-LIST %d bytes, WS-ITEM %+v; want 202 bytes at the maximum of 20", records[0].Length, item)
	}
}

func TestRecordsErrors(t *testing.T) {
	tests := map[string]string{
		"group with picture": "       01  R PIC X.\n           05  A PIC X.\n",
		"no picture":         "       01  R.\n           05  A.\n",
		"unknown redefines":  "       01  R.\n           05  A PIC X.\n           05  B REDEFINES C PIC X.\n",
		"orphan level":       "       05  A PIC X.\n",
		"orphan condition":   "       88  A VALUE 'Y'.\n",
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Records(entries(t, source), cobol.IBM); err == nil {
				t.Error("built without error")
			}
		})
	}
}

func TestMicroFocusClauses(t *testing.T) {
	source := `>>source format is free
78  floor                        value 25.00.
01  account.
    05  balance                  pic s9(7)v99 sign leading separate.
    05  overpunched              pic s9(3) sign is trailing.
    05  slot                     pic 9(4) comp-5.
    05  flag                     pic 9 comp-5.
`
	records, err := Records(entries(t, source), cobol.MicroFocus)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("%d records, want 1: a constant is no record", len(records))
	}
	var got []string
	for _, f := range records[0].Children {
		got = append(got, fmt.Sprintf("%s %d %d %s %v", f.Name, f.Start, f.Length, f.Sign, f.SignSeparate))
	}
	want := "BALANCE 1 10 LEADING true; OVERPUNCHED 11 3 TRAILING false; SLOT 14 2  false; FLAG 16 1  false"
	if strings.Join(got, "; ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, "; "), want)
	}
}
