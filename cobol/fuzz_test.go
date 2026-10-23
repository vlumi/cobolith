package cobol

import (
	"os"
	"path/filepath"
	"testing"
)

// seedSources adds the test programs and copybooks to a fuzzer's corpus.
func seedSources(f *testing.F) {
	f.Helper()
	for _, pattern := range []string{"testdata/program/*", "testdata/copy/*"} {
		paths, _ := filepath.Glob(pattern)
		for _, path := range paths {
			if source, err := os.ReadFile(path); err == nil {
				f.Add(source, false)
			}
		}
	}
	f.Add([]byte(">>SOURCE FORMAT FREE\n DISPLAY 'an unclosed literal on a line of well over sixty-five bytes\n"), false)
	f.Add([]byte("       01  A PIC X(3) VALUE 'ab\n      -    'c'.\n"), true)
}

// FuzzRead lexes and parses whatever it is given, in ASCII or Shift_JIS,
// and reads its data entries: a malformed source is an error, never a
// panic.
func FuzzRead(f *testing.F) {
	seedSources(f)
	f.Fuzz(func(t *testing.T, source []byte, japanese bool) {
		encoding := ""
		if japanese {
			encoding = ShiftJIS
		}
		tokens, err := LexIn("fuzz.cbl", source, encoding)
		if err != nil {
			return
		}
		_, _ = ParseAll("fuzz.cbl", tokens)
		_, _ = ParseDataEntry(tokens)
	})
}
