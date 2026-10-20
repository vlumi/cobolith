package record

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/vlumi/cobolith/cobol"
)

func acctMaster(t *testing.T) *Layout {
	t.Helper()
	l, err := FromCopybook("testdata/ACCOUNT.cpy", nil, "ACCT-MASTER-RECORD", cobol.IBM)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// An account record: account 10006, balance -150.00.
const eohippus = "000001000600000105EOHIPPUS ELLA       CO2020090920261015000000001500}000030000                      "

func TestRead(t *testing.T) {
	l := acctMaster(t)
	r := []byte(eohippus)
	if name, err := l.Text(r, "AM-SHORT-NAME"); err != nil || name != "EOHIPPUS ELLA" {
		t.Errorf("name %q, %v", name, err)
	}
	if balance, err := l.Number(r, "AM-BALANCE"); err != nil || balance != -15000 {
		t.Errorf("balance %d, %v, want -15000 cents", balance, err)
	}
	if limit, err := l.Number(r, "AM-OVERDRAFT-LIMIT"); err != nil || limit != 30000 {
		t.Errorf("limit %d, %v, want 30000 cents", limit, err)
	}
}

func TestFields(t *testing.T) {
	var names []string
	for _, f := range acctMaster(t).Fields() {
		names = append(names, fmt.Sprintf("%s@%d", f.Name, f.Start))
	}
	want := "AM-ACCT-NO@1 AM-CUST-NO@11 AM-SHORT-NAME@19 AM-ACCT-TYPE@39 AM-STATUS@40 AM-OPEN-DATE@41 " +
		"AM-LAST-ACTIVITY-DATE@49 AM-BALANCE@57 AM-OVERDRAFT-LIMIT@70 FILLER@79"
	if got := strings.Join(names, " "); got != want {
		t.Errorf("fields\n%s\nwant\n%s", got, want)
	}
}

func TestWriteReadsBack(t *testing.T) {
	l := acctMaster(t)
	r := l.Blank()
	for name, value := range map[string]int64{"AM-ACCT-NO": 10006, "AM-CUST-NO": 105, "AM-OPEN-DATE": 20200909,
		"AM-LAST-ACTIVITY-DATE": 20261015, "AM-BALANCE": -15000, "AM-OVERDRAFT-LIMIT": 30000} {
		if err := l.SetNumber(r, name, value); err != nil {
			t.Fatal(err)
		}
	}
	for name, text := range map[string]string{"AM-SHORT-NAME": "EOHIPPUS ELLA", "AM-ACCT-TYPE": "C", "AM-STATUS": "O"} {
		if err := l.SetText(r, name, text); err != nil {
			t.Fatal(err)
		}
	}
	if string(r) != eohippus {
		t.Errorf("wrote\n%q\nwant\n%q", r, eohippus)
	}
	if err := l.SetNumber(r, "AM-BALANCE", 52); err != nil || string(r[56:69]) != "000000000005B" {
		t.Errorf("0.52 written as %q, %v", r[56:69], err)
	}
}

func TestErrors(t *testing.T) {
	l := acctMaster(t)
	r := l.Blank()
	if err := l.SetNumber(r, "AM-OVERDRAFT-LIMIT", -1); err == nil {
		t.Error("wrote a negative number into an unsigned field")
	}
	if err := l.SetNumber(r, "AM-ACCT-NO", 12345678901); err == nil {
		t.Error("wrote 11 digits into 10")
	}
	if err := l.SetText(r, "AM-STATUS", "OPEN"); err == nil {
		t.Error("wrote 4 characters into 1")
	}
	if _, err := l.Number(r, "AM-SHORT-NAME"); err == nil {
		t.Error("read a text field as a number")
	}
	if _, err := l.Number(r[:10], "AM-ACCT-NO"); err == nil {
		t.Error("read a short record")
	}
}

func TestNamesUsedTwiceCannotBeLookedUp(t *testing.T) {
	l, err := FromCopybook("testdata/FILLERS.cpy", nil, "FILLERS-RECORD", cobol.IBM)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Text(l.Blank(), "FILLER"); err == nil {
		t.Error("looked up FILLER, which FILLERS-RECORD has three of")
	}
}

func TestPackedDecimal(t *testing.T) {
	l, err := FromCopybook("testdata/PACKED.cpy", nil, "PACKED-RECORD", cobol.IBM)
	if err != nil {
		t.Fatal(err)
	}
	r := l.Blank()
	if err := l.SetNumber(r, "PK-AMOUNT", -123456789); err != nil {
		t.Fatal(err)
	}
	if err := l.SetNumber(r, "PK-COUNT", 42); err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprintf("% X", r[:8]), "12 34 56 78 9D 00 04 2F"; got != want {
		t.Errorf("packed % X, want %s", r[:8], want)
	}
	if n, err := l.Number(r, "PK-AMOUNT"); err != nil || n != -123456789 {
		t.Errorf("amount %d, %v", n, err)
	}
	if n, err := l.Number(r, "PK-COUNT"); err != nil || n != 42 {
		t.Errorf("count %d, %v", n, err)
	}
	if err := l.SetNumber(r, "PK-COUNT", 12345); err == nil {
		t.Error("five digits fit in a four-digit field")
	}
	r[4] = 0x9A
	if _, err := l.Number(r, "PK-AMOUNT"); err == nil {
		t.Error("read a number without a sign")
	}
	r[0] = 0xA2
	if _, err := l.Number(r, "PK-AMOUNT"); err == nil {
		t.Error("read a digit above nine")
	}
}

func TestSigns(t *testing.T) {
	l, err := FromCopybook("testdata/SIGNS.cpy", nil, "SIGNS-RECORD", cobol.MicroFocus)
	if err != nil {
		t.Fatal(err)
	}
	r := l.Blank()
	for _, name := range []string{"SG-LEADING-SEPARATE", "SG-TRAILING-SEPARATE", "SG-LEADING", "SG-TRAILING"} {
		if err := l.SetNumber(r, name, -125); err != nil {
			t.Fatal(err)
		}
		if n, err := l.Number(r, name); err != nil || n != -125 {
			t.Errorf("%s: %d, %v, want -125", name, n, err)
		}
	}
	if got, want := string(r), "-125125-J2512N"; got != want {
		t.Errorf("record %q, want %q", got, want)
	}
	r[0] = ' '
	if _, err := l.Number(r, "SG-LEADING-SEPARATE"); err == nil {
		t.Error("read a separate sign that is neither + nor -")
	}
}

func binaryRecord(t *testing.T, dialect cobol.Dialect) *Layout {
	t.Helper()
	l, err := FromCopybook("testdata/BINARY.cpy", nil, "BINARY-RECORD", dialect)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// Binary numbers are big-endian, in two's complement when signed; native
// binary is in the dialect's byte order, which Micro Focus COBOL leaves to
// the platform.
func TestBinary(t *testing.T) {
	l := binaryRecord(t, cobol.IBM)
	r := make([]byte, l.Length)
	for name, value := range map[string]int64{"BN-SHORT": -2, "BN-COUNT": 123456789, "BN-NATIVE": -1} {
		if err := l.SetNumber(r, name, value); err != nil {
			t.Fatal(err)
		}
		if n, err := l.Number(r, name); err != nil || n != value {
			t.Errorf("%s: %d, %v, want %d", name, n, err, value)
		}
	}
	if got, want := fmt.Sprintf("% X", r[:10]), "FF FE 07 5B CD 15 FF FF FF FF"; got != want {
		t.Errorf("binary % X, want %s", r[:10], want)
	}
	for name, value := range map[string]int64{"BN-COUNT": -1, "BN-SHORT": 10000} {
		if err := l.SetNumber(r, name, value); err == nil {
			t.Errorf("%s: wrote %d", name, value)
		}
	}
	mf := binaryRecord(t, cobol.MicroFocus)
	r = make([]byte, mf.Length)
	if err := mf.SetNumber(r, "BN-SHORT", -2); err != nil {
		t.Error(err)
	}
	if _, err := mf.Number(r, "BN-NATIVE"); err == nil {
		t.Error("read COMP-5 in Micro Focus COBOL, whose byte order is the platform's")
	}
}

// IBM COBOL's floating point is hexadecimal: 1.0 is 41 10 00 00, -118.625
// C2 76 A0 00; Micro Focus COBOL's is the platform's.
func TestHexFloat(t *testing.T) {
	l := binaryRecord(t, cobol.IBM)
	r := make([]byte, l.Length)
	at := map[string]int{"BN-SINGLE": 10, "BN-DOUBLE": 14}
	for _, c := range []struct {
		name  string
		value float64
		want  string
	}{
		{"BN-SINGLE", 1, "41 10 00 00"},
		{"BN-SINGLE", -118.625, "C2 76 A0 00"},
		{"BN-SINGLE", 0.1, "40 19 99 9A"},
		{"BN-SINGLE", 0, "00 00 00 00"},
		{"BN-DOUBLE", 1, "41 10 00 00 00 00 00 00"},
	} {
		if err := l.SetFloat(r, c.name, c.value); err != nil {
			t.Fatal(err)
		}
		size := 4
		if c.name == "BN-DOUBLE" {
			size = 8
		}
		if got := fmt.Sprintf("% X", r[at[c.name]:at[c.name]+size]); got != c.want {
			t.Errorf("%s %g: % X, want %s", c.name, c.value, got, c.want)
		}
		if f, err := l.Float(r, c.name); err != nil || math.Abs(f-c.value) > 1e-6*math.Abs(c.value) {
			t.Errorf("%s: read %g, %v, want %g", c.name, f, err, c.value)
		}
	}
	if _, err := l.Number(r, "BN-SINGLE"); err == nil {
		t.Error("read a floating-point field as a number")
	}
	mf := binaryRecord(t, cobol.MicroFocus)
	if _, err := mf.Float(make([]byte, mf.Length), "BN-SINGLE"); err == nil {
		t.Error("read COMP-1 in Micro Focus COBOL, whose byte order is the platform's")
	}
}

// Fujitsu NetCOBOL's native binary is little-endian.
func TestFujitsuNativeBinary(t *testing.T) {
	l, err := FromCopybook("testdata/BINARY.cpy", nil, "BINARY-RECORD", cobol.Fujitsu)
	if err != nil {
		t.Fatal(err)
	}
	r := make([]byte, l.Length)
	if err := l.SetNumber(r, "BN-NATIVE", 258); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("% X", r[6:10]); got != "02 01 00 00" {
		t.Errorf("COMP-5 258 is % X, want 02 01 00 00", r[6:10])
	}
	if n, err := l.Number(r, "BN-NATIVE"); err != nil || n != 258 {
		t.Errorf("read %d, %v", n, err)
	}
}
