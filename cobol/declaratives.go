package cobol

import "slices"

// ioVerbs are the verbs whose failure on a file runs its declaratives.
var ioVerbs = map[string]bool{
	"OPEN": true, "CLOSE": true, "READ": true, "WRITE": true, "REWRITE": true, "DELETE": true, "START": true,
}

// ErrorProcedures lists the sections of the DECLARATIVES that run when a
// statement fails on a file: those for a file it names, or whose record it
// writes, and those for a mode the program opens that file in.
func (p *Program) ErrorProcedures(s Statement) []string {
	if !ioVerbs[s.Verb] || len(p.Declaratives) == 0 {
		return nil
	}
	var sections []string
	for _, file := range p.statementFiles(s) {
		modes := p.openModes(file)
		for _, d := range p.Declaratives {
			covers := slices.Contains(d.Files, file) || slices.ContainsFunc(d.Modes, func(m string) bool { return slices.Contains(modes, m) })
			if covers && !d.Debugging && !slices.Contains(sections, d.Section) {
				sections = append(sections, d.Section)
			}
		}
	}
	return sections
}

// statementFiles lists the files a statement uses: those it names, and
// for WRITE and REWRITE, which name a record, the file the record is of.
func (p *Program) statementFiles(s Statement) []string {
	var files []string
	for _, f := range s.Files {
		files = append(files, f.Name)
	}
	if s.Verb == "WRITE" || s.Verb == "REWRITE" {
		for _, w := range s.Writes {
			if file, ok := p.recordFile(w.Name); ok {
				files = append(files, file)
			}
		}
	}
	return files
}

// recordFile is the file whose FD describes a record.
func (p *Program) recordFile(record string) (string, bool) {
	for _, fd := range p.FileSection {
		if slices.ContainsFunc(fd.Entries, func(e *DataEntry) bool { return e.Level == 1 && e.Name == record }) {
			return fd.Name, true
		}
	}
	return "", false
}

// openModes lists the modes the program opens a file in.
func (p *Program) openModes(file string) []string {
	var modes []string
	for _, para := range p.Paragraphs {
		Walk(para.Statements, func(s Statement) {
			if s.Verb != "OPEN" {
				return
			}
			for _, f := range s.Files {
				if f.Name == file && !slices.Contains(modes, f.Mode) {
					modes = append(modes, f.Mode)
				}
			}
		})
	}
	return modes
}
