package cobol

import (
	"fmt"
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
	Status       string
	Pos          Pos
}

// FileDescription is an FD or SD entry with the records under it.
type FileDescription struct {
	Kind    string // FD or SD
	Name    string
	Entries []*DataEntry
	Pos     Pos
}

// Paragraph is a paragraph of the procedure division: the tokens of its
// body and the statements they make. Paragraphs before the first section
// header have no section.
type Paragraph struct {
	Name       string
	Section    string
	Body       []Token
	Statements []Statement
	Comment    []string
	Pos        Pos
}

// ReadProgram reads and parses a program, expanding its copybooks.
func ReadProgram(path string, copybookDirs []string) (*Program, error) {
	tokens, err := Read(path, copybookDirs)
	if err != nil {
		return nil, err
	}
	return Parse(path, tokens)
}

// Parse parses the tokens of a program.
func Parse(path string, tokens []Token) (*Program, error) {
	divisions, err := splitDivisions(tokens)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	p := &Program{Path: path}
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
			p.FileSection = append(p.FileSection, FileDescription{
				Kind: entry[0].Text, Name: entry[1].Text, Pos: entry[0].Pos,
			})
		default:
			if err := p.addDataEntry(section, entry); err != nil {
				return err
			}
		}
	}
	return nil
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
// their statements. A header is a name in area A followed by a period, or by
// SECTION and a period.
func (p *Program) parseProcedure(tokens []Token) error {
	if len(tokens) > 0 && tokens[0].Is("USING") {
		i := 1
		for ; i < len(tokens) && tokens[i].Kind == Word && !tokens[i].Pos.InAreaA(); i++ {
			p.Using = append(p.Using, tokens[i].Text)
		}
		tokens = tokens[i:]
	}
	section := ""
	procedures := map[string]bool{}
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch {
		case !t.Pos.InAreaA() || t.Kind != Word:
		case i+2 < len(tokens) && tokens[i+1].Is("SECTION") && tokens[i+2].Kind == Period:
			section = t.Text
			procedures[t.Text] = true
			i += 2
			continue
		case i+1 < len(tokens) && tokens[i+1].Kind == Period:
			if procedures[t.Text] {
				return fmt.Errorf("%s: a second procedure named %s is not supported yet", t.Pos, t.Text)
			}
			p.Paragraphs = append(p.Paragraphs, Paragraph{Name: t.Text, Section: section, Comment: t.Comment, Pos: t.Pos})
			procedures[t.Text] = true
			i++
			continue
		}
		if len(p.Paragraphs) == 0 {
			return fmt.Errorf("%s: statements before the first paragraph are not supported yet", t.Pos)
		}
		last := &p.Paragraphs[len(p.Paragraphs)-1]
		last.Body = append(last.Body, t)
	}
	conditions := map[string]bool{}
	for _, fd := range p.FileSection {
		addConditionNames(conditions, fd.Entries)
	}
	addConditionNames(conditions, p.WorkingStorage)
	for i := range p.Paragraphs {
		statements, err := ParseStatements(p.Paragraphs[i].Body, procedures, conditions)
		if err != nil {
			return fmt.Errorf("%s: %w", p.Paragraphs[i].Name, err)
		}
		p.Paragraphs[i].Statements = statements
	}
	return nil
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
