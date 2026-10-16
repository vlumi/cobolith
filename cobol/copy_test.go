package cobol

import (
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

func TestReadErrors(t *testing.T) {
	for _, file := range []string{"CYCLE.cbl", "MISSING.cbl", "REPLACING.cbl"} {
		t.Run(file, func(t *testing.T) {
			if _, err := Read("testdata/copy/"+file, []string{"testdata/copy"}); err == nil {
				t.Error("read without error")
			}
		})
	}
}
