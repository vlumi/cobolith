package main

import (
	"reflect"
	"strings"
	"testing"
)

// A population file splits into its members, a called program named in
// the fifth field of its header, and the optional code in its lines kept
// for the options selected and made a comment for the rest.
func TestReadPopulation(t *testing.T) {
	population := `VERSION LINE
*HEADER,COBOL,MAIN01
000100 IDENTIFICATION DIVISION.
000200A    DISPLAY 'KEPT'.
000300Q    DISPLAY 'DROPPED'.
000400D    DISPLAY 'DEBUG'.
*END-OF,MAIN01
*HEADER,COBOL,MAIN01,SUBRTN,SUB01
000100 IDENTIFICATION DIVISION.
*END-OF,SUB01
*HEADER,CLBRY,LIB01
000100     05  LB-ITEM PIC X.
*END-OF,LIB01
*HEADER,DATA*,DATA01
SOME DATA
*END-OF,DATA01
`
	members, err := readPopulation(strings.NewReader(population))
	if err != nil {
		t.Fatal(err)
	}
	want := []member{
		{kind: "COBOL", name: "MAIN01", lines: []string{
			"000100 IDENTIFICATION DIVISION.",
			"000200     DISPLAY 'KEPT'.",
			"000300*Q   DISPLAY 'DROPPED'.",
			"000400D    DISPLAY 'DEBUG'.",
		}},
		{kind: "COBOL", name: "SUB01", lines: []string{"000100 IDENTIFICATION DIVISION."}},
		{kind: "CLBRY", name: "LIB01", lines: []string{"000100     05  LB-ITEM PIC X."}},
		{kind: "DATA*", name: "DATA01", lines: []string{"SOME DATA"}},
	}
	if !reflect.DeepEqual(members, want) {
		t.Errorf("members\n%q\nwant\n%q", members, want)
	}
}

// A failure's reason leaves out where it happened.
func TestReasonOf(t *testing.T) {
	if got := reasonOf("NC101A.CBL:12: MAIN-PARA: ALTER is not supported yet"); got != "ALTER is not supported yet" {
		t.Errorf("reason %q", got)
	}
}
