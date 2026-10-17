package record

import (
	"fmt"
	"strings"
	"testing"
)

func acctMaster(t *testing.T) *Layout {
	t.Helper()
	l, err := FromCopybook("../../pangaea/copybooks/ACCTMAST.cpy", nil, "ACCT-MASTER-RECORD")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// A record from the daily fixture: account 10006, balance -150.00.
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
	l, err := FromCopybook("../../pangaea/copybooks/TRANREC.cpy", nil, "TRAN-RECORD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Text(l.Blank(), "FILLER"); err == nil {
		t.Error("looked up FILLER, which TRAN-RECORD has three of")
	}
}

func TestPackedDecimal(t *testing.T) {
	l, err := FromCopybook("testdata/PACKED.cpy", nil, "PACKED-RECORD")
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
