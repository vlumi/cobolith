package cobol_test

import (
	"fmt"

	"github.com/vlumi/cobolith/cobol"
)

// Parse a program and walk its statements, nested ones included, with the
// line each is on.
func Example() {
	source := `>>SOURCE FORMAT IS FREE
IDENTIFICATION DIVISION.
PROGRAM-ID. GREET.
DATA DIVISION.
WORKING-STORAGE SECTION.
01  WS-COUNT    PIC 9(2) VALUE 0.
PROCEDURE DIVISION.
MAIN-PARA.
    PERFORM UNTIL WS-COUNT = 3
        ADD 1 TO WS-COUNT
        IF WS-COUNT = 2
            DISPLAY 'TWO'
        END-IF
    END-PERFORM
    STOP RUN.
`
	tokens, err := cobol.Lex("GREET.cbl", []byte(source))
	if err != nil {
		panic(err)
	}
	p, err := cobol.Parse("GREET.cbl", tokens)
	if err != nil {
		panic(err)
	}
	for _, para := range p.Paragraphs {
		cobol.Walk(para.Statements, func(s cobol.Statement) {
			fmt.Printf("%s %d: %s\n", para.Name, s.Pos.Line, s.Text)
		})
	}
	// Output:
	// MAIN-PARA 9: PERFORM UNTIL WS-COUNT = 3
	// MAIN-PARA 10: ADD 1 TO WS-COUNT
	// MAIN-PARA 11: IF WS-COUNT = 2
	// MAIN-PARA 12: DISPLAY 'TWO'
	// MAIN-PARA 15: STOP RUN
}
