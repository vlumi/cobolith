// Package jcl parses job network files in a small subset of IBM JCL: a JOB
// statement, EXEC statements running programs, and DD statements binding a
// program's file names to datasets or to the job's output: what runs a
// job needs, and what draws the network of jobs and their datasets.
package jcl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Job is a parsed job: its name, its steps in order, and the comment
// banner that describes it: the comment lines after the JOB statement from
// a rule of = signs to the next.
type Job struct {
	Name    string
	Comment []string
	Steps   []Step
}

// Step runs one program with its files bound by DD statements. Comment
// holds the comment lines before its EXEC statement.
type Step struct {
	Name    string
	Program string
	Cond    *Cond
	DDs     []DD
	Comment []string
	Line    int
}

// Cond is a step's COND=(code,operator) test. The step is bypassed when
// "code operator RC" holds for the return code of any earlier step, so
// COND=(4,LT) bypasses the step if an earlier step ended with RC above 4.
type Cond struct {
	Code     int
	Operator string
}

// Bypasses reports whether the condition bypasses the step after a step
// that ended with return code rc.
func (c Cond) Bypasses(rc int) bool {
	switch c.Operator {
	case "GT":
		return c.Code > rc
	case "GE":
		return c.Code >= rc
	case "EQ":
		return c.Code == rc
	case "NE":
		return c.Code != rc
	case "LT":
		return c.Code < rc
	case "LE":
		return c.Code <= rc
	}
	return false
}

// Disp is a dataset's disposition: whether the step reads an existing
// dataset, creates a new one, or appends to one, creating it if needed.
type Disp string

const (
	Shr Disp = "SHR"
	Old Disp = "OLD"
	New Disp = "NEW"
	Mod Disp = "MOD"
)

// DD binds the file a program knows by Name to a dataset (DSN, Disp and,
// for new datasets, LRECL, if given), to the job's output (Sysout), or to
// nothing (Dummy: reads find no records, writes are discarded).
//
// DISP=(status,normal,abnormal) is Disp, the status, NEW when it is left
// out or there is no DISP; and Normal and Abnormal, what becomes of the
// dataset when the step ends, normally or not, as written: KEEP, DELETE,
// PASS, CATLG or UNCATLG, empty when not given.
//
// DD * is InStream: the job's in-stream data, which a job's text may not
// carry, its lines being no statements; whoever runs the job supplies it.
//
// RECORG=KS makes the dataset Indexed, a key-sequenced dataset whose
// records a program reads and writes by key. Its record length is given as
// LRECL=n, which a DD may also give instead of DCB=(RECFM=FB,LRECL=n).
//
// DSN=NAME(+1) is a generation of the generation data group NAME, relative
// to the newest generation when the job started: (0) is that one, (-1) the
// one before, (+1) the next. DSN=NAME of a generation data group is every
// generation it has, newest first. DSN=&&NAME is a temporary dataset that
// exists only while the job runs.
//
// A DD statement without a name continues the one before: the program
// reads the datasets of both, one after the other, as one file.
type DD struct {
	Name       string
	DSN        string // without the generation or the &&
	Generation int
	Relative   bool
	Temporary  bool
	Disp       Disp
	Normal     string
	Abnormal   string
	LRECL      int
	Indexed    bool
	Sysout     bool
	Dummy      bool
	InStream   bool
	Line       int
	// Concatenated are the datasets read after this one.
	Concatenated []DD
}

// statement is one JCL statement with its continuation lines joined, and
// the text of the comment lines before it, without their //*.
type statement struct {
	name      string
	operation string
	operands  string
	comment   []string
	line      int
}

// ParseFile reads a job from a file.
func ParseFile(path string) (*Job, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	job, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return job, nil
}

// Parse reads a job from r.
func Parse(r io.Reader) (*Job, error) {
	statements, err := readStatements(r)
	if err != nil {
		return nil, err
	}
	if len(statements) == 0 || statements[0].operation != "JOB" {
		return nil, fmt.Errorf("line 1: a job starts with a JOB statement")
	}
	job := &Job{Name: statements[0].name}
	if len(statements) > 1 {
		job.Comment, statements[1].comment = splitBanner(statements[1].comment)
	}
	for _, s := range statements[1:] {
		if err := job.add(s); err != nil {
			return nil, fmt.Errorf("line %d: %w", s.line, err)
		}
	}
	return job, nil
}

func (job *Job) add(s statement) error {
	switch s.operation {
	case "EXEC":
		step, err := parseExec(s)
		if err != nil {
			return err
		}
		step.Comment = s.comment
		job.Steps = append(job.Steps, step)
	case "DD":
		if len(job.Steps) == 0 {
			return fmt.Errorf("DD %s comes before any EXEC", s.name)
		}
		dd, err := parseDD(s)
		if err != nil {
			return err
		}
		step := &job.Steps[len(job.Steps)-1]
		if dd.Name == "" {
			return step.concatenate(dd)
		}
		step.DDs = append(step.DDs, dd)
	default:
		return fmt.Errorf("unknown operation %q", s.operation)
	}
	return nil
}

// concatenate adds a DD without a name to the step's last DD. Only
// datasets that are read can be concatenated.
func (step *Step) concatenate(dd DD) error {
	if len(step.DDs) == 0 {
		return fmt.Errorf("a DD without a name comes first in step %s", step.Name)
	}
	head := &step.DDs[len(step.DDs)-1]
	for _, d := range []DD{*head, dd} {
		if d.DSN == "" || d.Disp != Shr && d.Disp != Old {
			return fmt.Errorf("DD %s: only datasets read with DISP=SHR or OLD can be concatenated", head.Name)
		}
	}
	head.Concatenated = append(head.Concatenated, dd)
	return nil
}

// readStatements splits the input into statements, skipping comments and
// joining continuations: an operand field that ends with a comma continues
// on the next line. Text after the operand field is a comment.
func readStatements(r io.Reader) ([]statement, error) {
	var statements []statement
	var comment []string
	continuing := false
	scanner := bufio.NewScanner(r)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimRight(scanner.Text(), " ")
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "//*"):
			comment = append(comment, line[3:])
			continue
		case !strings.HasPrefix(line, "//"):
			return nil, fmt.Errorf("line %d: a statement starts with //", lineNo)
		case continuing:
			last := &statements[len(statements)-1]
			operands := firstField(line[2:])
			last.operands += operands
			continuing = strings.HasSuffix(operands, ",")
		default:
			s, err := splitStatement(line, lineNo)
			if err != nil {
				return nil, err
			}
			s.comment, comment = comment, nil
			statements = append(statements, s)
			continuing = strings.HasSuffix(s.operands, ",")
		}
	}
	return statements, scanner.Err()
}

// splitBanner splits the comment lines before the first step into the
// job's banner, from a rule of = signs to the next, and the step's own.
func splitBanner(lines []string) (banner, rest []string) {
	isRule := func(line string) bool {
		t := strings.Trim(strings.TrimSpace(line), "*")
		return t != "" && strings.Trim(t, "=") == ""
	}
	if len(lines) == 0 || !isRule(lines[0]) {
		return nil, lines
	}
	for i := 1; i < len(lines); i++ {
		if isRule(lines[i]) {
			return lines[:i+1], lines[i+1:]
		}
	}
	return lines, nil
}

// splitStatement splits a statement into its name, from column 3, its
// operation and its operands. Only a DD statement may go without a name.
func splitStatement(line string, lineNo int) (statement, error) {
	fields := strings.Fields(line[2:])
	if strings.HasPrefix(line, "// ") {
		fields = append([]string{""}, fields...)
	}
	if len(fields) < 2 || fields[0] == "" && fields[1] != "DD" {
		return statement{}, fmt.Errorf("line %d: a statement needs a name in column 3 and an operation", lineNo)
	}
	s := statement{name: fields[0], operation: fields[1], line: lineNo}
	if len(fields) > 2 {
		s.operands = fields[2]
	}
	return s, nil
}

// firstField returns the operands of a continuation line: what follows the
// first blank after them is a comment.
func firstField(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func parseExec(s statement) (Step, error) {
	step := Step{Name: s.name, Line: s.line}
	for _, operand := range splitOperands(s.operands) {
		key, value, _ := strings.Cut(operand, "=")
		switch key {
		case "PGM":
			step.Program = value
		case "COND":
			cond, err := parseCond(value)
			if err != nil {
				return Step{}, err
			}
			step.Cond = &cond
		default:
			return Step{}, fmt.Errorf("EXEC %s: unknown operand %q", s.name, key)
		}
	}
	if step.Program == "" {
		return Step{}, fmt.Errorf("EXEC %s: no PGM", s.name)
	}
	return step, nil
}

func parseCond(value string) (Cond, error) {
	parts := splitOperands(unparenthesize(value))
	if len(parts) != 2 {
		return Cond{}, fmt.Errorf("COND=%s: expected (code,operator)", value)
	}
	code, err := strconv.Atoi(parts[0])
	if err != nil {
		return Cond{}, fmt.Errorf("COND=%s: code is not a number", value)
	}
	cond := Cond{Code: code, Operator: parts[1]}
	switch cond.Operator {
	case "GT", "GE", "EQ", "NE", "LT", "LE":
		return cond, nil
	}
	return Cond{}, fmt.Errorf("COND=%s: unknown operator %q", value, cond.Operator)
}

func parseDD(s statement) (DD, error) {
	dd := DD{Name: s.name, Line: s.line}
	for _, operand := range splitOperands(s.operands) {
		key, value, _ := strings.Cut(operand, "=")
		switch key {
		case "DSN":
			if err := dd.setDSN(value); err != nil {
				return DD{}, err
			}
		case "DISP":
			dd.Disp, dd.Normal, dd.Abnormal = parseDisp(value)
		case "DCB":
			lrecl, err := parseDCB(value)
			if err != nil {
				return DD{}, err
			}
			dd.LRECL = lrecl
		case "LRECL":
			lrecl, err := parseLRECL(value)
			if err != nil {
				return DD{}, err
			}
			dd.LRECL = lrecl
		case "RECORG":
			if value != "KS" {
				return DD{}, fmt.Errorf("DD %s: RECORG=%s, only KS is supported", s.name, value)
			}
			dd.Indexed = true
		case "SYSOUT":
			dd.Sysout = true
		case "DUMMY":
			dd.Dummy = true
		case "*":
			dd.InStream = true
		default:
			return DD{}, fmt.Errorf("DD %s: unknown operand %q", s.name, key)
		}
	}
	if dd.DSN != "" && dd.Disp == "" {
		dd.Disp = New
	}
	return dd, dd.check()
}

// setDSN reads a dataset name with an optional generation or && prefix.
func (dd *DD) setDSN(value string) error {
	name, generation, relative := strings.Cut(value, "(")
	if strings.HasPrefix(name, "&&") {
		dd.Temporary = true
		name = name[2:]
	}
	dd.DSN = name
	if !relative {
		return nil
	}
	n, err := strconv.Atoi(strings.TrimSuffix(generation, ")"))
	if err != nil || !strings.HasSuffix(generation, ")") {
		return fmt.Errorf("DD %s: DSN=%s: bad generation", dd.Name, value)
	}
	dd.Generation, dd.Relative = n, true
	return nil
}

func (dd DD) check() error {
	targets := 0
	for _, set := range []bool{dd.Sysout, dd.Dummy, dd.InStream, dd.DSN != ""} {
		if set {
			targets++
		}
	}
	switch {
	case targets != 1:
		return fmt.Errorf("DD %s: needs exactly one of DSN, SYSOUT, DUMMY and *", dd.Name)
	case dd.Sysout || dd.Dummy || dd.InStream:
		return nil
	case dd.Temporary && dd.Relative:
		return fmt.Errorf("DD %s: a temporary dataset has no generations", dd.Name)
	case dd.Disp != Shr && dd.Disp != Old && dd.Disp != New && dd.Disp != Mod:
		return fmt.Errorf("DD %s: DISP=%s is not SHR, OLD, NEW or MOD", dd.Name, dd.Disp)
	}
	for _, end := range []string{dd.Normal, dd.Abnormal} {
		if end != "" && !slices.Contains([]string{"KEEP", "DELETE", "PASS", "CATLG", "UNCATLG"}, end) {
			return fmt.Errorf("DD %s: %s is not KEEP, DELETE, PASS, CATLG or UNCATLG", dd.Name, end)
		}
	}
	return nil
}

// parseDisp reads DISP=status or DISP=(status,normal,abnormal), any of
// them left out; the status is NEW when it is.
func parseDisp(value string) (status Disp, normal, abnormal string) {
	parts := strings.Split(unparenthesize(value), ",")
	parts = append(parts, "", "")
	status = Disp(parts[0])
	if status == "" {
		status = New
	}
	return status, parts[1], parts[2]
}

// parseDCB reads the record length from DCB=(RECFM=FB,LRECL=n): fixed-length
// records without delimiters, the only record format read.
func parseDCB(value string) (int, error) {
	lrecl := 0
	for _, attribute := range splitOperands(unparenthesize(value)) {
		key, v, _ := strings.Cut(attribute, "=")
		switch key {
		case "RECFM":
			if v != "FB" {
				return 0, fmt.Errorf("DCB: RECFM=%s, only FB is supported", v)
			}
		case "LRECL":
			n, err := parseLRECL(v)
			if err != nil {
				return 0, fmt.Errorf("DCB: %w", err)
			}
			lrecl = n
		default:
			return 0, fmt.Errorf("DCB: unknown attribute %q", key)
		}
	}
	return lrecl, nil
}

func parseLRECL(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("LRECL=%s is not a record length", value)
	}
	return n, nil
}

// splitOperands splits at the commas that are outside parentheses and quotes.
func splitOperands(s string) []string {
	var operands []string
	depth, quoted, start := 0, false, 0
	for i, c := range s {
		switch {
		case c == '\'':
			quoted = !quoted
		case quoted:
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == ',' && depth == 0:
			operands = append(operands, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		operands = append(operands, s[start:])
	}
	return operands
}

func unparenthesize(s string) string {
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return s[1 : len(s)-1]
	}
	return s
}
