// Package jcl parses Pangaea's job network files, a small subset of IBM JCL:
// a JOB statement, EXEC statements running programs, and DD statements
// binding a program's file names to datasets or to the job's output.
package jcl

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Job is a parsed job: its name and its steps in order.
type Job struct {
	Name  string
	Steps []Step
}

// Step runs one program with its files bound by DD statements.
type Step struct {
	Name    string
	Program string
	Cond    *Cond
	DDs     []DD
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
// dataset or creates a new one.
type Disp string

const (
	Shr Disp = "SHR"
	Old Disp = "OLD"
	New Disp = "NEW"
)

// DD binds the file a program knows by Name either to a cataloged dataset
// (DSN, Disp and, for new datasets, LRECL) or to the job's output (Sysout).
type DD struct {
	Name   string
	DSN    string
	Disp   Disp
	LRECL  int
	Sysout bool
	Line   int
}

// statement is one JCL statement with its continuation lines joined.
type statement struct {
	name      string
	operation string
	operands  string
	line      int
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
		step.DDs = append(step.DDs, dd)
	default:
		return fmt.Errorf("unknown operation %q", s.operation)
	}
	return nil
}

// readStatements splits the input into statements, skipping comments and
// joining continuations: an operand field that ends with a comma continues
// on the next line. Text after the operand field is a comment.
func readStatements(r io.Reader) ([]statement, error) {
	var statements []statement
	continuing := false
	scanner := bufio.NewScanner(r)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimRight(scanner.Text(), " ")
		switch {
		case line == "" || strings.HasPrefix(line, "//*"):
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
			statements = append(statements, s)
			continuing = strings.HasSuffix(s.operands, ",")
		}
	}
	return statements, scanner.Err()
}

func splitStatement(line string, lineNo int) (statement, error) {
	if strings.HasPrefix(line, "// ") {
		return statement{}, fmt.Errorf("line %d: a statement needs a name in column 3", lineNo)
	}
	fields := strings.Fields(line[2:])
	if len(fields) < 2 {
		return statement{}, fmt.Errorf("line %d: a statement needs a name and an operation", lineNo)
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
			dd.DSN = value
		case "DISP":
			dd.Disp = Disp(value)
		case "DCB":
			lrecl, err := parseDCB(value)
			if err != nil {
				return DD{}, err
			}
			dd.LRECL = lrecl
		case "SYSOUT":
			dd.Sysout = true
		default:
			return DD{}, fmt.Errorf("DD %s: unknown operand %q", s.name, key)
		}
	}
	return dd, dd.check()
}

func (dd DD) check() error {
	switch {
	case dd.Sysout && dd.DSN != "":
		return fmt.Errorf("DD %s: both SYSOUT and DSN", dd.Name)
	case dd.Sysout:
		return nil
	case dd.DSN == "":
		return fmt.Errorf("DD %s: no DSN", dd.Name)
	case dd.Disp == New && dd.LRECL == 0:
		return fmt.Errorf("DD %s: a new dataset needs DCB=(RECFM=FB,LRECL=n)", dd.Name)
	case dd.Disp != Shr && dd.Disp != Old && dd.Disp != New:
		return fmt.Errorf("DD %s: DISP=%s is not SHR, OLD or NEW", dd.Name, dd.Disp)
	}
	return nil
}

// parseDCB reads the record length from DCB=(RECFM=FB,LRECL=n). Pangaea's
// datasets are all fixed-length records without delimiters.
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
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return 0, fmt.Errorf("DCB: LRECL=%s is not a record length", v)
			}
			lrecl = n
		default:
			return 0, fmt.Errorf("DCB: unknown attribute %q", key)
		}
	}
	return lrecl, nil
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
