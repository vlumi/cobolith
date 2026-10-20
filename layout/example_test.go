package layout_test

import (
	"fmt"

	"github.com/vlumi/cobolith/cobol"
	"github.com/vlumi/cobolith/layout"
)

// Lay out a program's working storage: where each field starts, how many
// bytes it takes, packed decimal and binary as the dialect sizes them.
func Example() {
	source := `>>SOURCE FORMAT IS FREE
IDENTIFICATION DIVISION.
PROGRAM-ID. TOTALS.
DATA DIVISION.
WORKING-STORAGE SECTION.
01  WS-TOTALS.
    05  WS-COUNT    PIC 9(4)        COMP.
    05  WS-AMOUNT   PIC S9(7)V99    COMP-3.
    05  WS-CODE     PIC X(3).
        88  WS-DEPOSIT  VALUE 'DEP'.
PROCEDURE DIVISION.
MAIN-PARA.
    STOP RUN.
`
	tokens, err := cobol.Lex("TOTALS.cbl", []byte(source))
	if err != nil {
		panic(err)
	}
	p, err := cobol.Parse("TOTALS.cbl", tokens)
	if err != nil {
		panic(err)
	}
	records, err := layout.Program(p, cobol.IBM)
	if err != nil {
		panic(err)
	}
	for _, f := range records.WorkingStorage[0].Children {
		fmt.Printf("%-9s %2d-%2d %s\n", f.Name, f.Start, f.Start+f.Size()-1, f.Usage)
	}
	// Output:
	// WS-COUNT   1- 2 COMP
	// WS-AMOUNT  3- 7 COMP-3
	// WS-CODE    8-10 DISPLAY
}
