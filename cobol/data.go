package cobol

import (
	"fmt"
	"strings"
)

// DataEntry is one data description entry as written: a level, a name and
// the clauses that shape the item. Building the hierarchy and computing
// offsets is the layout's work.
type DataEntry struct {
	Level       int
	Name        string // FILLER when the entry has no name
	Picture     string
	Usage       string // DISPLAY, COMP or COMP-3
	Redefines   string
	Occurs      int // 0 when the item does not repeat; the maximum for DEPENDING ON
	OccursMin   int
	DependingOn string
	Values      []string // VALUE literals; for level 88, ranges as "A THRU B"
	Comment     []string // the comment lines before the entry
	Pos         Pos
}

// usages maps the spellings of each supported usage to one name.
var usages = map[string]string{
	"DISPLAY":         "DISPLAY",
	"COMP":            "COMP",
	"COMPUTATIONAL":   "COMP",
	"BINARY":          "COMP",
	"COMP-4":          "COMP",
	"COMPUTATIONAL-4": "COMP",
	"COMP-3":          "COMP-3",
	"COMPUTATIONAL-3": "COMP-3",
	"PACKED-DECIMAL":  "COMP-3",
}

// ParseDataEntry parses the tokens of one data description entry, without
// its closing period.
func ParseDataEntry(tokens []Token) (*DataEntry, error) {
	c := &cursor{tokens: tokens}
	level, err := c.number()
	if err != nil {
		return nil, err
	}
	entry := &DataEntry{Level: level, Name: "FILLER", Comment: tokens[0].Comment, Pos: tokens[0].Pos}
	if t := c.peek(); t.Kind == Word && !isClause(t.Text) {
		entry.Name = c.next().Text
	}
	for !c.done() {
		if err := entry.parseClause(c); err != nil {
			return nil, err
		}
	}
	return entry, nil
}

func isClause(word string) bool {
	switch word {
	case "PIC", "PICTURE", "USAGE", "REDEFINES", "OCCURS", "VALUE", "VALUES":
		return true
	}
	_, isUsage := usages[word]
	return isUsage
}

func (e *DataEntry) parseClause(c *cursor) error {
	clause := c.next()
	switch {
	case clause.Is("PIC") || clause.Is("PICTURE"):
		c.optional("IS")
		e.Picture = c.next().Text
	case clause.Is("USAGE"):
		c.optional("IS")
		return e.setUsage(c.next())
	case usages[clause.Text] != "":
		return e.setUsage(clause)
	case clause.Is("REDEFINES"):
		e.Redefines = c.next().Text
	case clause.Is("OCCURS"):
		return e.parseOccurs(c)
	case clause.Is("VALUE") || clause.Is("VALUES"):
		return e.parseValues(c)
	default:
		return fmt.Errorf("%s: %s: %s is not supported yet", clause.Pos, e.Name, clause)
	}
	return nil
}

func (e *DataEntry) setUsage(t Token) error {
	usage, ok := usages[t.Text]
	if !ok {
		return fmt.Errorf("%s: %s: USAGE %s is not supported yet", t.Pos, e.Name, t)
	}
	e.Usage = usage
	return nil
}

// parseOccurs parses OCCURS n [TIMES] or OCCURS n TO m [TIMES] DEPENDING [ON] x.
func (e *DataEntry) parseOccurs(c *cursor) error {
	n, err := c.number()
	if err != nil {
		return err
	}
	e.Occurs = n
	if c.optional("TO") {
		if e.Occurs, err = c.number(); err != nil {
			return err
		}
		e.OccursMin = n
	}
	c.optional("TIMES")
	if c.optional("DEPENDING") {
		c.optional("ON")
		e.DependingOn = c.next().Text
	}
	return nil
}

// parseValues reads the literals of a VALUE clause up to the next clause.
func (e *DataEntry) parseValues(c *cursor) error {
	c.optional("IS")
	c.optional("ARE")
	for !c.done() && !isClause(c.peek().Text) {
		value, err := literal(c)
		if err != nil {
			return err
		}
		if c.optional("THRU") || c.optional("THROUGH") {
			upper, err := literal(c)
			if err != nil {
				return err
			}
			value += " THRU " + upper
		}
		e.Values = append(e.Values, value)
	}
	if len(e.Values) == 0 {
		return fmt.Errorf("%s: %s: VALUE without a literal", e.Pos, e.Name)
	}
	return nil
}

// literal reads a literal as written: a quoted string, a number, a
// figurative constant, or ALL followed by a literal.
func literal(c *cursor) (string, error) {
	t := c.next()
	switch {
	case t.Kind == String:
		return "'" + strings.ReplaceAll(t.Text, "'", "''") + "'", nil
	case t.Is("ALL"):
		rest, err := literal(c)
		return "ALL " + rest, err
	case t.Kind == Word && (isNumber(t.Text) || figurative[t.Text]):
		return t.Text, nil
	}
	return "", fmt.Errorf("%s: expected a literal, found %s", t.Pos, t)
}

var figurative = map[string]bool{
	"ZERO": true, "ZEROS": true, "ZEROES": true,
	"SPACE": true, "SPACES": true,
	"HIGH-VALUE": true, "HIGH-VALUES": true,
	"LOW-VALUE": true, "LOW-VALUES": true,
	"QUOTE": true, "QUOTES": true,
}

// isNumber reports whether s is a numeric literal: digits with an optional
// sign and decimal point.
func isNumber(s string) bool {
	s = strings.TrimLeft(s, "+-")
	whole, fraction, _ := strings.Cut(s, ".")
	return whole+fraction != "" && digits(whole) && digits(fraction)
}

func digits(s string) bool {
	return strings.Trim(s, "0123456789") == ""
}
