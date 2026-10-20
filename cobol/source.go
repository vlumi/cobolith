// Package cobol reads COBOL source, in fixed format or, after a
// >>SOURCE FORMAT IS FREE directive, in free format: it splits the lines
// into tokens that keep their position in the original file, expands COPY
// statements, and parses a program's divisions. It reads IBM and Micro
// Focus COBOL, whose differences that matter to a record's layout its
// Dialect says, and the statements of a subset of the language: a verb
// outside it is an error, never a guess.
package cobol

import (
	"fmt"
	"strings"
)

// Fixed-format columns, 1-based: 1-6 sequence area, 7 indicator, 8-11 area A,
// 12-72 area B, 73 on ignored.
const (
	indicatorColumn = 7
	areaAColumn     = 8
	areaBColumn     = 12
	lastColumn      = 72
)

// Pos is a position in a source file.
type Pos struct {
	File string
	Line int
	Col  int
}

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d", p.File, p.Line)
}

// Kind is the kind of a token.
type Kind int

// The kinds of token.
const (
	Word   Kind = iota // a name, reserved word, number or picture string
	String             // a quoted literal, without its quotes
	Period             // the period that ends an entry, sentence or header
	LParen
	RParen
	// CommentEntry is the free text of an identification paragraph such as
	// AUTHOR or DATE-WRITTEN, which the compiler treats as a comment.
	CommentEntry
	// Hex is a hexadecimal literal, X'0D25': its hex digits, without the X
	// and the quotes.
	Hex
)

// commentEntryParagraphs are the identification paragraphs whose text is a
// comment entry.
var commentEntryParagraphs = map[string]bool{
	"AUTHOR": true, "INSTALLATION": true, "DATE-WRITTEN": true,
	"DATE-COMPILED": true, "SECURITY": true, "REMARKS": true,
}

// Token is one lexical unit of the source. Words are upper-cased: COBOL is
// case-insensitive outside literals. Comment holds the comment lines just
// before the token, if it is the first on its line. AreaA is set on a token
// where a division, section or paragraph header can start: in area A in
// fixed format, and first on its line in free format, which has no areas.
type Token struct {
	Kind    Kind
	Text    string
	Pos     Pos
	Comment []string
	AreaA   bool
}

func (t Token) String() string {
	switch t.Kind {
	case String:
		return "'" + t.Text + "'"
	case Hex:
		return "X'" + t.Text + "'"
	case Period:
		return "."
	}
	return t.Text
}

// isLiteral reports whether the token is a quoted literal, of characters
// or of hex digits.
func (t Token) isLiteral() bool {
	return t.Kind == String || t.Kind == Hex
}

// Is reports whether the token is the given word.
func (t Token) Is(word string) bool {
	return t.Kind == Word && t.Text == word
}

// Lex splits fixed-format source into tokens. A block of comment lines (* or
// / in the indicator column) is attached to the token that follows it; a
// blank line ends the block. Debugging lines (D) are skipped. A hyphen in the
// indicator column continues the previous line: an unfinished literal
// continues after the quote that opens the continuation line, anything else
// continues at the line's first non-blank character.
func Lex(file string, source []byte) ([]Token, error) {
	l := &lexer{file: file}
	for i, text := range strings.Split(string(source), "\n") {
		if err := l.line(i+1, strings.TrimRight(text, "\r")); err != nil {
			return nil, err
		}
	}
	if l.open != nil {
		return nil, fmt.Errorf("%s: literal not closed", l.open.Pos)
	}
	l.endCommentEntry()
	return l.tokens, nil
}

type lexer struct {
	file string
	// free is set by a >>SOURCE FORMAT IS FREE directive: lines have no
	// sequence area, indicator or right margin, and *> starts a comment.
	free   bool
	tokens []Token
	open   *Token // a literal still open at the end of the previous line
	quote  byte
	// picture is set after PIC or PICTURE (and IS), when the next word is a
	// picture string, which may contain parentheses, commas and periods.
	picture bool
	comment []string // comment lines waiting for the next token
	entry   *Token   // a comment entry still collecting lines
}

func (l *lexer) line(lineNo int, text string) error {
	if format, ok := sourceFormat(text); ok {
		l.free = format == "FREE"
		return nil
	}
	if l.free {
		l.freeLine(lineNo, text)
		return nil
	}
	if strings.TrimSpace(text[min(len(text), indicatorColumn-1):min(len(text), lastColumn)]) == "" {
		l.comment = nil
		return nil
	}
	indicator := text[indicatorColumn-1]
	body := []byte(text[indicatorColumn:min(len(text), lastColumn)])
	pos := Pos{File: l.file, Line: lineNo}
	switch indicator {
	case '*', '/':
		l.comment = append(l.comment, strings.TrimRight(string(body), " "))
		return nil
	case 'D', 'd':
		return nil
	case '-':
		return l.continuation(pos, body)
	case ' ':
	default:
		return fmt.Errorf("%s: unknown indicator %q", pos, indicator)
	}
	if l.open != nil {
		return fmt.Errorf("%s: literal not closed", l.open.Pos)
	}
	if l.entry != nil {
		if strings.TrimSpace(string(body[:min(len(body), areaBColumn-areaAColumn)])) == "" {
			l.addToCommentEntry(body)
			return nil
		}
		l.endCommentEntry()
	}
	l.scan(pos, body, 0)
	return nil
}

// sourceFormat reads a >>SOURCE FORMAT directive, wherever on its line it
// is: the format it sets, FREE or FIXED.
func sourceFormat(text string) (string, bool) {
	fields := strings.Fields(strings.ToUpper(text))
	if len(fields) < 2 || fields[0] != ">>SOURCE" {
		return "", false
	}
	format := fields[len(fields)-1]
	return format, format == "FREE" || format == "FIXED"
}

// freeLine lexes a line of free-format source: a line that starts with *>
// is a comment, and *> outside a literal ends the line's code.
func (l *lexer) freeLine(lineNo int, text string) {
	trimmed := strings.TrimSpace(text)
	switch {
	case trimmed == "":
		l.comment = nil
		return
	case strings.HasPrefix(trimmed, "*>"):
		l.comment = append(l.comment, strings.TrimSpace(strings.TrimPrefix(trimmed, "*>")))
		return
	}
	body := []byte(text)
	if end := inlineComment(body); end >= 0 {
		body = body[:end]
	}
	l.scan(Pos{File: l.file, Line: lineNo}, body, 0)
}

// inlineComment finds where a *> comment starts on a line, outside the
// literals on it, or -1.
func inlineComment(body []byte) int {
	var quote byte
	for i := 0; i < len(body); i++ {
		switch c := body[i]; {
		case quote != 0 && c == quote:
			quote = 0
		case quote == 0 && (c == '\'' || c == '"'):
			quote = c
		case quote == 0 && c == '*' && i+1 < len(body) && body[i+1] == '>':
			return i
		}
	}
	return -1
}

// addToCommentEntry adds a line's text to the open comment entry.
func (l *lexer) addToCommentEntry(text []byte) {
	if s := strings.TrimSpace(string(text)); s != "" {
		if l.entry.Text != "" {
			l.entry.Text += " "
		}
		l.entry.Text += s
	}
}

func (l *lexer) endCommentEntry() {
	if l.entry != nil {
		l.tokens = append(l.tokens, *l.entry)
		l.entry = nil
	}
}

// startsCommentEntry reports whether the last tokens are the header of an
// identification paragraph with a comment entry, such as AUTHOR followed by
// its period.
func (l *lexer) startsCommentEntry() bool {
	n := len(l.tokens)
	return n >= 2 && l.tokens[n-1].Kind == Period &&
		commentEntryParagraphs[l.tokens[n-2].Text] && l.tokens[n-2].AreaA
}

func (l *lexer) continuation(pos Pos, body []byte) error {
	start := 0
	for start < len(body) && body[start] == ' ' {
		start++
	}
	if start == len(body) {
		return nil
	}
	if l.open == nil {
		l.decimalPoint()
		if len(l.tokens) == 0 || l.tokens[len(l.tokens)-1].Kind != Word {
			return fmt.Errorf("%s: continuation line continues nothing", pos)
		}
		end := wordEnd(body, start, false)
		l.tokens[len(l.tokens)-1].Text += strings.ToUpper(string(body[start:end]))
		l.scan(pos, body, end)
		return nil
	}
	if body[start] != l.quote {
		return fmt.Errorf("%s: a continued literal resumes after a quote", pos)
	}
	l.scan(pos, body, start+1)
	return nil
}

// decimalPoint makes a period the decimal point of the number before it,
// when a continuation line continues it: 999 and then . on a line of its
// own, continued by 777, are 999.777.
func (l *lexer) decimalPoint() {
	n := len(l.tokens)
	if n >= 2 && l.tokens[n-1].Kind == Period && l.tokens[n-2].Kind == Word && isNumeral(l.tokens[n-2].Text) {
		l.tokens[n-2].Text += "."
		l.tokens = l.tokens[:n-1]
	}
}

// isNumeral reports whether a word is a number as written, signed or not.
func isNumeral(word string) bool {
	digits := strings.TrimLeft(word, "+-")
	return digits != "" && strings.Trim(digits, "0123456789") == ""
}

// scan lexes body from index i. If a literal is open, it continues first.
func (l *lexer) scan(pos Pos, body []byte, i int) {
	if l.open != nil {
		i = l.literal(body, i)
	}
	for i < len(body) {
		c := body[i]
		pos.Col = l.column(i)
		switch {
		case c == ' ':
			i++
		case c == '\'' || c == '"':
			l.open = &Token{Kind: String, Pos: pos}
			l.quote = c
			i = l.literal(body, i+1)
		case c == '(' && !l.picture:
			l.emit(Token{Kind: LParen, Text: "(", Pos: pos})
			i++
		case c == ')' && !l.picture:
			l.emit(Token{Kind: RParen, Text: ")", Pos: pos})
			i++
		case separator(body, i):
			if c == '.' {
				l.emit(Token{Kind: Period, Text: ".", Pos: pos})
				if l.startsCommentEntry() {
					l.entry = &Token{Kind: CommentEntry, Pos: pos}
					l.addToCommentEntry(body[i+1:])
					return
				}
			}
			i++
		case (c == 'X' || c == 'x') && i+1 < len(body) && (body[i+1] == '\'' || body[i+1] == '"'):
			l.open = &Token{Kind: Hex, Pos: pos}
			l.quote = body[i+1]
			i = l.literal(body, i+2)
		default:
			end := wordEnd(body, i, l.picture)
			l.emit(Token{Kind: Word, Text: strings.ToUpper(string(body[i:end])), Pos: pos})
			i = end
		}
	}
}

// column is the column, 1-based, of a byte of a line's code.
func (l *lexer) column(i int) int {
	if l.free {
		return i + 1
	}
	return indicatorColumn + 1 + i
}

// literal reads an open literal from body[i:], up to its closing quote or,
// if it does not close on this line, through column 72.
func (l *lexer) literal(body []byte, i int) int {
	for i < len(body) {
		if body[i] == l.quote {
			if i+1 < len(body) && body[i+1] == l.quote {
				l.open.Text += string(l.quote)
				i += 2
				continue
			}
			l.emit(*l.open)
			l.open = nil
			return i + 1
		}
		l.open.Text += string(body[i])
		i++
	}
	l.open.Text += strings.Repeat(" ", lastColumn-indicatorColumn-len(body))
	return i
}

func (l *lexer) emit(t Token) {
	t.AreaA = l.inAreaA(t.Pos)
	if l.comment != nil {
		t.Comment, l.comment = l.comment, nil
	}
	switch {
	case t.Is("PIC") || t.Is("PICTURE"):
		l.picture = true
	case l.picture && t.Is("IS"):
	default:
		l.picture = false
	}
	l.tokens = append(l.tokens, t)
}

// inAreaA reports whether a token at a position starts where a header can.
func (l *lexer) inAreaA(pos Pos) bool {
	if l.free {
		return len(l.tokens) == 0 || l.tokens[len(l.tokens)-1].Pos.Line != pos.Line
	}
	return pos.Col >= areaAColumn && pos.Col < areaBColumn
}

// separator reports whether body[i] is a period, comma or semicolon that
// separates rather than belongs to a word: one followed by a space or the
// end of the line.
func separator(body []byte, i int) bool {
	switch body[i] {
	case '.', ',', ';':
		return i+1 == len(body) || body[i+1] == ' '
	}
	return false
}

func wordEnd(body []byte, i int, picture bool) int {
	for i < len(body) {
		c := body[i]
		if c == ' ' || c == '\'' || c == '"' || separator(body, i) {
			break
		}
		if !picture && (c == '(' || c == ')') {
			break
		}
		i++
	}
	return i
}
