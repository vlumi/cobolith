package cobol

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// describe writes statements as an indented outline: verb, condition,
// procedures, reads and writes, then the branches.
func describe(b *strings.Builder, statements []Statement, indent string) {
	for _, s := range statements {
		b.WriteString(indent + s.Verb)
		if s.Condition != nil {
			b.WriteString(" [" + s.Condition.String() + "]")
		}
		for _, ref := range s.Procedures {
			b.WriteString(" -> " + ref.Name)
			if ref.Thru != "" {
				b.WriteString(" THRU " + ref.Thru)
			}
		}
		for _, f := range s.Files {
			fmt.Fprintf(b, " file %s", strings.TrimSpace(f.Mode+" "+f.Name))
		}
		if s.Calls != "" {
			b.WriteString(" calls " + s.Calls)
		}
		for _, list := range []struct {
			label string
			ops   []Operand
		}{{"reads", s.Reads}, {"writes", s.Writes}} {
			if len(list.ops) == 0 {
				continue
			}
			var texts []string
			for _, op := range list.ops {
				texts = append(texts, op.Text)
			}
			fmt.Fprintf(b, " %s %s", list.label, strings.Join(texts, ", "))
		}
		b.WriteString("\n")
		for _, branch := range s.Branches {
			b.WriteString(indent + "  " + branch.Kind)
			if branch.Condition != nil {
				b.WriteString(" [" + branch.Condition.String() + "]")
			}
			b.WriteString("\n")
			describe(b, branch.Statements, indent+"    ")
		}
	}
}

func TestStatements(t *testing.T) {
	p, err := ReadProgram("testdata/program/STATEMENTS.cbl", nil)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	describe(&b, p.Paragraphs[0].Statements, "")
	want := `MOVE reads WS-A writes WS-B, WS-C(WS-I)
ADD reads 1, WS-A, WS-B writes WS-B
SUBTRACT reads WS-A, WS-B writes WS-C
DIVIDE reads WS-A, 365 writes WS-D
COMPUTE reads WS-A * (WS-B + 1) - FUNCTION MOD(WS-C 7) writes WS-E
SET writes DONE
SET reads WS-I writes WS-X
STRING reads 'A', WS-F(1:6) writes WS-G
IF [WS-A = 1 OR (WS-A = 2 AND NOT DONE)]
  THEN
    PERFORM -> OTHER-PARA
  ELSE
    IF [WS-B IS NOT NUMERIC]
      THEN
        GO -> OTHER-PARA
EVALUATE reads WS-A
  WHEN [(WS-A = 1 AND WS-B > 2) OR (WS-A >= 3 AND WS-A <= 5)]
    CONTINUE
  OTHER
    DISPLAY reads 'NO', WS-A
PERFORM [WS-I > WS-N] reads 1, 1 writes WS-I
  BODY
    READ file IN-FILE
      AT END
        SET writes DONE
      NOT AT END
        WRITE reads IN-RECORD writes OUT-RECORD
PERFORM [DONE] -> OTHER-PARA THRU OTHER-EXIT
SEARCH reads WS-ENTRY
  AT END
    MOVE reads 0 writes WS-A
  WHEN [WS-CODE(WS-CX) = 'DP']
    MOVE reads WS-TEXT(WS-CX) writes WS-G
OPEN file INPUT IN-FILE file OUTPUT OUT-FILE
SORT file WORK-FILE file INPUT IN-FILE file OUTPUT OUT-FILE reads WS-A
GO -> OTHER-PARA -> OTHER-EXIT reads WS-A
IF [WS-A >= WS-B]
  THEN
    STOP
STOP
`
	if got := b.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	compute := p.Paragraphs[0].Statements[4].Reads[0]
	if got := strings.Join(compute.Refs, " "); got != "WS-A WS-B WS-C" {
		t.Errorf("COMPUTE reads %s, want WS-A WS-B WS-C", got)
	}
}

func TestStatementText(t *testing.T) {
	p, err := ReadProgram("testdata/program/STATEMENTS.cbl", nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	Walk(p.Paragraphs[0].Statements, func(s Statement) { got = append(got, s.Text) })
	for _, want := range []string{
		"MOVE WS-A TO WS-B WS-C(WS-I)",
		"COMPUTE WS-E = WS-A * (WS-B + 1) - FUNCTION MOD(WS-C 7)",
		"STRING 'A' WS-F(1:6) DELIMITED BY SIZE INTO WS-G",
		"IF WS-A = 1 OR 2 AND NOT DONE",
		"EVALUATE WS-A ALSO TRUE",
		"PERFORM VARYING WS-I FROM 1 BY 1 UNTIL WS-I > WS-N",
		"READ IN-FILE",
		"WRITE OUT-RECORD FROM IN-RECORD",
		"SEARCH WS-ENTRY",
		"GO TO OTHER-PARA OTHER-EXIT DEPENDING ON WS-A",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("no statement %q in\n%q", want, got)
		}
	}
}

func TestIndexedAndCalls(t *testing.T) {
	source := `           READ CARD-FILE KEY IS CR-NUMBER
               INVALID KEY MOVE 'N' TO WS-FOUND
           END-READ
           REWRITE CARD-RECORD FROM WS-CARD
               INVALID KEY MOVE 'E' TO WS-STATUS
           END-REWRITE
           DELETE HOLD-FILE RECORD
           START HOLD-FILE KEY IS NOT LESS THAN HD-KEY
               INVALID KEY SET HOLDS-DONE TO TRUE
           END-START
           CALL 'AUTHPUR' USING WS-REQUEST BY CONTENT WS-DATE
               ON EXCEPTION MOVE 'X' TO WS-STATUS
           END-CALL
           CALL WS-PROGRAM USING BY REFERENCE WS-REQUEST
           ACCEPT WS-REQUEST
               ON EXCEPTION SET END-OF-INPUT TO TRUE
               NOT ON EXCEPTION DISPLAY WS-REQUEST
           END-ACCEPT.
`
	tokens, err := Lex("test", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	statements, err := ParseStatements(tokens, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	describe(&b, statements, "")
	want := `READ file CARD-FILE reads CR-NUMBER
  INVALID KEY
    MOVE reads 'N' writes WS-FOUND
REWRITE reads WS-CARD writes CARD-RECORD
  INVALID KEY
    MOVE reads 'E' writes WS-STATUS
DELETE file HOLD-FILE
START file HOLD-FILE reads HD-KEY
  INVALID KEY
    SET writes HOLDS-DONE
CALL calls AUTHPUR reads WS-REQUEST, WS-DATE writes WS-REQUEST
  ON EXCEPTION
    MOVE reads 'X' writes WS-STATUS
CALL reads WS-PROGRAM, WS-REQUEST writes WS-REQUEST
ACCEPT writes WS-REQUEST
  ON EXCEPTION
    SET writes END-OF-INPUT
  NOT ON EXCEPTION
    DISPLAY reads WS-REQUEST
`
	if got := b.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// An identifier after OR is the object of an abbreviated relation, unless
// it is a condition name; a qualified condition name is one too.
func TestConditionNames(t *testing.T) {
	conditions := map[string]bool{"AM-OPEN": true}
	for source, want := range map[string]string{
		"IF A = B OR C CONTINUE.":                "A = B OR A = C",
		"IF A = B OR AM-OPEN CONTINUE.":          "A = B OR AM-OPEN",
		"IF AM-OPEN OF ACCT-RECORD CONTINUE.":    "AM-OPEN OF ACCT-RECORD",
		"IF C AND A = B CONTINUE.":               "C AND A = B",
		"IF A NOT = B AND NOT AM-OPEN CONTINUE.": "A NOT = B AND NOT AM-OPEN",
	} {
		tokens, err := Lex("test", []byte("           "+source+"\n"))
		if err != nil {
			t.Fatal(err)
		}
		statements, err := ParseStatements(tokens, nil, conditions)
		if err != nil {
			t.Errorf("%s: %v", source, err)
			continue
		}
		if got := statements[0].Condition.String(); got != want {
			t.Errorf("%s: condition %s, want %s", source, got, want)
		}
	}
}

func TestStatementErrors(t *testing.T) {
	procedures := map[string]bool{"P": true}
	for _, source := range []string{
		"INSPECT WS-A TALLYING WS-B FOR ALL 'X'.",
		"MOVE WS-A WS-B.",
		"PERFORM UNTIL WS-A = 1 CONTINUE.",
		"GO TO NOWHERE.",
	} {
		tokens, err := Lex("test", []byte("           "+source+"\n"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseStatements(tokens, procedures, nil); err == nil {
			t.Errorf("%s: parsed without error", source)
		}
	}
}
