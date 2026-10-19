package cobol

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// fixed builds fixed-format source from lines written from column 7 on, so
// tests can show the indicator column without the sequence area.
func fixed(lines ...string) []byte {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString("000100" + line + "\n")
	}
	return []byte(b.String())
}

func texts(tokens []Token) []string {
	var out []string
	for _, t := range tokens {
		out = append(out, t.String())
	}
	return out
}

func TestLex(t *testing.T) {
	source := fixed(
		"*COMMENT LINE IS SKIPPED",
		" 01  amount       PIC S9(7)V99 VALUE -1.25.",
		" 01  edited       PIC IS Z,ZZ9.99-.",
		"     MOVE 'IT''S' TO X(I), Y.",
		"D    DISPLAY 'DEBUG'.",
		fmt.Sprintf("%-66s%s", " 01  FAR-RIGHT    PIC X.", "IGNORED FROM COLUMN 73"),
	)
	tokens, err := Lex("T.cbl", source)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"01", "AMOUNT", "PIC", "S9(7)V99", "VALUE", "-1.25", ".",
		"01", "EDITED", "PIC", "IS", "Z,ZZ9.99-", ".",
		"MOVE", "'IT'S'", "TO", "X", "(", "I", ")", "Y", ".",
		"01", "FAR-RIGHT", "PIC", "X", ".",
	}
	if got := texts(tokens); !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if pos := tokens[1].Pos; pos.Line != 2 || pos.Col != 12 || pos.InAreaA() {
		t.Errorf("AMOUNT at %+v, want line 2, column 12, area B", pos)
	}
	if pos := tokens[0].Pos; !pos.InAreaA() {
		t.Errorf("level 01 at %+v, want area A", pos)
	}
}

func TestLexContinuation(t *testing.T) {
	source := fixed(
		"     MOVE 'A LITERAL THAT RUNS PAST COLUMN SEVENTY-TWO AND IS CONT",
		"-        'INUED' TO WS-MES",
		"-        SAGE.",
	)
	tokens, err := Lex("T.cbl", source)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"MOVE", "'A LITERAL THAT RUNS PAST COLUMN SEVENTY-TWO AND IS CONTINUED'",
		"TO", "WS-MESSAGE", ".",
	}
	if got := texts(tokens); !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestLexContinuationPadsToColumn72(t *testing.T) {
	source := fixed(
		"     MOVE 'SHORT",
		"-        'END' TO X.",
	)
	tokens, err := Lex("T.cbl", source)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := tokens[1].Text, "SHORT"+strings.Repeat(" ", 50)+"END"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLexErrors(t *testing.T) {
	tests := map[string][]byte{
		"unclosed literal":          fixed("     MOVE 'OPEN TO X."),
		"unknown indicator":         fixed("X    MOVE A TO B."),
		"continuation without open": fixed("-        'TEXT'."),
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Lex("T.cbl", source); err == nil {
				t.Error("lexed without error")
			}
		})
	}
}

func TestLexAttachesCommentsToTheNextToken(t *testing.T) {
	source := fixed(
		"*=========================",
		"* MAIN: DOES THE WORK",
		"*=========================",
		" MAIN.",
		"*    A STRAY NOTE",
		"",
		"     DISPLAY 'X'.",
	)
	tokens, err := Lex("T.cbl", source)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"=========================", " MAIN: DOES THE WORK", "========================="}
	if got := tokens[0].Comment; !slices.Equal(got, want) {
		t.Errorf("MAIN comment %q, want %q", got, want)
	}
	if got := tokens[2].Comment; got != nil {
		t.Errorf("DISPLAY comment %q, want none: the blank line ends the block", got)
	}
}

func TestLexCommentEntries(t *testing.T) {
	source := fixed(
		" IDENTIFICATION DIVISION.",
		" PROGRAM-ID. TXPOST.",
		" AUTHOR. M. O'KESTREL, OPERATIONS.",
		"         SECOND LINE.",
		" DATE-WRITTEN. 1987-03-12.",
		" ENVIRONMENT DIVISION.",
	)
	tokens, err := Lex("T.cbl", source)
	if err != nil {
		t.Fatal(err)
	}
	var entries []string
	for _, tok := range tokens {
		if tok.Kind == CommentEntry {
			entries = append(entries, tok.Text)
		}
	}
	want := []string{"M. O'KESTREL, OPERATIONS. SECOND LINE.", "1987-03-12."}
	if !slices.Equal(entries, want) {
		t.Errorf("comment entries %q, want %q", entries, want)
	}
	if last := tokens[len(tokens)-3]; !last.Is("ENVIRONMENT") {
		t.Errorf("%s after the entries, want ENVIRONMENT", last)
	}
}

// After >>SOURCE FORMAT IS FREE, a line has no columns: a word may start
// anywhere, *> starts a comment, outside a literal, and words are upper
// case whatever case they are written in.
func TestFreeFormat(t *testing.T) {
	source := `>>source format is free
*> the request
move "a *> b" to ws-text *> not code
    display ws-text.
`
	tokens, err := Lex("test", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tok := range tokens {
		got = append(got, fmt.Sprintf("%d:%d %s", tok.Pos.Line, tok.Pos.Col, tok.Text))
	}
	want := []string{"3:1 MOVE", "3:6 a *> b", "3:15 TO", "3:18 WS-TEXT", "4:5 DISPLAY", "4:13 WS-TEXT", "4:20 ."}
	if !slices.Equal(got, want) {
		t.Errorf("tokens %q, want %q", got, want)
	}
	if !slices.Equal(tokens[0].Comment, []string{"the request"}) {
		t.Errorf("comment %q", tokens[0].Comment)
	}
}
