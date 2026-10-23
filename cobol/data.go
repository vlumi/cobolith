package cobol

import (
	"fmt"
	"strings"
)

// DataEntry is one data description entry as written: a level, a name and
// the clauses that shape the item. Building the hierarchy and computing
// offsets is the layout's work.
type DataEntry struct {
	Level   int
	Name    string // FILLER when the entry has no name
	Picture string
	Usage   string // DISPLAY, NATIONAL, COMP, COMP-1, COMP-2, COMP-3, COMP-5 or INDEX
	// Sign is where a signed display number's sign is, LEADING or
	// TRAILING, if a SIGN clause says; SignSeparate when it takes a byte
	// of its own rather than being overpunched on a digit.
	Sign         string
	SignSeparate bool
	Redefines    string
	Occurs       int // 0 when the item does not repeat; the maximum for DEPENDING ON
	OccursMin    int
	DependingOn  string
	IndexedBy    []string // the index names of an OCCURS table
	// Keys are the ASCENDING and DESCENDING keys of an OCCURS table, which
	// SEARCH ALL searches by.
	Keys   []TableKey
	Values []string // VALUE literals; for level 88, ranges as "A THRU B"
	// Justified is JUSTIFIED RIGHT, BlankWhenZero BLANK WHEN ZERO, External
	// and Global the clauses of a record's scope; none changes its layout.
	Justified     bool
	BlankWhenZero bool
	External      bool
	Global        bool
	// Synchronized is SYNCHRONIZED, LEFT or RIGHT: the item aligned on its
	// natural boundary, as the dialect does it.
	Synchronized bool
	// Renames and RenamesThru are a level-66 entry's RENAMES: the field it
	// renames, or the first and last of the fields it renames together.
	Renames, RenamesThru string
	// The clauses of a report group of the REPORT SECTION. Type is the
	// part of the report a level-01 group is, REPORT HEADING, PAGE
	// HEADING, CONTROL HEADING, DETAIL, CONTROL FOOTING, PAGE FOOTING or
	// REPORT FOOTING, and TypeControl the control item of a CONTROL
	// HEADING or FOOTING, FINAL for the whole report's. Line is where a
	// line is printed, as written: a line number, PLUS and a number, or
	// NEXT PAGE. Column is where a field starts on its line. Source is the
	// item a field prints, and Sum the items a field adds up.
	Type        string
	TypeControl string
	Line        string
	Column      int
	Source      string
	Sum         []string
	Comment     []string // the comment lines before the entry
	Pos         Pos
}

// TableKey is a key of an OCCURS table, and whether it is descending.
type TableKey struct {
	Name       string
	Descending bool
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
	"COMP-5":          "COMP-5",
	"COMPUTATIONAL-5": "COMP-5",
	"COMP-1":          "COMP-1",
	"COMPUTATIONAL-1": "COMP-1",
	"COMP-2":          "COMP-2",
	"COMPUTATIONAL-2": "COMP-2",
	"INDEX":           "INDEX",
	"NATIONAL":        "NATIONAL",
}

// ParseDataEntry parses the tokens of one data description entry, without
// its closing period.
func ParseDataEntry(tokens []Token) (*DataEntry, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("a data description entry has no tokens")
	}
	c := &cursor{tokens: tokens}
	level, err := c.number()
	if err != nil {
		return nil, err
	}
	entry := &DataEntry{Level: level, Name: "FILLER", Comment: tokens[0].Comment, Pos: tokens[0].Pos}
	if t := c.peek(); t.Kind == Word && !isClause(t) {
		entry.Name = c.next().Text
	}
	for !c.done() {
		if err := entry.parseClause(c); err != nil {
			return nil, err
		}
	}
	return entry, nil
}

// isClause reports whether a token is a word that starts a clause of a data
// description entry; a literal of that text is not.
func isClause(t Token) bool {
	if t.Kind != Word {
		return false
	}
	switch t.Text {
	case "PIC", "PICTURE", "USAGE", "REDEFINES", "OCCURS", "VALUE", "VALUES", "SIGN", "LEADING", "TRAILING",
		"ASCENDING", "DESCENDING", "INDEXED", "JUSTIFIED", "JUST", "BLANK", "EXTERNAL", "GLOBAL", "SYNCHRONIZED", "SYNC", "RENAMES",
		"TYPE", "LINE", "LINES", "COLUMN", "COL", "SOURCE", "SUM", "NEXT", "GROUP", "RESET":
		return true
	}
	_, isUsage := usages[t.Text]
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
	case clause.Is("SIGN"):
		c.optional("IS")
		return e.parseSign(c, c.next())
	case clause.Is("LEADING") || clause.Is("TRAILING"):
		return e.parseSign(c, clause)
	case clause.Is("REDEFINES"):
		e.Redefines = c.next().Text
	case clause.Is("OCCURS"):
		return e.parseOccurs(c)
	case clause.Is("VALUE") || clause.Is("VALUES"):
		return e.parseValues(c)
	case clause.Is("JUSTIFIED") || clause.Is("JUST"):
		c.optional("RIGHT")
		e.Justified = true
	case clause.Is("BLANK"):
		c.optional("WHEN")
		if !c.optional("ZERO") && !c.optional("ZEROS") && !c.optional("ZEROES") {
			return fmt.Errorf("%s: %s: BLANK WHEN without ZERO", clause.Pos, e.Name)
		}
		e.BlankWhenZero = true
	case clause.Is("IS") && (c.peek().Is("EXTERNAL") || c.peek().Is("GLOBAL")):
	case clause.Is("SYNCHRONIZED") || clause.Is("SYNC"):
		if !c.optional("LEFT") {
			c.optional("RIGHT")
		}
		e.Synchronized = true
	case clause.Is("RENAMES"):
		e.Renames = c.qualified()
		if c.optional("THRU") || c.optional("THROUGH") {
			e.RenamesThru = c.qualified()
		}
	case clause.Is("EXTERNAL"):
		e.External = true
	case clause.Is("GLOBAL"):
		e.Global = true
	case clause.Is("TYPE"):
		return e.parseType(c)
	case clause.Is("LINE") || clause.Is("LINES"):
		c.optional("NUMBER")
		c.optional("NUMBERS")
		if !c.optional("IS") {
			c.optional("ARE")
		}
		e.Line = c.lineNumber()
	case clause.Is("COLUMN") || clause.Is("COL"):
		c.optional("NUMBER")
		c.optional("IS")
		column, err := c.number()
		if err != nil {
			return err
		}
		e.Column = column
	case clause.Is("SOURCE"):
		c.optional("IS")
		e.Source = c.qualified()
	case clause.Is("SUM"):
		c.optional("OF")
		for !c.done() && !isClause(c.peek()) && !c.peek().Is("UPON") {
			e.Sum = append(e.Sum, c.qualified())
		}
		if c.optional("UPON") {
			for !c.done() && !isClause(c.peek()) {
				c.next()
			}
		}
	case clause.Is("NEXT"):
		if err := c.expect("GROUP"); err != nil {
			return err
		}
		c.optional("IS")
		c.lineNumber()
	case clause.Is("GROUP"):
		c.optional("INDICATE")
	case clause.Is("RESET"):
		c.optional("ON")
		c.next()
	default:
		return fmt.Errorf("%s: %s: %s is not supported yet", clause.Pos, e.Name, clause)
	}
	return nil
}

// parseType parses the rest of a report group's TYPE clause: the part of
// the report, in full or by its two letters, and the control item of a
// CONTROL HEADING or FOOTING.
func (e *DataEntry) parseType(c *cursor) error {
	c.optional("IS")
	t := c.next()
	parts := map[string]string{
		"RH": "REPORT HEADING", "PH": "PAGE HEADING", "CH": "CONTROL HEADING", "DE": "DETAIL",
		"CF": "CONTROL FOOTING", "PF": "PAGE FOOTING", "RF": "REPORT FOOTING", "DETAIL": "DETAIL",
	}
	switch {
	case parts[t.Text] != "":
		e.Type = parts[t.Text]
	case t.Is("REPORT") || t.Is("PAGE") || t.Is("CONTROL"):
		part := c.next()
		if !part.Is("HEADING") && !part.Is("FOOTING") {
			return fmt.Errorf("%s: %s: TYPE %s %s, want HEADING or FOOTING", part.Pos, e.Name, t, part)
		}
		e.Type = t.Text + " " + part.Text
	default:
		return fmt.Errorf("%s: %s: TYPE %s is not a part of a report", t.Pos, e.Name, t)
	}
	if strings.HasPrefix(e.Type, "CONTROL") {
		c.optional("IS")
		if c.done() || isClause(c.peek()) {
			return fmt.Errorf("%s: %s: TYPE %s without its control", t.Pos, e.Name, e.Type)
		}
		e.TypeControl = c.qualified()
	}
	return nil
}

// lineNumber reads a line position as written: a number, PLUS and a
// number, or NEXT PAGE, empty when there is none.
func (c *cursor) lineNumber() string {
	switch {
	case c.optional("PLUS"):
		return "PLUS " + c.next().Text
	case c.optional("NEXT"):
		c.optional("PAGE")
		return "NEXT PAGE"
	case !c.done() && !isClause(c.peek()):
		return c.next().Text
	}
	return ""
}

func (e *DataEntry) setUsage(t Token) error {
	usage, ok := usages[t.Text]
	if !ok {
		return fmt.Errorf("%s: %s: USAGE %s is not supported yet", t.Pos, e.Name, t)
	}
	e.Usage = usage
	return nil
}

// parseOccurs parses OCCURS n [TIMES] or OCCURS n TO m [TIMES] DEPENDING
// [ON] x, either followed by ASCENDING or DESCENDING [KEY] [IS] keys, and
// INDEXED [BY] and index names.
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
	for c.peek().Is("ASCENDING") || c.peek().Is("DESCENDING") {
		descending := c.next().Is("DESCENDING")
		c.optional("KEY")
		c.optional("IS")
		for !c.done() && !isClause(c.peek()) {
			e.Keys = append(e.Keys, TableKey{Name: c.next().Text, Descending: descending})
		}
	}
	if c.optional("INDEXED") {
		c.optional("BY")
		for !c.done() && !isClause(c.peek()) {
			e.IndexedBy = append(e.IndexedBy, c.next().Text)
		}
		if len(e.IndexedBy) == 0 {
			return fmt.Errorf("%s: %s: INDEXED BY without an index name", e.Pos, e.Name)
		}
	}
	return nil
}

// parseSign parses the rest of [SIGN [IS]] LEADING|TRAILING [SEPARATE
// [CHARACTER]], from the position.
func (e *DataEntry) parseSign(c *cursor, position Token) error {
	if !position.Is("LEADING") && !position.Is("TRAILING") {
		return fmt.Errorf("%s: %s: SIGN %s, want LEADING or TRAILING", position.Pos, e.Name, position)
	}
	e.Sign = position.Text
	if c.optional("SEPARATE") {
		e.SignSeparate = true
		c.optional("CHARACTER")
	}
	return nil
}

// parseValues reads the literals of a VALUE clause up to the next clause.
func (e *DataEntry) parseValues(c *cursor) error {
	c.optional("IS")
	c.optional("ARE")
	for !c.done() && !isClause(c.peek()) {
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

// literal reads a literal as written: a quoted string, hex digits, a
// number, a figurative constant, or ALL followed by a literal.
func literal(c *cursor) (string, error) {
	t := c.next()
	switch {
	case t.Kind == String:
		return "'" + strings.ReplaceAll(t.Text, "'", "''") + "'", nil
	case t.Kind == Hex, t.Kind == National:
		return t.String(), nil
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
