package cobol

import (
	"fmt"
	"reflect"
	"slices"
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
		"unsupported clause": "       01  X PIC X SYNC.",
		"unsupported usage":  "       01  X PIC 9 USAGE INDEX.",
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
		"an FD clause not read": head + "       DATA DIVISION.\n       FILE SECTION.\n       FD  IN-FILE BLOCK CONTAINS 0 RECORDS.\n       PROCEDURE DIVISION.\n       MAIN-PARA.\n           STOP RUN.\n",
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
