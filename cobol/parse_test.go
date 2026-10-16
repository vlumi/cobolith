package cobol

import (
	"reflect"
	"testing"
)

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
		"FD without a name": head + "       DATA DIVISION.\n       FILE SECTION.\n       FD.\n       PROCEDURE DIVISION.\n       MAIN-PARA.\n           STOP RUN.\n",
		"a paragraph twice": head + "       PROCEDURE DIVISION.\n       MAIN-PARA.\n           STOP RUN.\n       MAIN-PARA.\n           STOP RUN.\n",
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
