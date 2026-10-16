// Package cobol reads fixed-format COBOL source: it splits the lines into
// tokens that keep their position in the original file, expands COPY
// statements, and parses a program's divisions.
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

// InAreaA reports whether the position is in area A, where division,
// section and paragraph headers start.
func (p Pos) InAreaA() bool {
	return p.Col >= areaAColumn && p.Col < areaBColumn
}

// Kind is the kind of a token.
type Kind int

const (
	Word   Kind = iota // a name, reserved word, number or picture string
	String             // a quoted literal, without its quotes
	Period             // the period that ends an entry, sentence or header
	LParen
	RParen
	// CommentEntry is the free text of an identification paragraph such as
	// AUTHOR or DATE-WRITTEN, which the compiler treats as a comment.
	CommentEntry
)

// commentEntryParagraphs are the identification paragraphs whose text is a
// comment entry.
var commentEntryParagraphs = map[string]bool{
	"AUTHOR": true, "INSTALLATION": true, "DATE-WRITTEN": true,
	"DATE-COMPILED": true, "SECURITY": true, "REMARKS": true,
}

// Token is one lexical unit of the source. Words are upper-cased: COBOL is
// case-insensitive outside literals. Comment holds the comment lines just
// before the token, if it is the first on its line.
type Token struct {
	Kind    Kind
	Text    string
	Pos     Pos
	Comment []string
}

func (t Token) String() string {
	switch t.Kind {
	case String:
		return "'" + t.Text + "'"
	case Period:
		return "."
	}
	return t.Text
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
	file   string
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
		commentEntryParagraphs[l.tokens[n-2].Text] && l.tokens[n-2].Pos.InAreaA()
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

// scan lexes body from index i. If a literal is open, it continues first.
func (l *lexer) scan(pos Pos, body []byte, i int) {
	if l.open != nil {
		i = l.literal(body, i)
	}
	for i < len(body) {
		c := body[i]
		pos.Col = indicatorColumn + 1 + i
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
		default:
			end := wordEnd(body, i, l.picture)
			l.emit(Token{Kind: Word, Text: strings.ToUpper(string(body[i:end])), Pos: pos})
			i = end
		}
	}
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
