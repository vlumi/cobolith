package layout

import (
	"cmp"
	"fmt"
	"strings"
	"testing"

	"github.com/vlumi/cobolith/cobol"
)

// layoutOf lays out working storage in a dialect, a line per field: its
// level, name, start, size, category and usage, and what else it has.
func layoutOf(t *testing.T, entries string, dialect cobol.Dialect) (string, error) {
	t.Helper()
	source := ">>SOURCE FORMAT IS FREE\nIDENTIFICATION DIVISION.\nPROGRAM-ID. FORMS.\nDATA DIVISION.\nWORKING-STORAGE SECTION.\n" +
		entries + "\nPROCEDURE DIVISION.\nMAIN-PARA.\n    STOP RUN.\n"
	tokens, err := cobol.Lex("FORMS.cbl", []byte(source))
	if err != nil {
		return "", err
	}
	p, err := cobol.Parse("FORMS.cbl", tokens)
	if err != nil {
		return "", err
	}
	records, err := Program(p, dialect)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	var visit func(f *Field)
	visit = func(f *Field) {
		fmt.Fprintf(&b, "%02d %s %d+%d %s %s", f.Level, f.Name, f.Start, f.Size(), f.Category, f.Usage)
		if f.Occurs > 0 {
			fmt.Fprintf(&b, " occurs %d", f.Occurs)
		}
		if f.Redefines != "" {
			fmt.Fprintf(&b, " redefines %s", f.Redefines)
		}
		if len(f.Values) > 0 {
			fmt.Fprintf(&b, " values %s", strings.Join(f.Values, " "))
		}
		for _, c := range f.Conditions {
			fmt.Fprintf(&b, " 88 %s=%s", c.Name, strings.Join(c.Values, ","))
		}
		b.WriteString("\n")
		for _, child := range f.Children {
			visit(child)
		}
	}
	for _, r := range records.WorkingStorage {
		visit(r)
	}
	return b.String(), nil
}

// dataForms are data description entries in forms the specimens do not all
// use, each with its layout in IBM COBOL, and in Micro Focus COBOL where
// it differs, or the error it is not read with.
var dataForms = []struct{ entries, ibm, mf string }{
	{"01 R.\n 05 A PIC S9(5)V99.\n 05 B PIC S9(5)V99 COMP-3.\n 05 C PIC 9(4) COMP.\n 05 D PIC S9(9) BINARY.\n 05 E PIC 9(18) COMP-5.", `01 R 1+25 group DISPLAY
05 A 1+7 numeric DISPLAY
05 B 8+4 numeric COMP-3
05 C 12+2 numeric COMP
05 D 14+4 numeric COMP
05 E 18+8 numeric COMP-5
`, ""},
	{"01 R.\n 05 A PIC S9(3) SIGN LEADING SEPARATE.\n 05 B PIC S9(3) SIGN TRAILING.", `01 R 1+7 group DISPLAY
05 A 1+4 numeric DISPLAY
05 B 5+3 numeric DISPLAY
`, ""},
	{"01 R.\n 05 A PIC ZZ,ZZ9.99-.\n 05 B PIC $$$9.99CR.\n 05 C PIC X(3)B9(2).\n 05 D PIC A(5).", `01 R 1+30 group DISPLAY
05 A 1+10 numeric-edited DISPLAY
05 B 11+9 numeric-edited DISPLAY
05 C 20+6 alphanumeric-edited DISPLAY
05 D 26+5 alphanumeric DISPLAY
`, ""},
	{"01 R.\n 05 A COMP-1.\n 05 B COMP-2.", `01 R 1+12 group DISPLAY
05 A 1+4 numeric COMP-1
05 B 5+8 numeric COMP-2
`, ""},
	{"01 R.\n 05 A USAGE INDEX.", "error: FORMS.cbl:7: A: USAGE INDEX is not supported yet", ""},
	{"01 R.\n 05 A PIC 9(2) COMP.\n 05 B PIC S9(5) COMP.\n 05 C PIC 9(7) BINARY.", `01 R 1+10 group DISPLAY
05 A 1+2 numeric COMP
05 B 3+4 numeric COMP
05 C 7+4 numeric COMP
`, `01 R 1+8 group DISPLAY
05 A 1+1 numeric COMP
05 B 2+3 numeric COMP
05 C 5+4 numeric COMP
`},
	{"01 R.\n 05 N PIC 99.\n 05 T OCCURS 1 TO 5 TIMES DEPENDING ON N ASCENDING KEY K INDEXED BY X.\n  10 K PIC X(2).\n  10 V PIC 9(3).", `01 R 1+27 group DISPLAY
05 N 1+2 numeric DISPLAY
05 T 3+25 group DISPLAY occurs 5
10 K 3+2 alphanumeric DISPLAY
10 V 5+3 numeric DISPLAY
`, ""},
	{"01 R.\n 05 A PIC X(4) VALUE ALL '*'.\n 05 B PIC X(2) VALUE X'0D25'.\n 05 C PIC 9 VALUE ZERO.\n  88 LOW VALUES 0 THRU 3.\n  88 ODD VALUE 1 3 5 7 9.", `01 R 1+7 group DISPLAY
05 A 1+4 alphanumeric DISPLAY values ALL '*'
05 B 5+2 alphanumeric DISPLAY values X'0D25'
05 C 7+1 numeric DISPLAY values ZERO 88 LOW=0 THRU 3 88 ODD=1,3,5,7,9
`, ""},
	{"01 R.\n 05 A PIC X(10) JUSTIFIED RIGHT.\n 05 B PIC 9(5) BLANK WHEN ZERO.", `01 R 1+15 group DISPLAY
05 A 1+10 alphanumeric DISPLAY
05 B 11+5 numeric DISPLAY
`, ""},
	{"01 R.\n 05 A PIC S9(4) COMP SYNCHRONIZED.", "error: FORMS.cbl:7: A: SYNCHRONIZED is not supported yet", ""},
	{"77 COUNTER PIC 9(4) VALUE 0.\n01 R.\n 05 A PIC X(2).\n 05 B REDEFINES A PIC 99.", `77 COUNTER 1+4 numeric DISPLAY values 0
01 R 1+2 group DISPLAY
05 A 1+2 alphanumeric DISPLAY
05 B 1+2 numeric DISPLAY redefines A
`, ""},
	{"01 R EXTERNAL.\n 05 A PIC X.\n01 S GLOBAL.\n 05 B PIC X.", `01 R 1+1 group DISPLAY
05 A 1+1 alphanumeric DISPLAY
01 S 1+1 group DISPLAY
05 B 1+1 alphanumeric DISPLAY
`, ""},
	{"01 R.\n 05 A PIC X(2).\n 05 B PIC X(3).\n66 AB RENAMES A THRU B.", "error: FORMS.cbl:9: AB: RENAMES is not supported yet", ""},
	{"01 R.\n 05 A PIC N(3).\n 05 B PIC X(3) USAGE DISPLAY.\n 05 C PIC 9(3) PACKED-DECIMAL.", "error: FORMS.cbl:7: A: PIC N(3): symbol N is not supported yet", ""},
}

func TestDataForms(t *testing.T) {
	for _, form := range dataForms {
		for dialect, want := range map[*cobol.Dialect]string{&cobol.IBM: form.ibm, &cobol.MicroFocus: cmp.Or(form.mf, form.ibm)} {
			got, err := layoutOf(t, form.entries, *dialect)
			if err != nil {
				got = "error: " + err.Error()
			}
			if got != want {
				t.Errorf("%s [%s]:\ngot\n%s\nwant\n%s", form.entries, dialect.Name, got, want)
			}
		}
	}
}
