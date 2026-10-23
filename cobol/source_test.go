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
	if tok := tokens[1]; tok.Pos.Line != 2 || tok.Pos.Col != 12 || tok.AreaA {
		t.Errorf("AMOUNT at %+v, want line 2, column 12, area B", tok.Pos)
	}
	if tok := tokens[0]; !tok.AreaA {
		t.Errorf("level 01 at %+v, want area A", tok.Pos)
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
	if !tokens[0].AreaA || tokens[2].AreaA || !tokens[4].AreaA {
		t.Error("in free format, the first token of each line, and only it, is where a header can start")
	}
}

// A number continued across lines, its decimal point on a line of its own,
// is one number.
func TestContinuedNumber(t *testing.T) {
	source := "" +
		"           MOVE     -\n" +
		"      -             9\n" +
		"      -             9\n" +
		"      -             .\n" +
		"      -             7\n" +
		"      -             7 TO CONT-B.\n"
	tokens, err := Lex("test", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tok := range tokens {
		got = append(got, tok.String())
	}
	if want := "MOVE -99.77 TO CONT-B ."; strings.Join(got, " ") != want {
		t.Errorf("tokens %q, want %q", strings.Join(got, " "), want)
	}
}

// A picture string may be continued across lines, parentheses and all.
func TestContinuedPicture(t *testing.T) {
	tokens, err := Lex("test", []byte("       01  A PIC S9(\n      -        6)V9(6).\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := tokens[3].Text; got != "S9(6)V9(6)" {
		t.Errorf("picture %q", got)
	}
}

// In Shift_JIS, a two-byte character stays whole and its second byte is
// not upper-cased, though it may be an ASCII letter's; and N, NC and NX
// literals are national.
func TestShiftJIS(t *testing.T) {
	source := "       01  \x8a\x61-\x93\xfa PIC N(2) VALUE N\"\x93\xfa\x96\x7b\".\n" +
		"           MOVE NC\"\x93\xfa\" TO A\n           MOVE NX\"93FA\" TO B\n"
	tokens, err := LexIn("test", []byte(source), ShiftJIS)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tok := range tokens {
		got = append(got, fmt.Sprintf("%d:%s", tok.Kind, tok.Text))
	}
	want := []string{"0:01", "0:蛎-日", "0:PIC", "0:N(2)", "0:VALUE", "7:日本", "2:.",
		"0:MOVE", "7:日", "0:TO", "0:A", "0:MOVE", "6:93FA", "0:TO", "0:B"}
	if !slices.Equal(got, want) {
		t.Errorf("tokens\n%q\nwant\n%q", got, want)
	}
	ascii, err := Lex("test", []byte("       MOVE \x8a\x61 TO B\n"))
	if err != nil {
		t.Fatal(err)
	}
	if ascii[1].Text != "\x8a\x41" {
		t.Errorf("without an encoding, the word is %q, want its ASCII letter upper-cased", ascii[1].Text)
	}
}

// Text beside a Shift_JIS source is transcoded as its tokens are.
func TestToUTF8(t *testing.T) {
	got, err := ToUTF8("\x93\xfa\x96\x7b", ShiftJIS)
	if err != nil || got != "日本" {
		t.Errorf("ToUTF8 gave %q, %v, want 日本", got, err)
	}
	if got, err := ToUTF8("plain", ""); err != nil || got != "plain" {
		t.Errorf("ToUTF8 of ASCII gave %q, %v", got, err)
	}
}

// In free format a literal closes on its line, there being no continuation
// lines: an unclosed one is an error, however long the line, where it was
// once padded to column 72 as a fixed-format literal is.
func TestFreeFormatLiteralMustClose(t *testing.T) {
	for _, line := range []string{
		"DISPLAY 'short",
		"DISPLAY 'an unclosed literal on a line of well over sixty-five bytes, which is longer than area B'",
		"DISPLAY \"an unclosed literal on a line of well over sixty-five bytes, which is longer than area B",
	} {
		_, err := Lex("x.cbl", []byte(">>SOURCE FORMAT FREE\n"+line+"\n"))
		if (err == nil) != strings.HasSuffix(line, "'") || err != nil && !strings.Contains(err.Error(), "literal not closed") {
			t.Errorf("%q: %v", line, err)
		}
	}
}
