package cobol

import (
	"fmt"
	"slices"
	"strconv"
)

// Program is a parsed COBOL program: what the analysis needs from each
// division.
type Program struct {
	ID             string
	Path           string
	Comment        []string // the comment lines before the program, its header banner
	Identification []IdentificationEntry
	Files          []FileControl
	FileSection    []FileDescription
	WorkingStorage []*DataEntry
	// Linkage are the items a calling program passes, which the procedure
	// division names in Using.
	Linkage    []*DataEntry
	Using      []string
	Paragraphs []Paragraph
	// Declaratives are the sections of the DECLARATIVES, whose
	// paragraphs come first among the paragraphs.
	Declaratives []Declarative
	// Contains are the programs nested in this one, which only it, and
	// the programs it contains, can call.
	Contains []*Program
}

// Declarative is a section of the DECLARATIVES: what its USE sentence
// says it is for, the files named or every file opened in one of the
// modes named. The runtime runs it when an input or output statement on
// one of them fails, and control goes on after that statement.
type Declarative struct {
	Section string
	Files   []string
	Modes   []string
	Pos     Pos
}

// IdentificationEntry is an identification paragraph with a comment entry,
// such as AUTHOR or DATE-WRITTEN.
type IdentificationEntry struct {
	Paragraph string
	Text      string
}

// FileControl is a SELECT entry: a file and the external name it is
// assigned to.
type FileControl struct {
	Name         string
	Assign       string
	Organization string
	Access       string // SEQUENTIAL, RANDOM or DYNAMIC
	RecordKey    string // an indexed file's key
	RelativeKey  string // the item that holds a relative file's record number
	Status       string
	Pos          Pos
}

// FileDescription is an FD or SD entry with the records under it, and the
// record lengths its RECORD clause gives, if it has one: the same for
// fixed-length records, and for variable-length ones the shortest, the
// longest, and the item that holds a record's length.
type FileDescription struct {
	Kind                 string // FD or SD
	Name                 string
	MinLength, MaxLength int
	LengthIn             string
	Entries              []*DataEntry
	Pos                  Pos
}

// Paragraph is a paragraph of the procedure division: the tokens of its
// body and the statements they make. Paragraphs before the first section
// header have no section. A declarative paragraph is one of the
// DECLARATIVES, which nothing performs or falls into.
type Paragraph struct {
	Name        string
	Section     string
	Declarative bool
	Body        []Token
	Statements  []Statement
	Comment     []string
	Pos         Pos
}

// ReadProgram reads and parses a program, expanding its copybooks.
func ReadProgram(path string, copybookDirs []string) (*Program, error) {
	tokens, err := Read(path, copybookDirs)
	if err != nil {
		return nil, err
	}
	return Parse(path, tokens)
}

// Parse parses the tokens of a program, and of the programs nested in it.
func Parse(path string, tokens []Token) (*Program, error) {
	own, nested, err := splitNested(tokens)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	divisions, err := splitDivisions(own)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	p := &Program{Path: path}
	for _, n := range nested {
		inner, err := Parse(path, n)
		if err != nil {
			return nil, err
		}
		p.Contains = append(p.Contains, inner)
	}
	if len(tokens) > 0 {
		p.Comment = tokens[0].Comment
	}
	steps := []struct {
		division string
		parse    func([]Token) error
	}{
		{"IDENTIFICATION", p.parseIdentification},
		{"ENVIRONMENT", p.parseEnvironment},
		{"DATA", p.parseData},
		{"PROCEDURE", p.parseProcedure},
	}
	for _, step := range steps {
		if err := step.parse(divisions[step.division]); err != nil {
			return nil, err
		}
	}
	if p.ID == "" {
		return nil, fmt.Errorf("%s: no PROGRAM-ID", path)
	}
	return p, nil
}

// splitNested separates a program's own tokens from those of the programs
// nested in it, each from its IDENTIFICATION DIVISION through the END
// PROGRAM that names it. The program's own END PROGRAM, if it has one,
// ends it.
func splitNested(tokens []Token) (own []Token, nested [][]Token, err error) {
	var open []string // the nested programs not yet ended, innermost last
	start := 0
	for i := 0; i < len(tokens); i++ {
		switch {
		case i > 0 && isDivision(tokens, i, "IDENTIFICATION", "ID"):
			if len(open) == 0 {
				start = i
			}
			open = append(open, programID(tokens, i))
		case tokens[i].Is("END") && i+1 < len(tokens) && tokens[i+1].Is("PROGRAM"):
			end := skipPast(tokens, i, Period)
			name := ""
			if i+2 < end {
				name = tokens[i+2].Text
			}
			if len(open) == 0 {
				if end+1 < len(tokens) {
					return nil, nil, fmt.Errorf("%s: %s after END PROGRAM", tokens[end+1].Pos, tokens[end+1])
				}
				return own, nested, nil
			}
			if innermost := open[len(open)-1]; name != innermost {
				return nil, nil, fmt.Errorf("%s: END PROGRAM %s where %s has not ended", tokens[i].Pos, name, innermost)
			}
			if open = open[:len(open)-1]; len(open) == 0 {
				nested = append(nested, tokens[start:end+1])
			}
			i = end
		case len(open) == 0:
			own = append(own, tokens[i])
		}
	}
	if len(open) > 0 {
		return nil, nil, fmt.Errorf("%s: no END PROGRAM %s", tokens[start].Pos, open[len(open)-1])
	}
	return own, nested, nil
}

// programID is the PROGRAM-ID of the program whose IDENTIFICATION
// DIVISION starts at i.
func programID(tokens []Token, i int) string {
	for j := i; j+2 < len(tokens); j++ {
		if tokens[j].Is("PROGRAM-ID") {
			return tokens[j+2].Text
		}
	}
	return ""
}

// isDivision reports whether a division header, by one of its names,
// starts at i.
func isDivision(tokens []Token, i int, names ...string) bool {
	return i+1 < len(tokens) && tokens[i].Kind == Word && slices.Contains(names, tokens[i].Text) && tokens[i+1].Is("DIVISION")
}

// splitDivisions returns the tokens of each division after its header,
// those of the procedure division led by its USING phrase, if any.
func splitDivisions(tokens []Token) (map[string][]Token, error) {
	divisions := map[string][]Token{}
	current := ""
	for i := 0; i < len(tokens); i++ {
		if i+1 < len(tokens) && tokens[i].Kind == Word && tokens[i+1].Is("DIVISION") {
			current = tokens[i].Text
			if current == "ID" {
				current = "IDENTIFICATION"
			}
			if _, seen := divisions[current]; seen {
				return nil, fmt.Errorf("%s: second %s DIVISION", tokens[i].Pos, current)
			}
			divisions[current] = nil
			end := skipPast(tokens, i, Period)
			if current == "PROCEDURE" && i+2 < end && tokens[i+2].Is("USING") {
				divisions[current] = append(divisions[current], tokens[i+2:end]...)
			}
			i = end
			continue
		}
		if current == "" {
			return nil, fmt.Errorf("%s: %s before IDENTIFICATION DIVISION", tokens[i].Pos, tokens[i])
		}
		divisions[current] = append(divisions[current], tokens[i])
	}
	return divisions, nil
}

// skipPast returns the index of the first token of the given kind at or
// after i, or the last index if there is none.
func skipPast(tokens []Token, i int, kind Kind) int {
	for i < len(tokens) && tokens[i].Kind != kind {
		i++
	}
	return min(i, len(tokens)-1)
}

// entries splits tokens into period-terminated entries, without the periods.
func entries(tokens []Token) [][]Token {
	var out [][]Token
	start := 0
	for i, t := range tokens {
		if t.Kind == Period {
			if i > start {
				out = append(out, tokens[start:i])
			}
			start = i + 1
		}
	}
	if start < len(tokens) {
		out = append(out, tokens[start:])
	}
	return out
}

func (p *Program) parseIdentification(tokens []Token) error {
	for i, t := range tokens {
		switch {
		case t.Is("PROGRAM-ID") && p.ID == "":
			for _, name := range tokens[i+1:] {
				if name.Kind == Word || name.Kind == String {
					p.ID = name.Text
					break
				}
			}
		case t.Kind == CommentEntry:
			p.Identification = append(p.Identification, IdentificationEntry{Paragraph: tokens[i-2].Text, Text: t.Text})
		}
	}
	return nil
}

func (p *Program) parseEnvironment(tokens []Token) error {
	for _, entry := range entries(tokens) {
		if entry[0].Is("SELECT") {
			file, err := parseSelect(entry)
			if err != nil {
				return err
			}
			p.Files = append(p.Files, file)
		}
	}
	return nil
}

// parseSelect parses SELECT [OPTIONAL] name ASSIGN [TO] target and the
// ORGANIZATION, ACCESS MODE, RECORD KEY and FILE STATUS clauses.
func parseSelect(entry []Token) (FileControl, error) {
	c := &cursor{tokens: entry}
	c.next()
	c.optional("OPTIONAL")
	file := FileControl{Name: c.next().Text, Organization: "SEQUENTIAL", Access: "SEQUENTIAL", Pos: entry[0].Pos}
	for !c.done() {
		clause := c.next()
		switch {
		case clause.Is("ASSIGN"):
			c.optional("TO")
			file.Assign = c.next().Text
		case clause.Is("ORGANIZATION"):
			c.optional("IS")
			file.Organization = c.next().Text
			if file.Organization == "LINE" || file.Organization == "RECORD" {
				file.Organization += " " + c.next().Text
			}
		case clause.Is("ACCESS"):
			c.optional("MODE")
			c.optional("IS")
			file.Access = c.next().Text
		case clause.Is("RECORD"):
			c.optional("KEY")
			c.optional("IS")
			file.RecordKey = c.next().Text
		case clause.Is("RELATIVE"):
			c.optional("KEY")
			c.optional("IS")
			file.RelativeKey = c.next().Text
		case clause.Is("FILE"):
			if err := c.expect("STATUS"); err != nil {
				return FileControl{}, err
			}
			c.optional("IS")
			file.Status = c.next().Text
		default:
			return FileControl{}, fmt.Errorf("%s: SELECT %s: %s is not supported yet", clause.Pos, file.Name, clause)
		}
	}
	return file, nil
}

func (p *Program) parseData(tokens []Token) error {
	section := ""
	for _, entry := range entries(tokens) {
		switch {
		case len(entry) == 2 && entry[1].Is("SECTION"):
			section = entry[0].Text
		case entry[0].Is("FD") || entry[0].Is("SD"):
			if len(entry) < 2 {
				return fmt.Errorf("%s: %s without a file name", entry[0].Pos, entry[0])
			}
			if section != "FILE" {
				return fmt.Errorf("%s: %s outside the FILE SECTION", entry[0].Pos, entry[0])
			}
			fd, err := parseFileDescription(entry)
			if err != nil {
				return err
			}
			p.FileSection = append(p.FileSection, fd)
		default:
			if err := p.addDataEntry(section, entry); err != nil {
				return err
			}
		}
	}
	return nil
}

// parseFileDescription parses FD or SD name and its RECORD clause:
// RECORD [CONTAINS] n [CHARACTERS], RECORD [CONTAINS] n TO m [CHARACTERS],
// or RECORD [IS] VARYING [IN] [SIZE] [FROM] n TO m [CHARACTERS] DEPENDING
// [ON] item.
func parseFileDescription(entry []Token) (FileDescription, error) {
	fd := FileDescription{Kind: entry[0].Text, Name: entry[1].Text, Pos: entry[0].Pos}
	c := &cursor{tokens: entry[2:]}
	for !c.done() {
		clause := c.next()
		if !clause.Is("RECORD") {
			return FileDescription{}, fmt.Errorf("%s: %s %s: %s is not supported yet", clause.Pos, fd.Kind, fd.Name, clause)
		}
		for _, word := range []string{"CONTAINS", "IS", "VARYING", "IN", "SIZE", "FROM"} {
			c.optional(word)
		}
		var err error
		if fd.MinLength, err = c.number(); err != nil {
			return FileDescription{}, err
		}
		fd.MaxLength = fd.MinLength
		if c.optional("TO") {
			if fd.MaxLength, err = c.number(); err != nil {
				return FileDescription{}, err
			}
		}
		c.optional("CHARACTERS")
		if c.optional("DEPENDING") {
			c.optional("ON")
			fd.LengthIn = c.next().Text
		}
	}
	return fd, nil
}

func (p *Program) addDataEntry(section string, tokens []Token) error {
	entry, err := ParseDataEntry(tokens)
	if err != nil {
		return err
	}
	switch section {
	case "FILE":
		if len(p.FileSection) == 0 {
			return fmt.Errorf("%s: record before any FD", entry.Pos)
		}
		fd := &p.FileSection[len(p.FileSection)-1]
		fd.Entries = append(fd.Entries, entry)
	case "WORKING-STORAGE":
		p.WorkingStorage = append(p.WorkingStorage, entry)
	case "LINKAGE":
		p.Linkage = append(p.Linkage, entry)
	default:
		return fmt.Errorf("%s: the %s SECTION is not supported yet", entry.Pos, section)
	}
	return nil
}

// parseProcedure splits the procedure division into paragraphs and parses
// their statements.
func (p *Program) parseProcedure(tokens []Token) error {
	if len(tokens) > 0 && tokens[0].Is("USING") {
		i := 1
		for ; i < len(tokens) && tokens[i].Kind == Word && !tokens[i].AreaA; i++ {
			p.Using = append(p.Using, tokens[i].Text)
		}
		tokens = tokens[i:]
	}
	s := &splitter{program: p, procedures: map[string]bool{}}
	for i := 0; i < len(tokens); i++ {
		taken, err := s.take(tokens, i)
		if err != nil {
			return err
		}
		i += taken
	}
	conditions := map[string]bool{}
	for _, fd := range p.FileSection {
		addConditionNames(conditions, fd.Entries)
	}
	addConditionNames(conditions, p.WorkingStorage)
	for i := range p.Paragraphs {
		statements, err := ParseStatements(p.Paragraphs[i].Body, s.procedures, conditions)
		if err != nil {
			return fmt.Errorf("%s: %w", p.Paragraphs[i].Name, err)
		}
		p.Paragraphs[i].Statements = statements
	}
	return nil
}

// splitter splits the procedure division into paragraphs. A header is a
// name in area A that ends a sentence, as it follows a period, and is
// followed by a period, or by SECTION and a period; a verb is no name. A
// section's statements before its first paragraph, if any, are a
// paragraph named after the section. The DECLARATIVES come first, each
// section of them starting with its USE sentence.
type splitter struct {
	program     *Program
	procedures  map[string]bool
	section     string
	header      *Token // a section's header, until its first statement or paragraph
	declarative bool
}

// take takes the token at i, and returns how many tokens after it it took
// with it.
func (s *splitter) take(tokens []Token, i int) (int, error) {
	t := tokens[i]
	switch {
	case t.Is("DECLARATIVES") && i+1 < len(tokens) && tokens[i+1].Kind == Period:
		s.declarative = true
		return 1, nil
	case t.Is("END") && i+2 < len(tokens) && tokens[i+1].Is("DECLARATIVES") && tokens[i+2].Kind == Period:
		s.declarative, s.section, s.header = false, "", nil
		return 2, nil
	case t.Is("USE") && s.header != nil:
		return s.use(tokens, i)
	case !isHeader(tokens, i):
	case i+2 < len(tokens) && tokens[i+1].Is("SECTION") && tokens[i+2].Kind == Period:
		s.section, s.header = t.Text, &tokens[i]
		s.procedures[t.Text] = true
		return 2, nil
	case i+1 < len(tokens) && tokens[i+1].Kind == Period:
		if s.procedures[t.Text] {
			return 0, fmt.Errorf("%s: a second procedure named %s is not supported yet", t.Pos, t.Text)
		}
		s.paragraph(t.Text, t)
		s.procedures[t.Text] = true
		return 1, nil
	}
	if s.header != nil {
		s.paragraph(s.section, *s.header)
	}
	paragraphs := s.program.Paragraphs
	if len(paragraphs) == 0 {
		return 0, fmt.Errorf("%s: statements before the first paragraph are not supported yet", t.Pos)
	}
	paragraphs[len(paragraphs)-1].Body = append(paragraphs[len(paragraphs)-1].Body, t)
	return 0, nil
}

// paragraph starts a paragraph, at its header or the header of the
// section it is the statements of.
func (s *splitter) paragraph(name string, header Token) {
	s.program.Paragraphs = append(s.program.Paragraphs, Paragraph{
		Name: name, Section: s.section, Declarative: s.declarative, Comment: header.Comment, Pos: header.Pos,
	})
	s.header = nil
}

// use reads the USE sentence that starts a section of the DECLARATIVES:
// USE [GLOBAL] AFTER [STANDARD] EXCEPTION|ERROR PROCEDURE [ON] followed by
// file names or open modes.
func (s *splitter) use(tokens []Token, i int) (int, error) {
	at := tokens[i].Pos
	if !s.declarative {
		return 0, fmt.Errorf("%s: USE outside the DECLARATIVES", at)
	}
	d := Declarative{Section: s.section, Pos: at}
	j := i + 1
	for ; j < len(tokens) && tokens[j].Kind != Period; j++ {
		switch t := tokens[j]; t.Text {
		case "GLOBAL", "AFTER", "STANDARD", "EXCEPTION", "ERROR", "PROCEDURE", "ON":
		case "INPUT", "OUTPUT", "I-O", "EXTEND":
			d.Modes = append(d.Modes, t.Text)
		case "FOR", "DEBUGGING":
			return 0, fmt.Errorf("%s: USE FOR DEBUGGING is not supported yet", at)
		default:
			d.Files = append(d.Files, t.Text)
		}
	}
	s.program.Declaratives = append(s.program.Declaratives, d)
	return j - i, nil
}

// isHeader reports whether the token at i can name a section or paragraph:
// a word in area A, not a verb, that starts a sentence.
func isHeader(tokens []Token, i int) bool {
	t := tokens[i]
	return t.Kind == Word && t.AreaA && !verbs[t.Text] && (i == 0 || tokens[i-1].Kind == Period)
}

func addConditionNames(names map[string]bool, entries []*DataEntry) {
	for _, e := range entries {
		if e.Level == 88 {
			names[e.Name] = true
		}
	}
}

// cursor walks the tokens of one entry.
type cursor struct {
	tokens []Token
	i      int
}

func (c *cursor) done() bool { return c.i >= len(c.tokens) }

func (c *cursor) peek() Token {
	if c.done() {
		return Token{}
	}
	return c.tokens[c.i]
}

func (c *cursor) next() Token {
	t := c.peek()
	c.i++
	return t
}

func (c *cursor) optional(word string) bool {
	if c.peek().Is(word) {
		c.i++
		return true
	}
	return false
}

func (c *cursor) expect(word string) error {
	t := c.next()
	if !t.Is(word) {
		return fmt.Errorf("%s: expected %s, found %s", t.Pos, word, t)
	}
	return nil
}

func (c *cursor) number() (int, error) {
	t := c.next()
	n, err := strconv.Atoi(t.Text)
	if t.Kind != Word || err != nil {
		return 0, fmt.Errorf("%s: expected a number, found %s", t.Pos, t)
	}
	return n, nil
}

// ReadDataEntries reads a copybook that holds data description entries.
func ReadDataEntries(path string, copybookDirs []string) ([]*DataEntry, error) {
	tokens, err := Read(path, copybookDirs)
	if err != nil {
		return nil, err
	}
	var out []*DataEntry
	for _, tokens := range entries(tokens) {
		entry, err := ParseDataEntry(tokens)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}
