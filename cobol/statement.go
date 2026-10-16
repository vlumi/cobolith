package cobol

import (
	"fmt"
	"slices"
	"strings"
)

// Statement is one statement of the procedure division: its verb, the
// data items it reads and writes, the files and procedures it names, the
// condition it tests, and the statements nested in it. EXIT keeps what it
// exits in its verb: EXIT PROGRAM, EXIT PARAGRAPH.
type Statement struct {
	Verb string
	// Text is the statement as written; for one with statements nested in
	// it, its head, before them: IF A = 1, READ ACCT-IN.
	Text       string
	Pos        Pos
	Reads      []Operand
	Writes     []Operand
	Files      []FileRef
	Procedures []ProcedureRef
	// Condition is what an IF tests, or the UNTIL of a PERFORM.
	Condition *Condition
	Branches  []Branch
}

// FileRef is a file a statement names, and the mode it uses it in: OPEN's
// INPUT, OUTPUT, EXTEND or I-O, a SORT's USING files as INPUT and GIVING
// files as OUTPUT, and no mode for the files READ, RETURN and CLOSE name
// and the sort file of a SORT.
type FileRef struct {
	Name string
	Mode string
}

// ProcedureRef names a procedure a statement transfers control to: a
// PERFORM's paragraph or THRU range, a GO TO's target, a SORT's input or
// output procedure.
type ProcedureRef struct {
	Name string
	Thru string
}

// Branch is a list of statements nested in another statement, run under a
// condition: an IF's THEN or ELSE, a WHEN of EVALUATE or SEARCH, WHEN
// OTHER, AT END, NOT AT END, or the body of an inline PERFORM. The
// condition of a WHEN of EVALUATE combines its subjects with its objects:
// EVALUATE A ALSO TRUE WHEN 1 ALSO B > 2 runs its WHEN if A = 1 AND B > 2.
type Branch struct {
	Kind       string
	Condition  *Condition
	Statements []Statement
}

// Operand is a data item, a literal or an arithmetic expression, as
// written: Text is its text, Name the data item it names, if it is one,
// and Refs every data item it reads to give its value, the item itself
// and those in its subscripts.
type Operand struct {
	Text string
	Name string
	Refs []string
	Pos  Pos
}

// Statement verbs that the parser knows.
var verbs = map[string]bool{
	"ACCEPT": true, "ADD": true, "CLOSE": true, "COMPUTE": true, "CONTINUE": true, "DISPLAY": true,
	"DIVIDE": true, "EVALUATE": true, "EXIT": true, "GO": true, "GOBACK": true, "IF": true,
	"INITIALIZE": true, "MOVE": true, "MULTIPLY": true, "OPEN": true, "PERFORM": true, "READ": true,
	"RELEASE": true, "RETURN": true, "SEARCH": true, "SET": true, "SORT": true, "STOP": true,
	"STRING": true, "SUBTRACT": true, "WRITE": true,
}

// keywords are reserved words that end an operand list.
var keywords = map[string]bool{
	"TO": true, "FROM": true, "BY": true, "INTO": true, "GIVING": true, "ROUNDED": true, "ALSO": true,
	"WHEN": true, "ELSE": true, "THEN": true, "AT": true, "NOT": true, "END": true, "INVALID": true,
	"ON": true, "SIZE": true, "OVERFLOW": true, "UNTIL": true, "VARYING": true, "TIMES": true,
	"THRU": true, "THROUGH": true, "DEPENDING": true, "DELIMITED": true, "WITH": true, "POINTER": true,
	"UPON": true, "ADVANCING": true, "AFTER": true, "BEFORE": true, "INPUT": true, "OUTPUT": true,
	"EXTEND": true, "I-O": true, "USING": true, "PROCEDURE": true, "ASCENDING": true, "DESCENDING": true,
	"KEY": true, "DUPLICATES": true, "IS": true, "OTHER": true, "TRUE": true, "FALSE": true, "AND": true,
	"OR": true, "UP": true, "DOWN": true, "RECORD": true, "NEXT": true, "REMAINDER": true, "OF": true,
	"IN": true, "THAN": true, "EQUAL": true, "GREATER": true, "LESS": true, "ANY": true, "RUN": true,
	"PARAGRAPH": true, "PROGRAM": true, "SECTION": true, "FUNCTION": true, "CORRESPONDING": true,
	"CORR": true, "LINE": true, "LINES": true, "PAGE": true, "NUMERIC": true, "ALPHABETIC": true,
	"POSITIVE": true, "NEGATIVE": true, "TEST": true,
	"=": true, "<": true, ">": true, "<=": true, ">=": true, "+": true, "-": true, "*": true, "/": true, "**": true,
}

func isTerminator(t Token) bool {
	return t.Kind == Word && strings.HasPrefix(t.Text, "END-")
}

// statementParser parses the statements of a paragraph body. Starts are
// the token indexes where the statements being parsed begin.
type statementParser struct {
	tokens     []Token
	i          int
	procedures map[string]bool
	starts     []int
}

// ParseStatements parses a paragraph body into statements, sentence by
// sentence. Procedures are the program's paragraph and section names,
// which tell PERFORM name from an inline PERFORM. A verb outside the
// subset Copal knows is an error.
func ParseStatements(body []Token, procedures map[string]bool) ([]Statement, error) {
	p := &statementParser{tokens: body, procedures: procedures}
	var all []Statement
	for !p.done() {
		statements, err := p.statements()
		if err != nil {
			return nil, err
		}
		all = append(all, statements...)
		switch t := p.peek(); {
		case p.done():
		case t.Kind == Period:
			p.i++
		default:
			return nil, fmt.Errorf("%s: unexpected %s", t.Pos, t)
		}
	}
	return all, nil
}

func (p *statementParser) done() bool  { return p.i >= len(p.tokens) }
func (p *statementParser) peek() Token { return p.at(p.i) }

func (p *statementParser) at(i int) Token {
	if i >= len(p.tokens) {
		return Token{Kind: Period}
	}
	return p.tokens[i]
}

func (p *statementParser) next() Token {
	t := p.peek()
	p.i++
	return t
}

// optional consumes the word if it comes next.
func (p *statementParser) optional(words ...string) bool {
	if t := p.peek(); t.Kind == Word && slices.Contains(words, t.Text) {
		p.i++
		return true
	}
	return false
}

func (p *statementParser) expect(word string) error {
	if !p.optional(word) {
		t := p.peek()
		return fmt.Errorf("%s: expected %s, found %s", t.Pos, word, t)
	}
	return nil
}

// statements parses statements until a token that does not start one.
func (p *statementParser) statements() ([]Statement, error) {
	var statements []Statement
	for t := p.peek(); t.Kind == Word && verbs[t.Text]; t = p.peek() {
		s, err := p.statement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, s)
	}
	if t := p.peek(); t.Kind == Word && !keywords[t.Text] && !isTerminator(t) {
		return nil, fmt.Errorf("%s: %s is not supported yet", t.Pos, t)
	}
	return statements, nil
}

func (p *statementParser) statement() (Statement, error) {
	start := p.i
	verb := p.next()
	s := Statement{Verb: verb.Text, Pos: verb.Pos}
	p.starts = append(p.starts, start)
	defer func() { p.starts = p.starts[:len(p.starts)-1] }()
	var err error
	switch verb.Text {
	case "MOVE":
		err = p.move(&s)
	case "ADD", "SUBTRACT", "MULTIPLY", "DIVIDE":
		err = p.arithmetic(&s)
	case "COMPUTE":
		err = p.compute(&s)
	case "SET":
		err = p.set(&s)
	case "INITIALIZE":
		s.Writes, err = p.operands()
	case "STRING":
		err = p.stringStatement(&s)
	case "DISPLAY", "ACCEPT":
		err = p.display(&s)
	case "IF":
		err = p.ifStatement(&s)
	case "EVALUATE":
		err = p.evaluate(&s)
	case "PERFORM":
		err = p.perform(&s)
	case "GO":
		err = p.goTo(&s)
	case "SEARCH":
		err = p.search(&s)
	case "OPEN", "CLOSE":
		p.files(&s)
	case "READ", "RETURN":
		err = p.read(&s)
	case "WRITE", "RELEASE":
		err = p.write(&s)
	case "SORT":
		err = p.sort(&s)
	case "STOP":
		err = p.expect("RUN")
	case "EXIT":
		if t := p.peek(); p.optional("PARAGRAPH", "PROGRAM", "SECTION", "PERFORM") {
			s.Verb += " " + t.Text
		}
	case "CONTINUE", "GOBACK":
	}
	if err != nil {
		return Statement{}, fmt.Errorf("%s: %s: %w", verb.Pos, verb.Text, err)
	}
	if s.Text == "" {
		p.head(&s)
	}
	return s, nil
}

// head sets a statement's text to its tokens so far: called before the
// statements nested in it, or at its end.
func (p *statementParser) head(s *Statement) {
	if s.Text == "" {
		s.Text = joinTokens(p.tokens[p.starts[len(p.starts)-1]:p.i])
	}
}

// joinTokens writes tokens as COBOL text, with parentheses snug.
func joinTokens(tokens []Token) string {
	var b strings.Builder
	for i, t := range tokens {
		snug := i == 0 || t.Kind == RParen || tokens[i-1].Kind == LParen ||
			t.Kind == LParen && tokens[i-1].Kind == Word && !keywords[tokens[i-1].Text] && !arithmeticOperators[tokens[i-1].Text]
		if !snug {
			b.WriteByte(' ')
		}
		b.WriteString(t.String())
	}
	return b.String()
}

// operand parses a data item with its subscripts or reference
// modification, a literal, a figurative constant, ALL and a literal, or a
// function call.
func (p *statementParser) operand() (Operand, error) {
	t := p.next()
	op := Operand{Text: t.String(), Pos: t.Pos}
	switch {
	case t.Kind == String:
		return op, nil
	case t.Kind != Word || verbs[t.Text] || keywords[t.Text] && !t.Is("FUNCTION"):
		return Operand{}, fmt.Errorf("expected an operand, found %s", t)
	case t.Is("ALL"):
		rest, err := p.operand()
		op.Text += " " + rest.Text
		return op, err
	case isNumber(t.Text) || figurative[t.Text]:
		return op, nil
	case t.Is("FUNCTION"):
		name := p.next()
		op.Text += " " + name.Text
		if p.peek().Kind == LParen {
			return p.arguments(op)
		}
		return op, nil
	}
	op.Name = t.Text
	op.Refs = []string{t.Text}
	for p.optional("OF", "IN") {
		op.Text += " OF " + p.next().Text
	}
	for p.peek().Kind == LParen {
		var err error
		if op, err = p.arguments(op); err != nil {
			return Operand{}, err
		}
	}
	return op, nil
}

// arguments parses the parenthesized subscripts, reference modification
// or function arguments after an operand, adding the data items they read.
func (p *statementParser) arguments(op Operand) (Operand, error) {
	p.next()
	var texts []string
	for p.peek().Kind != RParen {
		if p.done() {
			return Operand{}, fmt.Errorf("unclosed parenthesis after %s", op.Text)
		}
		arg, err := p.expression()
		if err != nil {
			return Operand{}, err
		}
		texts = append(texts, arg.Text)
		op.Refs = append(op.Refs, arg.Refs...)
	}
	p.next()
	op.Text += "(" + strings.Join(texts, " ") + ")"
	return op, nil
}

var arithmeticOperators = map[string]bool{"+": true, "-": true, "*": true, "/": true, "**": true}

// expression parses an arithmetic expression: operands joined by
// operators, with parentheses.
func (p *statementParser) expression() (Operand, error) {
	var texts []string
	var refs []string
	pos := p.peek().Pos
	for {
		t := p.peek()
		switch {
		case t.Kind == LParen:
			p.next()
			inner, err := p.expression()
			if err != nil {
				return Operand{}, err
			}
			if p.next().Kind != RParen {
				return Operand{}, fmt.Errorf("%s: unclosed parenthesis", t.Pos)
			}
			texts = append(texts, "("+inner.Text+")")
			refs = append(refs, inner.Refs...)
		case t.Kind == Word && arithmeticOperators[t.Text] && len(texts) == 0:
			p.next()
			texts = append(texts, t.Text)
			continue
		case t.Kind == Word && strings.Contains(t.Text, ":"):
			p.next()
			texts = append(texts, t.Text)
			for _, part := range strings.Split(t.Text, ":") {
				if part != "" && !isNumber(part) {
					refs = append(refs, part)
				}
			}
		default:
			op, err := p.operand()
			if err != nil {
				return Operand{}, err
			}
			texts = append(texts, op.Text)
			refs = append(refs, op.Refs...)
		}
		if t := p.peek(); t.Kind != Word || !arithmeticOperators[t.Text] {
			break
		}
		texts = append(texts, p.next().Text)
	}
	op := Operand{Text: strings.Join(texts, " "), Refs: refs, Pos: pos}
	if len(texts) == 1 && len(refs) > 0 && refs[0] == strings.SplitN(texts[0], "(", 2)[0] {
		op.Name = refs[0]
	}
	return op, nil
}

// operands parses operands up to a keyword, verb, terminator or period,
// skipping ROUNDED after any of them.
func (p *statementParser) operands() ([]Operand, error) {
	var ops []Operand
	for {
		p.optional("ROUNDED")
		t := p.peek()
		if (t.Kind != Word && t.Kind != String) || t.Kind == Word && (verbs[t.Text] || keywords[t.Text] && !t.Is("FUNCTION") || isTerminator(t)) {
			return ops, nil
		}
		op, err := p.operand()
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
}

func (p *statementParser) move(s *Statement) error {
	p.optional("CORRESPONDING", "CORR")
	source, err := p.operand()
	if err != nil {
		return err
	}
	s.Reads = []Operand{source}
	if err := p.expect("TO"); err != nil {
		return err
	}
	s.Writes, err = p.operands()
	return err
}

// arithmetic parses ADD, SUBTRACT, MULTIPLY and DIVIDE. Without GIVING,
// the operands after TO, FROM, BY or INTO are both read and written; with
// it, they are read and the GIVING operands written.
func (p *statementParser) arithmetic(s *Statement) error {
	reads, err := p.operands()
	if err != nil {
		return err
	}
	var updated []Operand
	if p.optional("TO", "FROM", "BY", "INTO") {
		if updated, err = p.operands(); err != nil {
			return err
		}
	}
	s.Reads = append(reads, updated...)
	if p.optional("GIVING") {
		if s.Writes, err = p.operands(); err != nil {
			return err
		}
	} else {
		s.Writes = updated
	}
	if p.optional("REMAINDER") {
		remainder, err := p.operands()
		if err != nil {
			return err
		}
		s.Writes = append(s.Writes, remainder...)
	}
	return p.conditionalPhrases(s, "END-"+s.Verb)
}

func (p *statementParser) compute(s *Statement) error {
	var err error
	if s.Writes, err = p.operands(); err != nil {
		return err
	}
	if !p.optional("=", "EQUAL") {
		return fmt.Errorf("expected =")
	}
	value, err := p.expression()
	if err != nil {
		return err
	}
	s.Reads = []Operand{value}
	return p.conditionalPhrases(s, "END-COMPUTE")
}

// set parses SET x TO value, SET condition TO TRUE and SET index UP BY n.
func (p *statementParser) set(s *Statement) error {
	var err error
	if s.Writes, err = p.operands(); err != nil {
		return err
	}
	if p.optional("UP", "DOWN") {
		if err := p.expect("BY"); err != nil {
			return err
		}
		s.Reads = append([]Operand{}, s.Writes...)
	} else if err := p.expect("TO"); err != nil {
		return err
	}
	if p.optional("TRUE", "FALSE") {
		return nil
	}
	value, err := p.operand()
	s.Reads = append(s.Reads, value)
	return err
}

func (p *statementParser) stringStatement(s *Statement) error {
	for !p.optional("INTO") {
		sources, err := p.operands()
		if err != nil {
			return err
		}
		s.Reads = append(s.Reads, sources...)
		if !p.optional("DELIMITED") {
			return fmt.Errorf("expected DELIMITED or INTO")
		}
		p.optional("BY")
		if !p.optional("SIZE") {
			delimiter, err := p.operand()
			if err != nil {
				return err
			}
			s.Reads = append(s.Reads, delimiter)
		}
	}
	target, err := p.operand()
	if err != nil {
		return err
	}
	s.Writes = []Operand{target}
	if p.optional("WITH") || p.optional("POINTER") {
		p.optional("POINTER")
		pointer, err := p.operand()
		if err != nil {
			return err
		}
		s.Reads = append(s.Reads, pointer)
		s.Writes = append(s.Writes, pointer)
	}
	return p.conditionalPhrases(s, "END-STRING")
}

func (p *statementParser) display(s *Statement) error {
	ops, err := p.operands()
	if s.Verb == "ACCEPT" {
		s.Writes = ops
	} else {
		s.Reads = ops
	}
	if p.optional("UPON", "FROM") {
		p.next()
	}
	if p.optional("WITH") {
		p.optional("NO")
		p.optional("ADVANCING")
	}
	return err
}

func (p *statementParser) ifStatement(s *Statement) error {
	condition, err := p.condition()
	if err != nil {
		return err
	}
	s.Condition = condition
	p.head(s)
	p.optional("THEN")
	then, err := p.statements()
	if err != nil {
		return err
	}
	s.Branches = []Branch{{Kind: "THEN", Statements: then}}
	if p.optional("ELSE") {
		otherwise, err := p.statements()
		if err != nil {
			return err
		}
		s.Branches = append(s.Branches, Branch{Kind: "ELSE", Statements: otherwise})
	}
	p.optional("END-IF")
	return nil
}

// evaluate parses EVALUATE, turning each WHEN into the condition under
// which it runs: its objects matched against the subjects, ALSO as AND,
// stacked WHENs as OR.
func (p *statementParser) evaluate(s *Statement) error {
	var subjects []*Operand
	for {
		if p.optional("TRUE") {
			subjects = append(subjects, nil)
		} else {
			subject, err := p.expression()
			if err != nil {
				return err
			}
			s.Reads = append(s.Reads, subject)
			subjects = append(subjects, &subject)
		}
		if !p.optional("ALSO") {
			break
		}
	}
	p.head(s)
	for p.peek().Is("WHEN") {
		var alternatives []*Condition
		kind := "WHEN"
		for p.optional("WHEN") {
			if p.optional("OTHER") {
				kind = "OTHER"
				continue
			}
			c, err := p.whenObjects(subjects)
			if err != nil {
				return err
			}
			alternatives = append(alternatives, c)
		}
		statements, err := p.statements()
		if err != nil {
			return err
		}
		b := Branch{Kind: kind, Statements: statements}
		if kind == "WHEN" {
			b.Condition = combine("OR", alternatives)
		}
		s.Branches = append(s.Branches, b)
	}
	p.optional("END-EVALUATE")
	return nil
}

// whenObjects parses one WHEN's objects, one per subject, into a condition.
func (p *statementParser) whenObjects(subjects []*Operand) (*Condition, error) {
	var parts []*Condition
	for i, subject := range subjects {
		if i > 0 {
			if err := p.expect("ALSO"); err != nil {
				return nil, err
			}
		}
		if p.optional("ANY") {
			continue
		}
		if subject == nil {
			c, err := p.condition()
			if err != nil {
				return nil, err
			}
			parts = append(parts, c)
			continue
		}
		negated := p.optional("NOT")
		value, err := p.expression()
		if err != nil {
			return nil, err
		}
		c := &Condition{Op: "=", Left: subject, Right: &value}
		if p.optional("THRU", "THROUGH") {
			upper, err := p.expression()
			if err != nil {
				return nil, err
			}
			c = combine("AND", []*Condition{
				{Op: ">=", Left: subject, Right: &value},
				{Op: "<=", Left: subject, Right: &upper},
			})
		}
		if negated {
			c = &Condition{Op: "NOT", Parts: []*Condition{c}}
		}
		parts = append(parts, c)
	}
	if len(parts) == 0 {
		return &Condition{Op: "TRUE"}, nil
	}
	return combine("AND", parts), nil
}

// perform parses PERFORM of a procedure or THRU range, and the inline
// PERFORM with its body, either with TIMES, UNTIL or VARYING.
func (p *statementParser) perform(s *Statement) error {
	t := p.peek()
	outOfLine := t.Kind == Word && p.procedures[t.Text]
	if outOfLine {
		ref := ProcedureRef{Name: p.next().Text}
		if p.optional("THRU", "THROUGH") {
			ref.Thru = p.next().Text
		}
		s.Procedures = []ProcedureRef{ref}
	}
	if err := p.performPhrase(s); err != nil {
		return err
	}
	p.head(s)
	if outOfLine {
		return nil
	}
	body, err := p.statements()
	if err != nil {
		return err
	}
	s.Branches = []Branch{{Kind: "BODY", Statements: body}}
	return p.expect("END-PERFORM")
}

func (p *statementParser) performPhrase(s *Statement) error {
	if p.optional("WITH") {
		if err := p.expect("TEST"); err != nil {
			return err
		}
		p.optional("BEFORE", "AFTER")
	}
	switch t := p.peek(); {
	case p.at(p.i+1).Is("TIMES") && !t.Is("UNTIL") && !t.Is("VARYING"):
		count, err := p.operand()
		if err != nil {
			return err
		}
		p.next()
		s.Reads = append(s.Reads, count)
	case p.optional("VARYING"):
		counter, err := p.operand()
		if err != nil {
			return err
		}
		s.Writes = append(s.Writes, counter)
		for _, word := range []string{"FROM", "BY"} {
			if err := p.expect(word); err != nil {
				return err
			}
			value, err := p.operand()
			if err != nil {
				return err
			}
			s.Reads = append(s.Reads, value)
		}
		return p.until(s)
	case t.Is("UNTIL"):
		return p.until(s)
	}
	return nil
}

func (p *statementParser) until(s *Statement) error {
	if err := p.expect("UNTIL"); err != nil {
		return err
	}
	c, err := p.condition()
	s.Condition = c
	return err
}

func (p *statementParser) goTo(s *Statement) error {
	p.optional("TO")
	for t := p.peek(); t.Kind == Word && p.procedures[t.Text]; t = p.peek() {
		s.Procedures = append(s.Procedures, ProcedureRef{Name: p.next().Text})
	}
	if len(s.Procedures) == 0 {
		return fmt.Errorf("GO TO names no paragraph")
	}
	if p.optional("DEPENDING") {
		p.optional("ON")
		selector, err := p.operand()
		if err != nil {
			return err
		}
		s.Reads = append(s.Reads, selector)
	}
	return nil
}

func (p *statementParser) search(s *Statement) error {
	p.optional("ALL")
	table, err := p.operand()
	if err != nil {
		return err
	}
	s.Reads = append(s.Reads, table)
	if p.optional("VARYING") {
		index, err := p.operand()
		if err != nil {
			return err
		}
		s.Writes = append(s.Writes, index)
	}
	p.head(s)
	if err := p.conditionalPhrases(s, ""); err != nil {
		return err
	}
	for p.optional("WHEN") {
		c, err := p.condition()
		if err != nil {
			return err
		}
		statements, err := p.statements()
		if err != nil {
			return err
		}
		s.Branches = append(s.Branches, Branch{Kind: "WHEN", Condition: c, Statements: statements})
	}
	p.optional("END-SEARCH")
	return nil
}

// files parses OPEN and CLOSE: the files, with OPEN's modes.
func (p *statementParser) files(s *Statement) {
	mode := ""
	for {
		t := p.peek()
		switch {
		case t.Kind == Word && slices.Contains([]string{"INPUT", "OUTPUT", "EXTEND", "I-O"}, t.Text):
			mode = p.next().Text
		case t.Kind == Word && !verbs[t.Text] && !keywords[t.Text] && !isTerminator(t):
			s.Files = append(s.Files, FileRef{Name: p.next().Text, Mode: mode})
		default:
			return
		}
	}
}

func (p *statementParser) read(s *Statement) error {
	s.Files = []FileRef{{Name: p.next().Text}}
	p.optional("NEXT")
	p.optional("RECORD")
	if p.optional("INTO") {
		into, err := p.operand()
		if err != nil {
			return err
		}
		s.Writes = append(s.Writes, into)
	}
	return p.conditionalPhrases(s, "END-"+s.Verb)
}

func (p *statementParser) write(s *Statement) error {
	record, err := p.operand()
	if err != nil {
		return err
	}
	s.Writes = []Operand{record}
	if p.optional("FROM") {
		from, err := p.operand()
		if err != nil {
			return err
		}
		s.Reads = []Operand{from}
	}
	if p.optional("AFTER", "BEFORE") {
		p.optional("ADVANCING")
		lines, err := p.operand()
		if err != nil {
			return err
		}
		s.Reads = append(s.Reads, lines)
		p.optional("LINE", "LINES")
	}
	if s.Verb == "RELEASE" {
		return nil
	}
	return p.conditionalPhrases(s, "END-WRITE")
}

// sort parses SORT file ON ASCENDING KEY keys, then USING files or an
// input procedure, then GIVING files or an output procedure.
func (p *statementParser) sort(s *Statement) error {
	s.Files = []FileRef{{Name: p.next().Text}}
	for p.optional("ON") || p.peek().Is("ASCENDING") || p.peek().Is("DESCENDING") {
		p.optional("ASCENDING", "DESCENDING")
		p.optional("KEY")
		keys, err := p.operands()
		if err != nil {
			return err
		}
		s.Reads = append(s.Reads, keys...)
	}
	if p.optional("WITH") {
		p.optional("DUPLICATES")
	}
	for _, phrase := range []struct{ procedure, files, mode string }{{"INPUT", "USING", "INPUT"}, {"OUTPUT", "GIVING", "OUTPUT"}} {
		switch {
		case p.optional(phrase.procedure):
			if err := p.expect("PROCEDURE"); err != nil {
				return err
			}
			p.optional("IS")
			ref := ProcedureRef{Name: p.next().Text}
			if p.optional("THRU", "THROUGH") {
				ref.Thru = p.next().Text
			}
			s.Procedures = append(s.Procedures, ref)
		case p.optional(phrase.files):
			for t := p.peek(); t.Kind == Word && !verbs[t.Text] && !keywords[t.Text]; t = p.peek() {
				s.Files = append(s.Files, FileRef{Name: p.next().Text, Mode: phrase.mode})
			}
		}
	}
	return nil
}

// conditionalPhrases parses AT END, INVALID KEY, ON SIZE ERROR and ON
// OVERFLOW, and their NOT forms, each as a branch, then the statement's
// scope terminator if one is given and present.
func (p *statementParser) conditionalPhrases(s *Statement, terminator string) error {
	p.head(s)
	for {
		start := p.i
		negated := p.optional("NOT")
		var kind string
		switch {
		case p.optional("AT"):
			if err := p.expect("END"); err != nil {
				return err
			}
			kind = "AT END"
		case p.optional("INVALID"):
			p.optional("KEY")
			kind = "INVALID KEY"
		case p.optional("ON") || p.peek().Is("SIZE") || p.peek().Is("OVERFLOW"):
			if p.optional("OVERFLOW") {
				kind = "ON OVERFLOW"
			} else {
				if err := p.expect("SIZE"); err != nil {
					return err
				}
				if err := p.expect("ERROR"); err != nil {
					return err
				}
				kind = "ON SIZE ERROR"
			}
		default:
			p.i = start
			if terminator != "" {
				p.optional(terminator)
			}
			return nil
		}
		if negated {
			kind = "NOT " + kind
		}
		statements, err := p.statements()
		if err != nil {
			return err
		}
		s.Branches = append(s.Branches, Branch{Kind: kind, Statements: statements})
	}
}

// Walk visits statements and the statements nested in them, depth first,
// in the order they are written.
func Walk(statements []Statement, visit func(Statement)) {
	for _, s := range statements {
		visit(s)
		for _, b := range s.Branches {
			Walk(b.Statements, visit)
		}
	}
}
