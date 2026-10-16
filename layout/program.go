package layout

import "github.com/vlumi/copal/cobol"

// ProgramRecords are a program's records laid out: those of each file, in
// the order the file section describes the files, and those of working
// storage.
type ProgramRecords struct {
	Files          []FileRecords
	WorkingStorage []*Field
}

// FileRecords are the records of one file's FD or SD.
type FileRecords struct {
	File    string
	Records []*Field
}

// Program lays out a program's records.
func Program(p *cobol.Program) (*ProgramRecords, error) {
	r := &ProgramRecords{}
	for _, fd := range p.FileSection {
		records, err := Records(fd.Entries)
		if err != nil {
			return nil, err
		}
		r.Files = append(r.Files, FileRecords{File: fd.Name, Records: records})
	}
	var err error
	r.WorkingStorage, err = Records(p.WorkingStorage)
	return r, err
}
