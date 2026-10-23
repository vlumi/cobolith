package record

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vlumi/cobolith/cobol"
	"golang.org/x/text/encoding/japanese"
)

// TestShiftJISCopybook reads a copybook written in Shift_JIS, as the
// Japanese compilers have them, and finds its fields by their half-width
// katakana names in UTF-8, while the record stays in the file's bytes.
func TestShiftJISCopybook(t *testing.T) {
	source := `       01  PM-ｹｲﾔｸ-ﾚｺｰﾄﾞ.
           05  PM-ｼｮｳｹﾝﾊﾞﾝｺﾞｳ          PIC X(10).
           05  PM-ｼﾒｲ                  PIC X(20).
           05  PM-ﾎｹﾝｷﾝｶﾞｸ             PIC 9(9).
`
	encoded, err := japanese.ShiftJIS.NewEncoder().String(source)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "POLMST.cpy")
	if err := os.WriteFile(path, []byte(encoded), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := FromCopybook(path, nil, "PM-ｹｲﾔｸ-ﾚｺｰﾄﾞ", cobol.Fujitsu)
	if err != nil {
		t.Fatal(err)
	}
	if l.Length != 39 {
		t.Errorf("record of %d bytes, want 39", l.Length)
	}
	r := l.Blank()
	name, _ := japanese.ShiftJIS.NewEncoder().String("山田　太郎")
	if err := l.SetText(r, "PM-ｼﾒｲ", name); err != nil {
		t.Fatal(err)
	}
	if err := l.SetNumber(r, "PM-ﾎｹﾝｷﾝｶﾞｸ", 10000000); err != nil {
		t.Fatal(err)
	}
	if got, _ := l.Text(r, "PM-ｼﾒｲ"); got != name {
		t.Errorf("the name reads back as %q, want the Shift_JIS bytes %q", got, name)
	}
	if n, _ := l.Number(r, "PM-ﾎｹﾝｷﾝｶﾞｸ"); n != 10000000 {
		t.Errorf("the sum reads back as %d", n)
	}
}
