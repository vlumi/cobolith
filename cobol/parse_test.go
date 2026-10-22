package cobol

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestLinkage(t *testing.T) {
	p, err := ReadProgram("testdata/program/CALLED.cbl", nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range p.Linkage {
		names = append(names, e.Name)
	}
	if want := []string{"LK-REQUEST", "LK-CODE", "LK-AMOUNT", "LK-RESPONSE"}; !reflect.DeepEqual(names, want) {
		t.Errorf("linkage %q, want %q", names, want)
	}
	if want := []string{"LK-REQUEST", "LK-RESPONSE"}; !reflect.DeepEqual(p.Using, want) {
		t.Errorf("using %q, want %q", p.Using, want)
	}
	if len(p.Paragraphs) != 1 || p.Paragraphs[0].Name != "MAIN-PARA" || len(p.Paragraphs[0].Statements) != 2 {
		t.Errorf("paragraphs %+v, want MAIN-PARA with two statements", p.Paragraphs)
	}
}

func TestReadProgram(t *testing.T) {
	p, err := ReadProgram("testdata/program/SAMPLE.cbl", []string{"testdata/program"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "SAMPLE" {
		t.Errorf("ID %q, want SAMPLE", p.ID)
	}
	if want := []string{" SAMPLE: READS HOLD RECORDS"}; !reflect.DeepEqual(p.Comment, want) {
		t.Errorf("comment %q, want %q", p.Comment, want)
	}
	wantIdentification := []IdentificationEntry{{"AUTHOR", "A. N. AUTHOR."}, {"DATE-WRITTEN", "1987-03-12."}}
	if !reflect.DeepEqual(p.Identification, wantIdentification) {
		t.Errorf("identification %+v, want %+v", p.Identification, wantIdentification)
	}
	if want := []string{" READ ONE RECORD."}; !reflect.DeepEqual(p.Paragraphs[1].Comment, want) {
		t.Errorf("READ-PARA comment %q, want %q", p.Paragraphs[1].Comment, want)
	}
	if want := []string{" A HOLD RECORD"}; !reflect.DeepEqual(p.FileSection[0].Entries[0].Comment, want) {
		t.Errorf("HOLD-RECORD comment %q, want the copybook's comment", p.FileSection[0].Entries[0].Comment)
	}

	wantFiles := []FileControl{{
		Name: "HOLD-FILE", Assign: "HOLDIN", Organization: "LINE SEQUENTIAL", Access: "SEQUENTIAL", Status: "WS-STATUS",
		Pos: Pos{File: "testdata/program/SAMPLE.cbl", Line: 9, Col: 12},
	}, {
		Name: "KEYED-FILE", Assign: "KEYED", Organization: "INDEXED", Access: "DYNAMIC", RecordKey: "KEYED-ID", Status: "WS-STATUS",
		Pos: Pos{File: "testdata/program/SAMPLE.cbl", Line: 12, Col: 12},
	}}
	if !reflect.DeepEqual(p.Files, wantFiles) {
		t.Errorf("files %+v, want %+v", p.Files, wantFiles)
	}

	if len(p.FileSection) != 1 || p.FileSection[0].Name != "HOLD-FILE" {
		t.Fatalf("file section %+v, want the FD of HOLD-FILE", p.FileSection)
	}
	var names []string
	for _, e := range p.FileSection[0].Entries {
		names = append(names, e.Name)
	}
	wantNames := []string{
		"HOLD-RECORD", "HR-TYPE", "HR-OPEN", "HR-SETTLED", "HR-BODY",
		"HR-DETAIL", "HR-ACCT-NO", "HR-AMOUNT", "FILLER",
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Errorf("FD entries %q, want %q", names, wantNames)
	}
	amount := p.FileSection[0].Entries[7]
	if amount.Usage != "COMP-3" || amount.Picture != "S9(7)V99" || amount.Pos.File != "testdata/program/HOLDREC.cpy" {
		t.Errorf("HR-AMOUNT %+v, want COMP-3 S9(7)V99 from the copybook", amount)
	}
	if settled := p.FileSection[0].Entries[3]; !reflect.DeepEqual(settled.Values, []string{"'S'", "'X'"}) {
		t.Errorf("HR-SETTLED values %q", settled.Values)
	}
	if detail := p.FileSection[0].Entries[5]; detail.Redefines != "HR-BODY" {
		t.Errorf("HR-DETAIL redefines %q, want HR-BODY", detail.Redefines)
	}

	item := p.WorkingStorage[3]
	if item.Name != "WS-ITEM" || item.OccursMin != 0 || item.Occurs != 20 || item.DependingOn != "WS-COUNT" ||
		!reflect.DeepEqual(item.IndexedBy, []string{"WS-IX"}) {
		t.Errorf("WS-ITEM %+v, want OCCURS 0 TO 20 DEPENDING ON WS-COUNT INDEXED BY WS-IX", item)
	}
	if passing := p.WorkingStorage[5]; !reflect.DeepEqual(passing.Values, []string{"5 THRU 9"}) {
		t.Errorf("PASSING values %q, want 5 THRU 9", passing.Values)
	}

	if len(p.Paragraphs) != 2 || p.Paragraphs[0].Name != "MAIN-PARA" || p.Paragraphs[1].Name != "READ-PARA" {
		t.Fatalf("paragraphs %+v", p.Paragraphs)
	}
	if got := texts(p.Paragraphs[0].Body); !reflect.DeepEqual(got, []string{"PERFORM", "READ-PARA", "STOP", "RUN", "."}) {
		t.Errorf("MAIN-PARA body %q", got)
	}
}

func TestParseDataEntryErrors(t *testing.T) {
	tests := map[string]string{
		"unsupported clause": "       01  X PIC X OCCURS 3 KEYED.",
		"unsupported usage":  "       01  X USAGE POINTER.",
		"no level":           "       X PIC X.",
		"value without one":  "       01  X PIC X VALUE.",
	}
	for name, line := range tests {
		t.Run(name, func(t *testing.T) {
			tokens, err := Lex("T.cbl", []byte(line))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseDataEntry(tokens[:len(tokens)-1]); err == nil {
				t.Error("parsed without error")
			}
		})
	}
}

func TestReadDataEntries(t *testing.T) {
	entries, err := ReadDataEntries("testdata/program/HOLDREC.cpy", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 9 || entries[0].Name != "HOLD-RECORD" || entries[8].Name != "FILLER" {
		t.Errorf("entries %+v, want the 9 entries of HOLD-RECORD", entries)
	}
}

func TestParseErrors(t *testing.T) {
	head := "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. BROKEN.\n"
	for name, source := range map[string]string{
		"FD without a name":     head + "       DATA DIVISION.\n       FILE SECTION.\n       FD.\n       PROCEDURE DIVISION.\n       MAIN-PARA.\n           STOP RUN.\n",
		"a paragraph twice":     head + "       PROCEDURE DIVISION.\n       MAIN-PARA.\n           STOP RUN.\n       MAIN-PARA.\n           STOP RUN.\n",
		"an FD clause not read": head + "       DATA DIVISION.\n       FILE SECTION.\n       FD  IN-FILE RECORDING MODE F.\n       PROCEDURE DIVISION.\n       MAIN-PARA.\n           STOP RUN.\n",
	} {
		tokens, err := Lex("test", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse("test", tokens); err == nil {
			t.Errorf("%s: parsed without error", name)
		}
	}
}

func TestParseSectionsInFreeFormat(t *testing.T) {
	source := `>>source format is free
identification division.
program-id. sections.
procedure division.
*> the run
main section.
    perform work
    goback.
work section.
first-step.
    move 1
        to ws-a.
second-step.
    exit section.
`
	tokens, err := Lex("test", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("test", tokens)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, para := range p.Paragraphs {
		got = append(got, fmt.Sprintf("%s/%s %d", para.Section, para.Name, len(para.Statements)))
	}
	want := []string{"MAIN/MAIN 2", "WORK/FIRST-STEP 1", "WORK/SECOND-STEP 1"}
	if !slices.Equal(got, want) {
		t.Errorf("paragraphs %q, want %q", got, want)
	}
	if !slices.Equal(p.Paragraphs[0].Comment, []string{"the run"}) {
		t.Errorf("MAIN's comment %q, want the section header's", p.Paragraphs[0].Comment)
	}
	for _, c := range []struct {
		ref         ProcedureRef
		first, last int
		ok          bool
	}{
		{ProcedureRef{Name: "WORK"}, 1, 2, true},
		{ProcedureRef{Name: "FIRST-STEP"}, 1, 1, true},
		{ProcedureRef{Name: "MAIN", Thru: "WORK"}, 0, 2, true},
		{ProcedureRef{Name: "SECOND-STEP", Thru: "FIRST-STEP"}, 2, 1, false},
		{ProcedureRef{Name: "NONE"}, 0, 0, false},
	} {
		if first, last, ok := p.Span(c.ref); ok != c.ok || ok && (first != c.first || last != c.last) {
			t.Errorf("span of %+v: %d to %d, %v; want %d to %d, %v", c.ref, first, last, ok, c.first, c.last, c.ok)
		}
	}
}

func TestParseFileClauses(t *testing.T) {
	source := `>>source format is free
identification division.
program-id. files.
environment division.
input-output section.
file-control.
    select slots assign to "SLOTS"
        organization is relative
        access mode is random
        relative key is ws-slot.
    select lines-in assign to "LINES"
        organization is line sequential.
data division.
file section.
fd  slots record contains 9 characters.
01  slot-record                  pic x(9).
fd  lines-in
    record varying in size from 14 to 61 characters depending on ws-length.
01  short-line                   pic x(14).
01  long-line                    pic x(61).
working-storage section.
01  ws-slot                      pic 9(4) comp-5.
01  ws-length                    pic 9(4) comp-5.
procedure division.
main section.
    goback.
`
	tokens, err := Lex("test", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("test", tokens)
	if err != nil {
		t.Fatal(err)
	}
	if f := p.Files[0]; f.Organization != "RELATIVE" || f.RelativeKey != "WS-SLOT" || f.RecordKey != "" {
		t.Errorf("SLOTS %+v, want a relative file with its record number in WS-SLOT", f)
	}
	var got []string
	for _, fd := range p.FileSection {
		got = append(got, fmt.Sprintf("%s %d %d %s %d", fd.Name, fd.MinLength, fd.MaxLength, fd.LengthIn, len(fd.Entries)))
	}
	if want := []string{"SLOTS 9 9  1", "LINES-IN 14 61 WS-LENGTH 2"}; !slices.Equal(got, want) {
		t.Errorf("file descriptions %q, want %q", got, want)
	}
}

func TestParseDeclaratives(t *testing.T) {
	source := `>>source format is free
identification division.
program-id. errors.
procedure division.
declaratives.
file-errors section.
    use after error procedure on in-file out-file.
note-failure.
    move "y" to ws-failed.
open-errors section.
    use after standard exception procedure input.
    move "o" to ws-failed.
end declaratives.
main section.
    open input in-file
    goback.
`
	tokens, err := Lex("test", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("test", tokens)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, para := range p.Paragraphs {
		got = append(got, fmt.Sprintf("%s/%s %v %d", para.Section, para.Name, para.Declarative, len(para.Statements)))
	}
	want := []string{"FILE-ERRORS/NOTE-FAILURE true 1", "OPEN-ERRORS/OPEN-ERRORS true 1", "MAIN/MAIN false 2"}
	if !slices.Equal(got, want) {
		t.Errorf("paragraphs %q, want %q", got, want)
	}
	got = nil
	for _, d := range p.Declaratives {
		got = append(got, fmt.Sprintf("%s %v %v", d.Section, d.Files, d.Modes))
	}
	want = []string{"FILE-ERRORS [IN-FILE OUT-FILE] []", "OPEN-ERRORS [] [INPUT]"}
	if !slices.Equal(got, want) {
		t.Errorf("declaratives %q, want %q", got, want)
	}
	open := p.Paragraphs[2].Statements[0]
	if got, want := p.ErrorProcedures(open), []string{"FILE-ERRORS", "OPEN-ERRORS"}; !slices.Equal(got, want) {
		t.Errorf("an OPEN of IN-FILE runs %q on failure, want %q", got, want)
	}
	if got := p.ErrorProcedures(p.Paragraphs[2].Statements[1]); got != nil {
		t.Errorf("GOBACK runs %q on failure, want none", got)
	}
}

func TestParseNestedPrograms(t *testing.T) {
	source := `>>source format is free
identification division.
program-id. outer.
procedure division.
main section.
    call "inner" using ws-a
    goback.
identification division.
program-id. inner.
data division.
linkage section.
01  lk-a                         pic 9.
procedure division using lk-a.
main section.
    move 1 to lk-a
    goback.
end program inner.
end program outer.
`
	tokens, err := Lex("test", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("test", tokens)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "OUTER" || len(p.Paragraphs) != 1 || len(p.Paragraphs[0].Statements) != 2 {
		t.Errorf("outer %s with %d paragraphs, want OUTER with MAIN of two statements", p.ID, len(p.Paragraphs))
	}
	if len(p.Contains) != 1 {
		t.Fatalf("%d nested programs, want 1", len(p.Contains))
	}
	if inner := p.Contains[0]; inner.ID != "INNER" || !slices.Equal(inner.Using, []string{"LK-A"}) || len(inner.Linkage) != 1 {
		t.Errorf("nested %s using %q, want INNER using LK-A", inner.ID, inner.Using)
	}
	for name, broken := range map[string]string{
		"no END PROGRAM":        strings.Replace(source, "end program inner.\n", "", 1),
		"after its END PROGRAM": source + "display 'x'.\n",
	} {
		tokens, err := Lex("test", []byte(broken))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse("test", tokens); err == nil {
			t.Errorf("%s: parsed without error", name)
		}
	}
}

// Statements before a procedure division's first paragraph are a paragraph
// with no name.
func TestStatementsBeforeTheFirstParagraph(t *testing.T) {
	source := `>>SOURCE FORMAT IS FREE
IDENTIFICATION DIVISION.
PROGRAM-ID. FIRST.
PROCEDURE DIVISION.
    DISPLAY 'START'
    PERFORM LATER-PARA
    STOP RUN.
LATER-PARA.
    DISPLAY 'LATER'.
`
	tokens, err := Lex("FIRST.cbl", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("FIRST.cbl", tokens)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, para := range p.Paragraphs {
		got = append(got, fmt.Sprintf("%q %d", para.Name, len(para.Statements)))
	}
	if want := []string{`"" 3`, `"LATER-PARA" 1`}; !slices.Equal(got, want) {
		t.Errorf("paragraphs %q, want %q", got, want)
	}
}

// The clauses that change no layout are kept on the entry: a table's keys,
// JUSTIFIED, BLANK WHEN ZERO, EXTERNAL and GLOBAL; and a hex literal is a
// value as written.
func TestDataClauses(t *testing.T) {
	source := `>>SOURCE FORMAT IS FREE
IDENTIFICATION DIVISION.
PROGRAM-ID. CLAUSES.
DATA DIVISION.
WORKING-STORAGE SECTION.
01  R EXTERNAL.
    05  T OCCURS 5 ASCENDING KEY IS K1 K2 DESCENDING K3 INDEXED BY X.
        10  K1 PIC X.
        10  K2 PIC X.
        10  K3 PIC X.
    05  A PIC X(4) JUST RIGHT VALUE X'C1C2C3C4'.
    05  B PIC 9(3) BLANK WHEN ZEROS.
01  S PIC X GLOBAL.
PROCEDURE DIVISION.
MAIN-PARA.
    STOP RUN.
`
	tokens, err := Lex("CLAUSES.cbl", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("CLAUSES.cbl", tokens)
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*DataEntry{}
	for _, e := range p.WorkingStorage {
		entries[e.Name] = e
	}
	if !entries["R"].External || !entries["S"].Global || !entries["A"].Justified || !entries["B"].BlankWhenZero {
		t.Errorf("clauses lost: R %+v, S %+v, A %+v, B %+v", *entries["R"], *entries["S"], *entries["A"], *entries["B"])
	}
	want := []TableKey{{Name: "K1"}, {Name: "K2"}, {Name: "K3", Descending: true}}
	if got := entries["T"].Keys; !slices.Equal(got, want) || !slices.Equal(entries["T"].IndexedBy, []string{"X"}) {
		t.Errorf("keys %v, indexes %v", got, entries["T"].IndexedBy)
	}
	if got := entries["A"].Values; !slices.Equal(got, []string{"X'C1C2C3C4'"}) {
		t.Errorf("hex value %q", got)
	}
}

// The clauses of SELECT and FD entries that change no record are passed
// over; an indexed file's alternate keys, a relative key without KEY, and
// an organization given alone are read.
func TestFileClauses(t *testing.T) {
	source := `       IDENTIFICATION DIVISION.
       PROGRAM-ID. FILES.
       ENVIRONMENT DIVISION.
       INPUT-OUTPUT SECTION.
       FILE-CONTROL.
           SELECT MASTER ASSIGN TO MASTERDD INDEXED
               ACCESS MODE IS DYNAMIC
               RECORD KEY IS MS-KEY
               ALTERNATE RECORD KEY IS MS-NAME WITH DUPLICATES
               ALTERNATE KEY MS-CODE
               RESERVE 2 AREAS
               STATUS IS MS-STATUS OF MS-STATUSES.
           SELECT SLOTS ASSIGN TO SLOTSDD
               ORGANIZATION RELATIVE
               RELATIVE SL-NUMBER
               PADDING CHARACTER IS '*'
               RECORD DELIMITER IS STANDARD-1.
       DATA DIVISION.
       FILE SECTION.
       FD  MASTER
           BLOCK CONTAINS 10 RECORDS
           LABEL RECORDS ARE STANDARD
           VALUE OF FILE-ID IS 'MASTER'
           DATA RECORD IS MS-RECORD
           RECORD CONTAINS 20 CHARACTERS.
       01  MS-RECORD.
           05  MS-KEY  PIC X(4).
           05  MS-NAME PIC X(10).
           05  MS-CODE PIC X(6).
       FD  SLOTS LINAGE IS 60 LINES WITH FOOTING AT 55
           CODE-SET IS ASCII IS EXTERNAL RECORD IS VARYING.
       01  SL-RECORD PIC X(80).
       WORKING-STORAGE SECTION.
       01  MS-STATUSES.
           05  MS-STATUS PIC XX.
       01  SL-NUMBER PIC 9(4).
       PROCEDURE DIVISION.
       MAIN-PARA.
           STOP RUN.
`
	tokens, err := Lex("FILES.cbl", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("FILES.cbl", tokens)
	if err != nil {
		t.Fatal(err)
	}
	master, slots := p.Files[0], p.Files[1]
	wantKeys := []AlternateKey{{Name: "MS-NAME", Duplicates: true}, {Name: "MS-CODE"}}
	if master.Organization != "INDEXED" || master.RecordKey != "MS-KEY" || !slices.Equal(master.AlternateKeys, wantKeys) || master.Status != "MS-STATUS" {
		t.Errorf("master %+v", master)
	}
	if slots.Organization != "RELATIVE" || slots.RelativeKey != "SL-NUMBER" {
		t.Errorf("slots %+v", slots)
	}
	if fd := p.FileSection[0]; fd.MinLength != 20 || fd.MaxLength != 20 {
		t.Errorf("master FD lengths %d to %d, want 20", fd.MinLength, fd.MaxLength)
	}
}

// A section may give its priority number, an entry IS GLOBAL, and a COPY
// REPLACING a qualified name.
func TestSectionsAndQualifiedReplacing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "BOOK.cpy"), []byte("           MOVE A OF G TO B.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	program := `       IDENTIFICATION DIVISION.
       PROGRAM-ID. SECTIONS.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01  G IS GLOBAL.
           05  A PIC X.
           05  C PIC X.
       01  B PIC X.
       PROCEDURE DIVISION.
       MAIN SECTION 50.
       MAIN-PARA.
           COPY BOOK REPLACING A OF G BY C OF G.
           STOP RUN.
`
	path := filepath.Join(dir, "SECTIONS.cbl")
	if err := os.WriteFile(path, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := ReadProgram(path, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	move := p.Paragraphs[0].Statements[0]
	if p.Paragraphs[0].Section != "MAIN" || move.Text != "MOVE C OF G TO B" || !p.WorkingStorage[0].Global {
		t.Errorf("section %q, statement %q, global %v", p.Paragraphs[0].Section, move.Text, p.WorkingStorage[0].Global)
	}
}

// A source file of programs one after the other is ParseAll's, each ended
// by END PROGRAM; Parse reads one.
func TestProgramsOneAfterTheOther(t *testing.T) {
	one := "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. %s.\n       PROCEDURE DIVISION.\n       MAIN-PARA.\n           GOBACK.\n       END PROGRAM %s.\n"
	source := fmt.Sprintf(one, "FIRST", "FIRST") + fmt.Sprintf(one, "SECOND", "SECOND")
	tokens, err := Lex("TWO.cbl", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	programs, err := ParseAll("TWO.cbl", tokens)
	if err != nil || len(programs) != 2 || programs[0].ID != "FIRST" || programs[1].ID != "SECOND" {
		t.Errorf("programs %v, %v", programs, err)
	}
	if _, err := Parse("TWO.cbl", tokens); err == nil {
		t.Error("Parse read two programs as one")
	}
}

// A paragraph name may repeat in another section: a reference by the name
// alone is to its own section's, a reference with OF to the section's
// named, and a name alone that two other sections have is an error.
func TestParagraphsOfSections(t *testing.T) {
	head := "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. SECTIONS.\n       PROCEDURE DIVISION.\n"
	sections := "" +
		"       FIRST-SECTION SECTION.\n       STEP.\n           EXIT.\n       FIRST-MAIN.\n           PERFORM STEP.\n" +
		"       SECOND-SECTION SECTION.\n       STEP.\n           EXIT.\n"
	tokens, err := Lex("SECTIONS.cbl", []byte(head+sections+
		"       THIRD-SECTION SECTION.\n       THIRD-MAIN.\n           PERFORM STEP OF SECOND-SECTION THRU STEP IN SECOND-SECTION\n           STOP RUN.\n"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("SECTIONS.cbl", tokens)
	if err != nil {
		t.Fatal(err)
	}
	spans := map[string]int{}
	for _, para := range p.Paragraphs {
		for _, s := range para.Statements {
			if len(s.Procedures) > 0 {
				first, _, ok := p.Span(s.Procedures[0])
				if !ok {
					t.Fatalf("%s: no span for %+v", para.Name, s.Procedures[0])
				}
				spans[para.Name] = first
			}
		}
	}
	if p.Paragraphs[spans["FIRST-MAIN"]].Section != "FIRST-SECTION" || p.Paragraphs[spans["THIRD-MAIN"]].Section != "SECOND-SECTION" {
		t.Errorf("FIRST-MAIN performs %+v, THIRD-MAIN %+v", p.Paragraphs[spans["FIRST-MAIN"]], p.Paragraphs[spans["THIRD-MAIN"]])
	}
	ambiguous, err := Lex("SECTIONS.cbl", []byte(head+sections+
		"       THIRD-SECTION SECTION.\n       THIRD-MAIN.\n           PERFORM STEP.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("SECTIONS.cbl", ambiguous); err == nil {
		t.Error("parsed PERFORM STEP, which two other sections have")
	}
}

// SPECIAL-NAMES may make the comma the decimal point and name the
// currency sign: the program's pictures and literals are read as written
// and kept in the standard form.
func TestSpecialNames(t *testing.T) {
	source := `       IDENTIFICATION DIVISION.
       PROGRAM-ID. EUROPE.
       ENVIRONMENT DIVISION.
       CONFIGURATION SECTION.
       SPECIAL-NAMES.
           CURRENCY SIGN IS "E"
           DECIMAL-POINT IS COMMA.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01  PRICE   PIC 9(5)V99 VALUE 1234,50.
       01  SHOWN   PIC EE.EEE.EE9,99.
       PROCEDURE DIVISION.
       MAIN-PARA.
           IF PRICE > 1000,5 MOVE PRICE TO SHOWN END-IF
           STOP RUN.
`
	tokens, err := Lex("EUROPE.cbl", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("EUROPE.cbl", tokens)
	if err != nil {
		t.Fatal(err)
	}
	price, shown := p.WorkingStorage[0], p.WorkingStorage[1]
	if !p.DecimalPointComma || p.CurrencySign != "E" || price.Values[0] != "1234.50" || shown.Picture != "$$,$$$,$$9.99" {
		t.Errorf("comma %v, sign %q, value %q, picture %q", p.DecimalPointComma, p.CurrencySign, price.Values[0], shown.Picture)
	}
	if got := p.Paragraphs[0].Statements[0].Condition.String(); got != "PRICE > 1000.5" {
		t.Errorf("condition %q", got)
	}
}

// A debugging declarative is kept apart from those for file errors, and
// names no file.
func TestDebuggingDeclarative(t *testing.T) {
	source := `       IDENTIFICATION DIVISION.
       PROGRAM-ID. DEBUGGED.
       PROCEDURE DIVISION.
       DECLARATIVES.
       WATCH SECTION.
           USE FOR DEBUGGING ON ALL PROCEDURES.
       WATCH-PARA.
           DISPLAY DEBUG-NAME.
       END DECLARATIVES.
       MAIN SECTION.
       MAIN-PARA.
           STOP RUN.
`
	tokens, err := Lex("DEBUGGED.cbl", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("DEBUGGED.cbl", tokens)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Declaratives) != 1 || !p.Declaratives[0].Debugging || len(p.Declaratives[0].Files) != 0 {
		t.Errorf("declaratives %+v", p.Declaratives)
	}
}

// A VALUE may be a literal that reads like a clause word.
func TestValueLikeAClause(t *testing.T) {
	e, err := ParseDataEntry(mustLex(t, `       01  D PICTURE X(7) VALUE "PICTURE".`))
	if err != nil || len(e.Values) != 1 || e.Values[0] != `'PICTURE'` {
		t.Errorf("entry %+v, %v", e, err)
	}
}

func mustLex(t *testing.T, source string) []Token {
	t.Helper()
	tokens, err := Lex("test", []byte(source+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	return tokens[:len(tokens)-1]
}

// A program in Hitachi COBOL2002, in Shift_JIS: Japanese names for the
// record, its fields and a paragraph, national items and literals.
func TestJapaneseProgram(t *testing.T) {
	source := "       IDENTIFICATION DIVISION.\n       PROGRAM-ID. JTEST.\n       DATA DIVISION.\n       WORKING-STORAGE SECTION.\n" +
		"       01  \x83\x8c\x83\x52\x81\x5b\x83\x68.\n" + // レコード
		"           05  \x8e\x81\x96\xbc PIC N(3) VALUE N\"\x93\xfa\x96\x7b\x8c\xea\".\n" + // 氏名, 日本語
		"           05  \x8b\xe6\x95\xaa PIC X(2) VALUE \"ab\".\n" + // 区分
		"       PROCEDURE DIVISION.\n       \x8f\x88\x97\x9d.\n" + // 処理
		"           MOVE N\"\x93\xfa\" TO \x8e\x81\x96\xbc\n           STOP RUN.\n"
	path := filepath.Join(t.TempDir(), "JTEST.cbl")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := ReadProgramIn(path, nil, Hitachi)
	if err != nil {
		t.Fatal(err)
	}
	record := p.WorkingStorage[0]
	if record.Name != "\x83\x8c\x83\x52\x81\x5b\x83\x68" || len(p.WorkingStorage) != 3 || p.WorkingStorage[1].Values[0] != "N'\x93\xfa\x96\x7b\x8c\xea'" {
		t.Errorf("record %q, %d entries, value %q", record.Name, len(p.WorkingStorage), p.WorkingStorage[1].Values)
	}
	if para := p.Paragraphs[0]; para.Name != "\x8f\x88\x97\x9d" || para.Statements[0].Text != "MOVE N'\x93\xfa' TO \x8e\x81\x96\xbc" {
		t.Errorf("paragraph %q, statement %q", para.Name, para.Statements[0].Text)
	}
}

// A report of the Report Writer: the FD names it, the RD describes its
// controls and page, and its groups are entries with a TYPE, lines and
// columns, each printing a SOURCE item or a SUM of items.
func TestParseReport(t *testing.T) {
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
01  WS-DATE    PIC X(7).
REPORT SECTION.
RD  PREMIUMS
    CONTROLS ARE FINAL WS-PRODUCT
    PAGE LIMIT IS 60 LINES HEADING 1 FIRST DETAIL 4 LAST DETAIL 56 FOOTING 58.
01  TYPE IS PAGE HEADING.
    05  LINE 1.
        10  COLUMN 1  PIC X(12) VALUE "PREMIUMS DUE".
        10  COLUMN 20 PIC X(7)  SOURCE WS-DATE.
        10  COLUMN 70 PIC ZZ9   SOURCE PAGE-COUNTER.
01  TYPE IS CONTROL HEADING WS-PRODUCT.
    05  LINE PLUS 1.
        10  COLUMN 1  PIC X(2)  SOURCE WS-PRODUCT.
01  PREMIUM-DETAIL TYPE IS DETAIL.
    05  LINE PLUS 1.
        10  COLUMN 5  PIC Z,ZZZ,ZZ9 SOURCE WS-AMOUNT.
01  TYPE IS CONTROL FOOTING WS-PRODUCT NEXT GROUP PLUS 1.
    05  LINE PLUS 1.
        10  COLUMN 5  PIC ZZ,ZZZ,ZZ9 SUM WS-AMOUNT.
01  TYPE IS CF FINAL.
    05  LINE PLUS 2.
        10  COLUMN 5  PIC ZZZ,ZZZ,ZZ9 SUM WS-AMOUNT.
PROCEDURE DIVISION.
MAIN.
    OPEN OUTPUT RPT
    INITIATE PREMIUMS
    GENERATE PREMIUM-DETAIL
    TERMINATE PREMIUMS
    CLOSE RPT
    STOP RUN.
`
	tokens, err := Lex("RWTEST.cbl", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Parse("RWTEST.cbl", tokens)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.FileSection[0].Reports; !slices.Equal(got, []string{"PREMIUMS"}) || len(p.FileSection[0].Entries) != 0 {
		t.Errorf("FD RPT reports %v with %d entries, want PREMIUMS and no records", got, len(p.FileSection[0].Entries))
	}
	if len(p.ReportSection) != 1 {
		t.Fatalf("%d reports, want 1", len(p.ReportSection))
	}
	rd := p.ReportSection[0]
	if rd.Name != "PREMIUMS" || !slices.Equal(rd.Controls, []string{"FINAL", "WS-PRODUCT"}) ||
		rd.PageLimit != 60 || rd.Heading != 1 || rd.FirstDetail != 4 || rd.LastDetail != 56 || rd.Footing != 58 {
		t.Errorf("RD %+v", rd)
	}
	var got []string
	for _, e := range rd.Entries {
		got = append(got, fmt.Sprintf("%02d %s %s %s %q %d %s %v", e.Level, e.Name, e.Type, e.TypeControl, e.Line, e.Column, e.Source, e.Sum))
	}
	want := []string{
		`01 FILLER PAGE HEADING  "" 0  []`,
		`05 FILLER   "1" 0  []`,
		`10 FILLER   "" 1  []`,
		`10 FILLER   "" 20 WS-DATE []`,
		`10 FILLER   "" 70 PAGE-COUNTER []`,
		`01 FILLER CONTROL HEADING WS-PRODUCT "" 0  []`,
		`05 FILLER   "PLUS 1" 0  []`,
		`10 FILLER   "" 1 WS-PRODUCT []`,
		`01 PREMIUM-DETAIL DETAIL  "" 0  []`,
		`05 FILLER   "PLUS 1" 0  []`,
		`10 FILLER   "" 5 WS-AMOUNT []`,
		`01 FILLER CONTROL FOOTING WS-PRODUCT "" 0  []`,
		`05 FILLER   "PLUS 1" 0  []`,
		`10 FILLER   "" 5  [WS-AMOUNT]`,
		`01 FILLER CONTROL FOOTING FINAL "" 0  []`,
		`05 FILLER   "PLUS 2" 0  []`,
		`10 FILLER   "" 5  [WS-AMOUNT]`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("report groups\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	var verbs []string
	for _, s := range p.Paragraphs[0].Statements {
		verbs = append(verbs, s.Verb)
	}
	if want := []string{"OPEN", "INITIATE", "GENERATE", "TERMINATE", "CLOSE", "STOP"}; !slices.Equal(verbs, want) {
		t.Errorf("statements %v, want %v", verbs, want)
	}
}
