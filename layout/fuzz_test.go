package layout_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vlumi/cobolith/cobol"
	"github.com/vlumi/cobolith/layout"
	"github.com/vlumi/cobolith/record"
)

// FuzzLayouts lays out whatever program it is given, in each dialect, and
// reads a record of each layout by every field: a malformed program or
// record is an error, never a panic.
func FuzzLayouts(f *testing.F) {
	paths, _ := filepath.Glob("../cobol/testdata/program/*")
	for _, path := range paths {
		if source, err := os.ReadFile(path); err == nil {
			f.Add(source, []byte("0123456789ABCDEF"), 0)
		}
	}
	f.Add([]byte("       01  R.\n           05  A PIC S9(5)V99 COMP-3.\n           05  B PIC X(3) OCCURS 2.\n           05  C REDEFINES B PIC 9(6).\n"), []byte("x"), 1)
	dialects := cobol.Dialects
	f.Fuzz(func(t *testing.T, source, data []byte, d int) {
		dialect := dialects[(d%len(dialects)+len(dialects))%len(dialects)]
		tokens, err := cobol.LexIn("fuzz.cbl", source, dialect.Encoding)
		if err != nil {
			return
		}
		if programs, err := cobol.ParseAll("fuzz.cbl", tokens); err == nil {
			for _, p := range programs {
				_, _ = layout.Program(p, dialect)
			}
		}
		entry, err := cobol.ParseDataEntry(tokens)
		if err != nil {
			return
		}
		roots, err := layout.Records([]*cobol.DataEntry{entry}, dialect)
		if err != nil {
			return
		}
		for _, root := range roots {
			l := record.New(root, dialect)
			r := l.Blank()
			copy(r, data)
			for _, field := range l.Fields() {
				_, _ = l.Number(r, field.Name)
				_, _ = l.Text(r, field.Name)
				_, _ = l.Float(r, field.Name)
				_ = l.SetNumber(r, field.Name, -12345)
				_ = l.SetText(r, field.Name, "ab")
			}
		}
	})
}
