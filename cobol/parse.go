package cobol

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
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
	ReportSection  []ReportDescription
	WorkingStorage []*DataEntry
	// Linkage are the items a calling program passes, which the procedure
	// division names in Using.
	Linkage    []*DataEntry
	Using      []string
	Paragraphs []Paragraph
	// DecimalPointComma is SPECIAL-NAMES' DECIMAL-POINT IS COMMA, and
	// CurrencySign its CURRENCY SIGN. The program's pictures and numeric
	// literals are read as written with them, and kept in the standard
	// form: the decimal point a period, the currency symbol $.
	DecimalPointComma bool
	CurrencySign      string
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
// one of them fails, and control goes on after that statement. A
// Debugging one, USE FOR DEBUGGING, runs as the debugger's hook instead,
// and names no file.
type Declarative struct {
	Section   string
	Files     []string
	Modes     []string
	Debugging bool
	Pos       Pos
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
	// AlternateKeys are an indexed file's other keys, which it may also be
	// read by.
	AlternateKeys []AlternateKey
	RelativeKey   string // the item that holds a relative file's record number
	Status        string
	Pos           Pos
}

// selectClauses are the words that start a clause of a SELECT entry: a
// word of them after RELATIVE makes RELATIVE the organization, any other
// the name of the relative key.
var selectClauses = map[string]bool{
	"ASSIGN": true, "ORGANIZATION": true, "ACCESS": true, "RECORD": true, "ALTERNATE": true,
	"RELATIVE": true, "FILE": true, "STATUS": true, "RESERVE": true, "PADDING": true,
	"SEQUENTIAL": true, "INDEXED": true,
}

// AlternateKey is an alternate record key of an indexed file, and whether
// records may share its value.
type AlternateKey struct {
	Name       string
	Duplicates bool
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
	// Reports are the reports of the REPORT SECTION written to the file,
	// REPORT IS or REPORTS ARE; a file with any has no records of its own.
	Reports []string
	Pos     Pos
}

// ReportDescription is an RD of the REPORT SECTION, the Report Writer's
// description of a report: the file writes it as INITIATE, GENERATE and
// TERMINATE say. Controls are the items whose change of value starts a
// control heading and ends a control footing, FINAL among them when the
// report has one; the page limit and the line numbers of its parts are
// as the PAGE clause gives them, zero where it does not. Its entries are
// the report groups, each a level-01 entry whose TYPE says which part of
// the report it is, with the lines and columns under it.
type ReportDescription struct {
	Name        string
	Controls    []string
	PageLimit   int
	Heading     int
	FirstDetail int
	LastDetail  int
	Footing     int
	Entries     []*DataEntry
	Comment     []string
	Pos         Pos
}

// Paragraph is a paragraph of the procedure division: the tokens of its
// body and the statements they make. Paragraphs before the first section
// header have no section. A declarative paragraph is one of the
// DECLARATIVES, which nothing performs or falls into. The statements of a
// procedure division before its first paragraph or section are a
// paragraph with no name.
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
	return ReadProgramIn(path, copybookDirs, Dialect{})
}

// ReadProgramIn reads and parses a program written in a dialect, whose
// encoding the source is read in.
func ReadProgramIn(path string, copybookDirs []string, d Dialect) (*Program, error) {
	tokens, err := ReadIn(path, copybookDirs, d.Encoding)
	if err != nil {
		return nil, err
	}
	return Parse(path, tokens)
}

// Parse parses the tokens of a program, and of the programs nested in it.
// A source file of more than one program is ParseAll's.
func Parse(path string, tokens []Token) (*Program, error) {
	p, rest, err := parseFirst(path, tokens)
	if err == nil && len(rest) > 0 {
		err = fmt.Errorf("%s: %s: a second program after END PROGRAM, which ParseAll reads", path, rest[0].Pos)
	}
	return p, err
}

// ReadPrograms reads and parses every program of a source file, its
// copybooks expanded: those it holds one after the other, each ended by
// END PROGRAM, each with the programs nested in it.
func ReadPrograms(path string, copybookDirs []string) ([]*Program, error) {
	return ReadProgramsIn(path, copybookDirs, Dialect{})
}

// ReadProgramsIn reads and parses every program of a source file written
// in a dialect.
func ReadProgramsIn(path string, copybookDirs []string, d Dialect) ([]*Program, error) {
	tokens, err := ReadIn(path, copybookDirs, d.Encoding)
	if err != nil {
		return nil, err
	}
	return ParseAll(path, tokens)
}

// ParseAll parses the tokens of every program of a source file, in order.
func ParseAll(path string, tokens []Token) ([]*Program, error) {
	var programs []*Program
	for len(tokens) > 0 {
		p, rest, err := parseFirst(path, tokens)
		if err != nil {
			return nil, err
		}
		programs = append(programs, p)
		tokens = rest
	}
	return programs, nil
}

// parseFirst parses the first program of the tokens, and returns the
// tokens after it.
func parseFirst(path string, tokens []Token) (*Program, []Token, error) {
	own, nested, rest, err := splitNested(tokens)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	p, err := parseProgram(path, tokens, own, nested)
	return p, rest, err
}

// parseProgram parses a program from its own tokens, and the programs
// nested in it from theirs; its comment is that before its first token.
func parseProgram(path string, tokens, own []Token, nested [][]Token) (*Program, error) {
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
	if err := p.parseIdentification(divisions["IDENTIFICATION"]); err != nil {
		return nil, err
	}
	if err := p.parseEnvironment(divisions["ENVIRONMENT"]); err != nil {
		return nil, err
	}
	if err := p.parseData(p.standardized(divisions["DATA"])); err != nil {
		return nil, err
	}
	if err := p.parseProcedure(p.standardized(divisions["PROCEDURE"])); err != nil {
		return nil, err
	}
	if p.ID == "" {
		return nil, fmt.Errorf("%s: no PROGRAM-ID", path)
	}
	return p, nil
}

// splitNested separates a program's own tokens from those of the programs
// nested in it, each from its IDENTIFICATION DIVISION through the END
// PROGRAM that names it. The program's own END PROGRAM, if it has one,
// ends it, and the rest are the tokens after it.
func splitNested(tokens []Token) (own []Token, nested [][]Token, rest []Token, err error) {
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
				return own, nested, tokens[end+1:], nil
			}
			if innermost := open[len(open)-1]; name != innermost {
				return nil, nil, nil, fmt.Errorf("%s: END PROGRAM %s where %s has not ended", tokens[i].Pos, name, innermost)
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
		return nil, nil, nil, fmt.Errorf("%s: no END PROGRAM %s", tokens[start].Pos, open[len(open)-1])
	}
	return own, nested, nil, nil
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
		case t.Kind == CommentEntry && i >= 2:
			p.Identification = append(p.Identification, IdentificationEntry{Paragraph: tokens[i-2].Text, Text: t.Text})
		}
	}
	return nil
}

func (p *Program) parseEnvironment(tokens []Token) error {
	for _, entry := range entries(tokens) {
		switch {
		case entry[0].Is("SELECT"):
			file, err := parseSelect(entry)
			if err != nil {
				return err
			}
			p.Files = append(p.Files, file)
		default:
			p.specialNames(entry)
		}
	}
	return nil
}

// specialNames reads the clauses of a SPECIAL-NAMES entry that change how
// the program is written: DECIMAL-POINT IS COMMA, and CURRENCY [SIGN]
// [IS] literal.
func (p *Program) specialNames(entry []Token) {
	for i, t := range entry {
		rest := entry[i+1:]
		for len(rest) > 0 && (rest[0].Is("IS") || rest[0].Is("SIGN")) {
			rest = rest[1:]
		}
		switch {
		case len(rest) == 0:
		case t.Is("DECIMAL-POINT") && rest[0].Is("COMMA"):
			p.DecimalPointComma = true
		case t.Is("CURRENCY") && rest[0].Kind == String:
			p.CurrencySign = rest[0].Text
		}
	}
}

// standardized gives the data and procedure divisions' tokens their
// standard form, where SPECIAL-NAMES changed it: a numeric literal's comma
// becomes a period, and in a picture, periods and commas swap and the
// currency sign becomes $.
func (p *Program) standardized(tokens []Token) []Token {
	if !p.DecimalPointComma && p.CurrencySign == "" {
		return tokens
	}
	out := slices.Clone(tokens)
	picture := false
	for i := range out {
		t := &out[i]
		switch {
		case t.Kind != Word:
		case t.Is("PIC") || t.Is("PICTURE"):
			picture = true
			continue
		case picture && t.Is("IS"):
			continue
		case picture:
			if p.DecimalPointComma {
				t.Text = strings.Map(swapPoint, t.Text)
			}
			if p.CurrencySign != "" {
				t.Text = strings.ReplaceAll(t.Text, p.CurrencySign, "$")
			}
		case p.DecimalPointComma && isNumber(strings.Replace(t.Text, ",", ".", 1)) && strings.Contains(t.Text, ","):
			t.Text = strings.Replace(t.Text, ",", ".", 1)
		}
		picture = false
	}
	return out
}

func swapPoint(r rune) rune {
	switch r {
	case '.':
		return ','
	case ',':
		return '.'
	}
	return r
}

// parseSelect parses SELECT [OPTIONAL] name ASSIGN [TO] target and the
// ORGANIZATION, ACCESS MODE, RECORD KEY, ALTERNATE RECORD KEY, RELATIVE
// KEY and FILE STATUS clauses; and RESERVE, PADDING and RECORD DELIMITER,
// which change no record, and are passed over.
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
		case clause.Is("ORGANIZATION") || clause.Is("SEQUENTIAL") || clause.Is("INDEXED") || clause.Is("RELATIVE") && (c.done() || selectClauses[c.peek().Text]):
			if clause.Is("ORGANIZATION") {
				c.optional("IS")
				clause = c.next()
			}
			file.Organization = clause.Text
			if file.Organization == "LINE" || file.Organization == "RECORD" {
				file.Organization += " " + c.next().Text
			}
		case clause.Is("ACCESS"):
			c.optional("MODE")
			c.optional("IS")
			file.Access = c.next().Text
		case clause.Is("RECORD") && c.peek().Is("DELIMITER"):
			c.next()
			c.optional("IS")
			c.next()
		case clause.Is("RECORD"):
			c.optional("KEY")
			c.optional("IS")
			file.RecordKey = c.qualified()
		case clause.Is("ALTERNATE"):
			c.optional("RECORD")
			c.optional("KEY")
			c.optional("IS")
			key := AlternateKey{Name: c.qualified()}
			if c.optional("WITH") || c.peek().Is("DUPLICATES") {
				key.Duplicates = c.optional("DUPLICATES")
			}
			file.AlternateKeys = append(file.AlternateKeys, key)
		case clause.Is("RESERVE"):
			if _, err := c.number(); err != nil {
				return FileControl{}, err
			}
			if !c.optional("AREA") {
				c.optional("AREAS")
			}
		case clause.Is("PADDING"):
			c.optional("CHARACTER")
			c.optional("IS")
			c.next()
		case clause.Is("RELATIVE"):
			c.optional("KEY")
			c.optional("IS")
			file.RelativeKey = c.qualified()
		case clause.Is("FILE") || clause.Is("STATUS"):
			if clause.Is("FILE") {
				if err := c.expect("STATUS"); err != nil {
					return FileControl{}, err
				}
			}
			c.optional("IS")
			file.Status = c.qualified()
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
		case entry[0].Is("RD"):
			if len(entry) < 2 {
				return fmt.Errorf("%s: RD without a report name", entry[0].Pos)
			}
			if section != "REPORT" {
				return fmt.Errorf("%s: RD outside the REPORT SECTION", entry[0].Pos)
			}
			rd, err := parseReportDescription(entry)
			if err != nil {
				return err
			}
			p.ReportSection = append(p.ReportSection, rd)
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
// [ON] item. The clauses that change no record, BLOCK, LABEL, VALUE OF,
// DATA RECORDS, LINAGE, CODE-SET, EXTERNAL and GLOBAL, are passed over.
func parseFileDescription(entry []Token) (FileDescription, error) {
	fd := FileDescription{Kind: entry[0].Text, Name: entry[1].Text, Pos: entry[0].Pos}
	c := &cursor{tokens: entry[2:]}
	for !c.done() {
		clause := c.next()
		switch {
		case clause.Is("RECORD"):
			if err := fd.parseRecordClause(c); err != nil {
				return FileDescription{}, err
			}
		case clause.Is("LABEL") || clause.Is("DATA"):
			if !c.optional("RECORD") {
				c.optional("RECORDS")
			}
			c.passClause()
		case clause.Is("BLOCK") || clause.Is("VALUE") || clause.Is("LINAGE") || clause.Is("CODE-SET") || clause.Is("IS"):
			c.passClause()
		case clause.Is("REPORT") || clause.Is("REPORTS"):
			if !c.optional("IS") {
				c.optional("ARE")
			}
			for !c.done() && (c.peek().Kind != Word || !fdClauses[c.peek().Text]) {
				fd.Reports = append(fd.Reports, c.next().Text)
			}
			if len(fd.Reports) == 0 {
				return FileDescription{}, fmt.Errorf("%s: %s %s: REPORT without a report name", clause.Pos, fd.Kind, fd.Name)
			}
		case clause.Is("EXTERNAL") || clause.Is("GLOBAL"):
		default:
			return FileDescription{}, fmt.Errorf("%s: %s %s: %s is not supported yet", clause.Pos, fd.Kind, fd.Name, clause)
		}
	}
	return fd, nil
}

// fdClauses are the words that start a clause of an FD or SD entry.
var fdClauses = map[string]bool{
	"RECORD": true, "BLOCK": true, "LABEL": true, "VALUE": true, "DATA": true, "LINAGE": true,
	"CODE-SET": true, "EXTERNAL": true, "GLOBAL": true, "REPORT": true, "REPORTS": true,
}

// parseReportDescription parses RD name and its clauses: CONTROL IS or
// CONTROLS ARE and the control items, FINAL among them; PAGE [LIMIT IS] n
// [LINES] and the line numbers HEADING, FIRST DETAIL, LAST DETAIL and
// FOOTING, in any order; CODE and GLOBAL, which change nothing here.
func parseReportDescription(entry []Token) (ReportDescription, error) {
	rd := ReportDescription{Name: entry[1].Text, Comment: entry[0].Comment, Pos: entry[0].Pos}
	c := &cursor{tokens: entry[2:]}
	var err error
	for !c.done() {
		clause := c.next()
		switch {
		case clause.Is("CONTROL") || clause.Is("CONTROLS"):
			if !c.optional("IS") {
				c.optional("ARE")
			}
			for !c.done() && (c.peek().Kind != Word || !rdClauses[c.peek().Text]) {
				rd.Controls = append(rd.Controls, c.qualified())
			}
		case clause.Is("PAGE"):
			if !c.optional("LIMIT") {
				c.optional("LIMITS")
			}
			if !c.optional("IS") {
				c.optional("ARE")
			}
			if rd.PageLimit, err = c.number(); err != nil {
				return ReportDescription{}, err
			}
			if !c.optional("LINE") {
				c.optional("LINES")
			}
		case clause.Is("HEADING"):
			rd.Heading, err = c.number()
		case clause.Is("FIRST"):
			if err = c.expect("DETAIL"); err == nil {
				rd.FirstDetail, err = c.number()
			}
		case clause.Is("LAST"):
			if err = c.expect("DETAIL"); err == nil {
				rd.LastDetail, err = c.number()
			}
		case clause.Is("FOOTING"):
			rd.Footing, err = c.number()
		case clause.Is("IS") || clause.Is("GLOBAL"):
		case clause.Is("CODE"):
			c.optional("IS")
			c.next()
		default:
			return ReportDescription{}, fmt.Errorf("%s: RD %s: %s is not supported yet", clause.Pos, rd.Name, clause)
		}
		if err != nil {
			return ReportDescription{}, err
		}
	}
	return rd, nil
}

// rdClauses are the words that start a clause of an RD entry.
var rdClauses = map[string]bool{
	"CONTROL": true, "CONTROLS": true, "PAGE": true, "HEADING": true, "FIRST": true, "LAST": true,
	"FOOTING": true, "CODE": true, "GLOBAL": true, "IS": true,
}

// passClause passes over the rest of a clause of an FD or SD entry, up to
// the next clause.
func (c *cursor) passClause() {
	for !c.done() && (c.peek().Kind != Word || !fdClauses[c.peek().Text]) {
		c.next()
	}
}

// parseRecordClause parses the rest of a RECORD clause, after RECORD: no
// lengths at all, for RECORD IS VARYING alone.
func (fd *FileDescription) parseRecordClause(c *cursor) error {
	for _, word := range []string{"CONTAINS", "IS", "VARYING", "IN", "SIZE", "FROM"} {
		c.optional(word)
	}
	if c.done() || fdClauses[c.peek().Text] {
		return nil
	}
	if c.optional("DEPENDING") {
		c.optional("ON")
		fd.LengthIn = c.next().Text
		return nil
	}
	var err error
	if fd.MinLength, err = c.number(); err != nil {
		return err
	}
	fd.MaxLength = fd.MinLength
	if c.optional("TO") {
		if fd.MaxLength, err = c.number(); err != nil {
			return err
		}
	}
	c.optional("CHARACTERS")
	if c.optional("DEPENDING") {
		c.optional("ON")
		fd.LengthIn = c.next().Text
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
	case "REPORT":
		if len(p.ReportSection) == 0 {
			return fmt.Errorf("%s: report group before any RD", entry.Pos)
		}
		rd := &p.ReportSection[len(p.ReportSection)-1]
		rd.Entries = append(rd.Entries, entry)
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
	s := &splitter{program: p, procedures: map[string]bool{}, paragraphs: map[string]map[string]bool{}}
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
		statements, err := parseStatements(p.Paragraphs[i].Body, s.procedures, conditions)
		if err != nil {
			return fmt.Errorf("%s: %w", p.Paragraphs[i].Name, err)
		}
		p.Paragraphs[i].Statements = statements
	}
	return p.resolveProcedures()
}

// resolveProcedures gives each reference to a paragraph by its name alone
// the section of the paragraph it names: that of the referring paragraph's
// own section, if the section has one by the name, or else the only one.
// A name more than one section has, and the referring one not, is an
// error: it needs OF and its section.
func (p *Program) resolveProcedures() error {
	sections := map[string]bool{}
	in := map[string][]string{} // a paragraph name, and the sections that have one by it
	for _, para := range p.Paragraphs {
		sections[para.Section] = para.Section != ""
		if para.Name != "" && para.Name != para.Section {
			in[para.Name] = append(in[para.Name], para.Section)
		}
	}
	resolve := func(name, section, own string, at Pos) (string, error) {
		switch candidates := in[name]; {
		case section != "" || sections[name] || len(candidates) == 0:
			return section, nil
		case len(candidates) == 1:
			return candidates[0], nil
		case slices.Contains(candidates, own):
			return own, nil
		}
		return "", fmt.Errorf("%s: %s is a paragraph of more than one section, none of them this one: name its section with OF", at, name)
	}
	for i := range p.Paragraphs {
		para := &p.Paragraphs[i]
		var err error
		eachStatement(para.Statements, func(s *Statement) {
			for j := range s.Procedures {
				ref := &s.Procedures[j]
				if err == nil {
					ref.Section, err = resolve(ref.Name, ref.Section, para.Section, s.Pos)
				}
				if err == nil && ref.Thru != "" {
					ref.ThruSection, err = resolve(ref.Thru, ref.ThruSection, para.Section, s.Pos)
				}
			}
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// eachStatement visits statements and those nested in them, in place.
func eachStatement(statements []Statement, visit func(*Statement)) {
	for i := range statements {
		visit(&statements[i])
		for _, b := range statements[i].Branches {
			eachStatement(b.Statements, visit)
		}
	}
}

// splitter splits the procedure division into paragraphs. A header is a
// name in area A that ends a sentence, as it follows a period, and is
// followed by a period, or by SECTION and a period; a verb is no name. A
// section's statements before its first paragraph, if any, are a
// paragraph named after the section. The DECLARATIVES come first, each
// section of them starting with its USE sentence.
type splitter struct {
	program    *Program
	procedures map[string]bool
	// paragraphs are the names of each section's paragraphs, which need
	// be unique only within it.
	paragraphs  map[string]map[string]bool
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
	case i+3 < len(tokens) && tokens[i+1].Is("SECTION") && isNumber(tokens[i+2].Text) && tokens[i+3].Kind == Period:
		s.section, s.header = t.Text, &tokens[i]
		s.procedures[t.Text] = true
		return 3, nil
	case i+1 < len(tokens) && tokens[i+1].Kind == Period:
		if s.paragraphs[s.section][t.Text] {
			return 0, fmt.Errorf("%s: a second paragraph named %s in the same section", t.Pos, t.Text)
		}
		if s.paragraphs[s.section] == nil {
			s.paragraphs[s.section] = map[string]bool{}
		}
		s.paragraphs[s.section][t.Text] = true
		s.paragraph(t.Text, t)
		s.procedures[t.Text] = true
		return 1, nil
	}
	if s.header != nil {
		s.paragraph(s.section, *s.header)
	}
	if len(s.program.Paragraphs) == 0 {
		s.paragraph("", t)
	}
	paragraphs := s.program.Paragraphs
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
// file names or open modes, or USE FOR DEBUGGING ON the procedures or
// items it watches, which are not kept.
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
			d.Debugging = true
		case "ALL", "PROCEDURES", "REFERENCES", "OF":
		default:
			if !d.Debugging {
				d.Files = append(d.Files, t.Text)
			}
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

// qualified reads a name and passes over the OF or IN names that qualify
// it, which the name alone is enough to find in the programs read.
func (c *cursor) qualified() string {
	name := c.next().Text
	for !c.done() && (c.peek().Is("OF") || c.peek().Is("IN")) {
		c.next()
		c.next()
	}
	return name
}

func (c *cursor) number() (int, error) {
	if c.done() {
		return 0, fmt.Errorf("%s: expected a number, found the end of the entry", c.tokens[len(c.tokens)-1].Pos)
	}
	t := c.next()
	n, err := strconv.Atoi(t.Text)
	if t.Kind != Word || err != nil {
		return 0, fmt.Errorf("%s: expected a number, found %s", t.Pos, t)
	}
	return n, nil
}

// ReadDataEntries reads a copybook that holds data description entries.
func ReadDataEntries(path string, copybookDirs []string) ([]*DataEntry, error) {
	return ReadDataEntriesIn(path, copybookDirs, "")
}

// ReadDataEntriesIn reads a copybook of data description entries in an
// encoding, as ReadIn reads it.
func ReadDataEntriesIn(path string, copybookDirs []string, encoding string) ([]*DataEntry, error) {
	tokens, err := ReadIn(path, copybookDirs, encoding)
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
