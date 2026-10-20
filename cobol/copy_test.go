package cobol

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReadExpandsCopybooks(t *testing.T) {
	tokens, err := Read("testdata/copy/MAIN.cbl", []string{"testdata/copy"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"01", "OUTER", ".",
		"05", "FROM-INNER", "PIC", "X(3)", ".",
		"05", "FROM-LEAF", "PIC", "9", ".",
		"01", "AFTER", "PIC", "X", ".",
	}
	if got := texts(tokens); !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if pos := tokens[9].Pos; pos.File != "testdata/copy/LEAF.cpy" || pos.Line != 1 {
		t.Errorf("FROM-LEAF at %v, want its own line in LEAF.cpy", pos)
	}
	if pos := tokens[14].Pos; pos.File != "testdata/copy/MAIN.cbl" || pos.Line != 3 {
		t.Errorf("AFTER at %v, want line 3 of MAIN.cbl", pos)
	}
}

func TestReadReplaces(t *testing.T) {
	tokens, err := Read("testdata/copy/REPLACING.cbl", []string{"testdata/copy"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"01", "WS-RECORD", ".",
		"05", "WS-COUNT", "PIC", "9(3)", "VALUE", "1", ".",
		"05", "WS-NAME", "PIC", "X(10)", "VALUE", "'B'", ".",
	}
	if got := texts(tokens); !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if pos := tokens[8].Pos; pos.File != "testdata/copy/TAGGED.cpy" || pos.Line != 2 {
		t.Errorf("the replacing 1 at %v, want where ZERO was in TAGGED.cpy", pos)
	}
}

func TestReadErrors(t *testing.T) {
	for _, file := range []string{"CYCLE.cbl", "MISSING.cbl", "NOBY.cbl", "UNCLOSED.cbl"} {
		t.Run(file, func(t *testing.T) {
			if _, err := Read("testdata/copy/"+file, []string{"testdata/copy"}); err == nil {
				t.Error("read without error")
			}
		})
	}
}

func TestReadCopiesByLiteralName(t *testing.T) {
	tokens, err := Read("testdata/copy/LITERAL.cbl", []string{"testdata/copy"})
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(tokens); !slices.Contains(got, "FROM-INNER") {
		t.Errorf("got %q, want INNER.cpy copied", got)
	}
}

// REPLACE replaces text in the source after it until REPLACE OFF, and a
// REPLACING operand may carry a subscript; a paragraph name replaced is
// still a header.
func TestReplaceStatement(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "PARAS.cpy"), []byte("       12345.\n           MOVE TAB TO WS-B.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	program := `       IDENTIFICATION DIVISION.
       PROGRAM-ID. REPLACED.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       REPLACE ==PICTURE== BY ==PIC==.
       01  A     PICTURE X.
       REPLACE OFF.
       01  D     PICTURE X(7) VALUE "PICTURE".
       PROCEDURE DIVISION.
       MAIN-PARA.
           STOP RUN.
           COPY PARAS REPLACING 12345 BY PARA-3 TAB BY WS-T (1, 2).
`
	path := filepath.Join(dir, "REPLACED.cbl")
	if err := os.WriteFile(path, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := ReadProgram(path, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if p.WorkingStorage[0].Picture != "X" || p.WorkingStorage[1].Values[0] != `'PICTURE'` {
		t.Errorf("entries %+v %+v", p.WorkingStorage[0], p.WorkingStorage[1])
	}
	if len(p.Paragraphs) != 2 || p.Paragraphs[1].Name != "PARA-3" || p.Paragraphs[1].Statements[0].Text != "MOVE WS-T(1 2) TO WS-B" {
		t.Errorf("paragraphs %+v", p.Paragraphs)
	}
}
