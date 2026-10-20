package cobol

import (
	"fmt"
	"strings"
)

// Condition is a condition as a tree. Op is AND, OR or NOT, joining Parts;
// a relational operator (=, <, >, <=, >=) between Left and Right; IS, a
// class or sign test of Left, such as NUMERIC or POSITIVE, in Class; NAME,
// a condition name (level 88) in Left; or TRUE, a WHEN of EVALUATE whose
// objects are all ANY.
type Condition struct {
	Op    string
	Parts []*Condition
	Left  *Operand
	Right *Operand
	Class string
}

// String writes the condition in COBOL, parenthesized where AND and OR
// nest, and with NOT before a relation written as in A NOT = B.
func (c *Condition) String() string {
	switch c.Op {
	case "AND", "OR":
		var parts []string
		for _, part := range c.Parts {
			s := part.String()
			if (part.Op == "AND" || part.Op == "OR") && part.Op != c.Op {
				s = "(" + s + ")"
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, " "+c.Op+" ")
	case "NOT":
		inner := c.Parts[0]
		switch inner.Op {
		case "=", "<", ">", "<=", ">=":
			return fmt.Sprintf("%s NOT %s %s", inner.Left.Text, inner.Op, inner.Right.Text)
		case "IS":
			return fmt.Sprintf("%s IS NOT %s", inner.Left.Text, inner.Class)
		case "NAME":
			return "NOT " + inner.Left.Text
		}
		return "NOT (" + inner.String() + ")"
	case "IS":
		return fmt.Sprintf("%s IS %s", c.Left.Text, c.Class)
	case "NAME":
		return c.Left.Text
	case "TRUE":
		return "TRUE"
	}
	return fmt.Sprintf("%s %s %s", c.Left.Text, c.Op, c.Right.Text)
}

// Refs lists the data items and condition names the condition reads.
func (c *Condition) Refs() []string {
	var refs []string
	for _, part := range c.Parts {
		refs = append(refs, part.Refs()...)
	}
	for _, op := range []*Operand{c.Left, c.Right} {
		if op != nil {
			refs = append(refs, op.Refs...)
		}
	}
	return refs
}

// combine joins conditions with AND or OR; a single one stands alone.
func combine(op string, parts []*Condition) *Condition {
	if len(parts) == 1 {
		return parts[0]
	}
	return &Condition{Op: op, Parts: parts}
}

var classes = map[string]bool{
	"NUMERIC": true, "ALPHABETIC": true, "ALPHABETIC-LOWER": true, "ALPHABETIC-UPPER": true,
	"POSITIVE": true, "NEGATIVE": true, "ZERO": true,
}

// condition parses a condition: conditions joined by OR and AND, NOT, and
// parentheses around simple conditions.
func (p *statementParser) condition() (*Condition, error) {
	var last *Condition
	return p.orCondition(&last)
}

// The parsers below pass the last relation along, for abbreviated
// combined conditions: in A = 1 OR 2, the 2 means A = 2.
func (p *statementParser) orCondition(last **Condition) (*Condition, error) {
	return p.joined("OR", last, p.andCondition)
}

func (p *statementParser) andCondition(last **Condition) (*Condition, error) {
	return p.joined("AND", last, p.notCondition)
}

func (p *statementParser) joined(op string, last **Condition, part func(**Condition) (*Condition, error)) (*Condition, error) {
	first, err := part(last)
	if err != nil {
		return nil, err
	}
	parts := []*Condition{first}
	for p.optional(op) {
		next, err := part(last)
		if err != nil {
			return nil, err
		}
		parts = append(parts, next)
	}
	return combine(op, parts), nil
}

func (p *statementParser) notCondition(last **Condition) (*Condition, error) {
	if p.optional("NOT") {
		c, err := p.notCondition(last)
		if err != nil {
			return nil, err
		}
		return &Condition{Op: "NOT", Parts: []*Condition{c}}, nil
	}
	if start := p.i; p.peek().Kind == LParen {
		p.next()
		c, err := p.orCondition(last)
		if err == nil && p.peek().Kind == RParen {
			p.next()
			if t := p.peek(); !t.Is("IS") && !t.Is("NOT") && !p.atRelationalOperator() {
				return c, nil
			}
		}
		// Not a condition in parentheses: an expression in them, as in
		// (A + 1) = B, which simpleCondition reads.
		p.i = start
	}
	return p.simpleCondition(last)
}

// simpleCondition parses a relation, a class or sign test, a condition
// name, or the object of an abbreviated relation, with or without its
// operator: in A = 1 OR 2, the 2 means A = 2, and in A > 1 AND < 9, the
// < 9 means A < 9.
func (p *statementParser) simpleCondition(last **Condition) (*Condition, error) {
	if start := p.i; *last != nil {
		p.optional("IS")
		negated := p.optional("NOT")
		if op, ok := p.relationalOperator(); ok {
			right, err := p.expression()
			if err != nil {
				return nil, err
			}
			c := &Condition{Op: op, Left: (*last).Left, Right: &right}
			*last = c
			if negated {
				c = &Condition{Op: "NOT", Parts: []*Condition{c}}
			}
			return c, nil
		}
		p.i = start
	}
	left, err := p.expression()
	if err != nil {
		return nil, err
	}
	p.optional("IS")
	negated := p.optional("NOT")
	var c *Condition
	if op, ok := p.relationalOperator(); ok {
		right, err := p.expression()
		if err != nil {
			return nil, err
		}
		c = &Condition{Op: op, Left: &left, Right: &right}
		*last = c
	} else if t := p.peek(); t.Kind == Word && classes[t.Text] {
		c = &Condition{Op: "IS", Left: &left, Class: p.next().Text}
	} else if left.Name != "" && !negated && (p.conditions[left.Name] || *last == nil) {
		return &Condition{Op: "NAME", Left: &left}, nil
	} else if *last != nil {
		c = &Condition{Op: (*last).Op, Left: (*last).Left, Right: &left}
	} else {
		return nil, fmt.Errorf("%s: %s is not a condition", left.Pos, left.Text)
	}
	if negated {
		c = &Condition{Op: "NOT", Parts: []*Condition{c}}
	}
	return c, nil
}

// atRelationalOperator reports whether a relational operator comes next.
func (p *statementParser) atRelationalOperator() bool {
	start := p.i
	_, ok := p.relationalOperator()
	p.i = start
	return ok
}

// relationalOperator reads =, <, >, <=, >= or their words: EQUAL TO,
// GREATER THAN, LESS THAN, GREATER THAN OR EQUAL TO, LESS THAN OR EQUAL TO.
func (p *statementParser) relationalOperator() (string, bool) {
	t := p.peek()
	if t.Kind != Word {
		return "", false
	}
	switch t.Text {
	case "=", "<", ">", "<=", ">=":
		p.next()
		return t.Text, true
	case "EQUAL":
		p.next()
		p.optional("TO")
		return "=", true
	case "GREATER", "LESS":
		p.next()
		p.optional("THAN")
		op := map[string]string{"GREATER": ">", "LESS": "<"}[t.Text]
		if p.peek().Is("OR") && p.at(p.i+1).Is("EQUAL") {
			p.i += 2
			p.optional("TO")
			op += "="
		}
		return op, true
	}
	return "", false
}
