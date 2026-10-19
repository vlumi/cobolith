package layout

import (
	"slices"

	"github.com/vlumi/copal/bedrock/cobol"
)

// ProgramRecords are a program's records laid out: those of each file, in
// the order the file section describes the files, those of working
// storage, and those a calling program passes, in the linkage section.
type ProgramRecords struct {
	Files          []FileRecords
	WorkingStorage []*Field
	Linkage        []*Field
}

// All lists every record: working storage, linkage, then the files'.
func (r *ProgramRecords) All() []*Field {
	all := append(slices.Clone(r.WorkingStorage), r.Linkage...)
	for _, f := range r.Files {
		all = append(all, f.Records...)
	}
	return all
}

// FileRecords are the records of one file's FD or SD.
type FileRecords struct {
	File    string
	Records []*Field
}

// Program lays out a program's records, written in the dialect.
func Program(p *cobol.Program, dialect cobol.Dialect) (*ProgramRecords, error) {
	r := &ProgramRecords{}
	for _, fd := range p.FileSection {
		records, err := Records(fd.Entries, dialect)
		if err != nil {
			return nil, err
		}
		r.Files = append(r.Files, FileRecords{File: fd.Name, Records: records})
	}
	var err error
	if r.WorkingStorage, err = Records(p.WorkingStorage, dialect); err != nil {
		return nil, err
	}
	r.Linkage, err = Records(p.Linkage, dialect)
	return r, err
}
