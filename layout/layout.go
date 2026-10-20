// Package layout computes record layouts from data description entries:
// every field's position, length, picture and usage, how it repeats and
// what it redefines, and the condition names on it.
package layout

import (
	"fmt"

	"github.com/vlumi/cobolith/cobol"
)

// Field is a group or elementary item of a record.
type Field struct {
	Level    int
	Name     string
	Start    int // 1-based position in the record, of the first occurrence
	Length   int // bytes of one occurrence
	Picture  string
	Usage    string
	Category string
	// Digits, Scale and Signed describe numeric items: how many digits,
	// how many of them after the implied decimal point, and whether signed.
	Digits int
	Scale  int
	Signed bool
	// Sign is where a signed display number's sign is, LEADING or
	// TRAILING, and SignSeparate whether in a byte of its own, as its
	// SIGN clause says; without one, it is overpunched on the last digit.
	Sign         string
	SignSeparate bool
	// Synchronized is whether the item is aligned on its natural boundary,
	// slack bytes before it, as its SYNCHRONIZED clause asks.
	Synchronized bool
	Redefines    string
	// Occurs is the number of occurrences, the maximum when it depends on
	// another field; 0 for a field that does not repeat.
	Occurs      int
	OccursMin   int
	DependingOn string
	IndexedBy   []string
	Values      []string
	Comment     []string
	Conditions  []Condition
	Children    []*Field
	Pos         cobol.Pos
}

// Condition is a level-88 condition name and the values that make it true.
type Condition struct {
	Name   string
	Values []string
	Pos    cobol.Pos
}

// Size is the bytes the field takes in its parent, all occurrences included.
func (f *Field) Size() int {
	return f.Length * max(f.Occurs, 1)
}

// Records builds the records described by the entries, in order: one for
// each level-01 and level-77 entry, with the entries below it as fields,
// laid out as the dialect lays them out. A level-78 entry is a constant,
// which takes no storage.
func Records(entries []*cobol.DataEntry, dialect cobol.Dialect) ([]*Field, error) {
	records, err := buildTree(entries)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if err := place(record, 1, "DISPLAY", sign{}, dialect); err != nil {
			return nil, err
		}
	}
	record := -1
	for _, e := range entries {
		if e.Level == 1 || e.Level == 77 {
			record++
		}
		if e.Level == 66 {
			if err := rename(records[record], e); err != nil {
				return nil, err
			}
		}
	}
	return records, nil
}

// rename adds a level-66 entry to its record: a field over the bytes of
// the field it renames, or of the first through the last of those it
// renames together, after the record's own fields. It is a group when it
// spans more than one, or renames one.
func rename(record *Field, e *cobol.DataEntry) error {
	from, thru := named(record, e.Renames), named(record, e.Renames)
	if e.RenamesThru != "" {
		thru = named(record, e.RenamesThru)
	}
	if from == nil || thru == nil || thru.Start < from.Start {
		return fmt.Errorf("%s: %s renames %s THRU %s, which are not fields of %s in that order", e.Pos, e.Name, e.Renames, e.RenamesThru, record.Name)
	}
	f := &Field{Level: 66, Name: e.Name, Start: from.Start, Length: thru.Start + thru.Size() - from.Start, Comment: e.Comment, Pos: e.Pos}
	if from == thru && len(from.Children) == 0 {
		f.Picture, f.Usage, f.Category = from.Picture, from.Usage, from.Category
		f.Digits, f.Scale, f.Signed, f.Sign, f.SignSeparate = from.Digits, from.Scale, from.Signed, from.Sign, from.SignSeparate
	} else {
		f.Category, f.Usage = Group, "DISPLAY"
	}
	record.Children = append(record.Children, f)
	return nil
}

// named finds a field of a record by name, at any level.
func named(f *Field, name string) *Field {
	if f.Name == name {
		return f
	}
	for _, child := range f.Children {
		if found := named(child, name); found != nil {
			return found
		}
	}
	return nil
}

// buildTree nests the entries by level number.
func buildTree(entries []*cobol.DataEntry) ([]*Field, error) {
	var records, open []*Field
	for _, e := range entries {
		switch {
		case e.Level == 88:
			if len(open) == 0 {
				return nil, fmt.Errorf("%s: condition %s has no field", e.Pos, e.Name)
			}
			f := open[len(open)-1]
			f.Conditions = append(f.Conditions, Condition{Name: e.Name, Values: e.Values, Pos: e.Pos})
		case e.Level == 78, e.Level == 66:
		case e.Level == 1 || e.Level == 77:
			f := newField(e)
			records = append(records, f)
			open = []*Field{f}
		case e.Level >= 2 && e.Level <= 49:
			for len(open) > 0 && open[len(open)-1].Level >= e.Level {
				open = open[:len(open)-1]
			}
			if len(open) == 0 {
				return nil, fmt.Errorf("%s: level %02d %s is not under a record", e.Pos, e.Level, e.Name)
			}
			f := newField(e)
			parent := open[len(open)-1]
			parent.Children = append(parent.Children, f)
			open = append(open, f)
		default:
			return nil, fmt.Errorf("%s: level %d is not supported yet", e.Pos, e.Level)
		}
	}
	return records, nil
}

func newField(e *cobol.DataEntry) *Field {
	return &Field{
		Level:        e.Level,
		Name:         e.Name,
		Picture:      e.Picture,
		Usage:        e.Usage,
		Sign:         e.Sign,
		SignSeparate: e.SignSeparate,
		Synchronized: e.Synchronized,
		Redefines:    e.Redefines,
		Occurs:       e.Occurs,
		OccursMin:    e.OccursMin,
		DependingOn:  e.DependingOn,
		IndexedBy:    e.IndexedBy,
		Values:       e.Values,
		Comment:      e.Comment,
		Pos:          e.Pos,
	}
}

// sign is where a SIGN clause puts a signed display number's sign, and
// whether in a byte of its own.
type sign struct {
	position string
	separate bool
}

// place sets the position and length of f, starting at start, and of
// everything under it. A field without a usage takes its group's, and a
// signed display number without a SIGN clause its group's SIGN.
func place(f *Field, start int, usage string, groupSign sign, dialect cobol.Dialect) error {
	f.Start = start
	if f.Usage == "" {
		f.Usage = usage
	}
	if len(f.Children) == 0 {
		if err := f.placeElementary(groupSign, dialect); err != nil {
			return err
		}
		return f.synchronize(dialect)
	}
	if f.Picture != "" {
		return fmt.Errorf("%s: group %s has a picture", f.Pos, f.Name)
	}
	if f.Synchronized {
		return fmt.Errorf("%s: group %s: SYNCHRONIZED is for elementary items", f.Pos, f.Name)
	}
	if f.Sign != "" {
		groupSign = sign{f.Sign, f.SignSeparate}
		f.Sign, f.SignSeparate = "", false
	}
	f.Category = Group
	next, end := start, start
	for i, child := range f.Children {
		childStart := next
		if child.Redefines != "" {
			target := redefined(f.Children[:i], child.Redefines)
			if target == nil {
				return fmt.Errorf("%s: %s redefines %s, which is not an earlier field at its level", child.Pos, child.Name, child.Redefines)
			}
			childStart = target.Start
		}
		if err := place(child, childStart, f.Usage, groupSign, dialect); err != nil {
			return err
		}
		if child.Redefines == "" {
			next = child.Start + child.Size()
		}
		end = max(end, child.Start+child.Size())
	}
	f.Length = end - start
	if m := largestAlignment(f); f.Occurs > 0 && m > 0 && f.Length%m != 0 {
		f.Length += m - f.Length%m
	}
	return nil
}

// alignment is the boundary a synchronized item is aligned on, from the
// start of its record, in IBM COBOL: 2 bytes for a binary item of four
// digits or fewer, 4 for one of more and for COMP-1 and an index, 8 for
// COMP-2; 0, none, for an item not synchronized, and for display and
// packed decimal, which SYNCHRONIZED does not move.
func alignment(f *Field) int {
	if !f.Synchronized {
		return 0
	}
	switch f.Usage {
	case "COMP", "COMP-5":
		if f.Digits <= 4 {
			return 2
		}
		return 4
	case "COMP-1", "INDEX":
		return 4
	case "COMP-2":
		return 8
	}
	return 0
}

// largestAlignment is the largest alignment of the items in a group: each
// occurrence of a table of it is padded to a multiple of it, so that the
// items of the next are aligned too.
func largestAlignment(f *Field) int {
	m := alignment(f)
	for _, child := range f.Children {
		m = max(m, largestAlignment(child))
	}
	return m
}

// synchronize puts slack bytes before a synchronized item, as many as
// align it on its boundary from the start of its record.
func (f *Field) synchronize(dialect cobol.Dialect) error {
	if !f.Synchronized {
		return nil
	}
	if !dialect.Synchronizes {
		return fmt.Errorf("%s: %s: in %s COBOL, where SYNCHRONIZED puts an item depends on compiler directives", f.Pos, f.Name, dialect.Title)
	}
	if m := alignment(f); m > 0 && (f.Start-1)%m != 0 {
		f.Start += m - (f.Start-1)%m
	}
	return nil
}

func (f *Field) placeElementary(groupSign sign, dialect cobol.Dialect) error {
	if f.Usage == "INDEX" {
		switch {
		case f.Picture != "":
			return fmt.Errorf("%s: %s: USAGE INDEX takes no picture", f.Pos, f.Name)
		case dialect.IndexSize == 0:
			return fmt.Errorf("%s: %s: the size of USAGE INDEX in %s COBOL depends on more than the dialect", f.Pos, f.Name, dialect.Title)
		}
		f.Length, f.Category = dialect.IndexSize, Index
		return nil
	}
	if size, ok := floatSizes[f.Usage]; ok {
		if f.Picture != "" {
			return fmt.Errorf("%s: %s: %s takes no picture", f.Pos, f.Name, f.Usage)
		}
		f.Length, f.Category, f.Signed = size, Numeric, true
		return nil
	}
	if f.Picture == "" {
		return fmt.Errorf("%s: elementary %s has no picture", f.Pos, f.Name)
	}
	p, err := parsePicture(f.Picture)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", f.Pos, f.Name, err)
	}
	f.Length, err = storageSize(p, f.Usage, dialect)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", f.Pos, f.Name, err)
	}
	f.Category = p.category
	if p.category == Numeric {
		f.Digits, f.Scale, f.Signed = p.digits, p.scale, p.signed
	}
	if f.Sign == "" && f.Signed && f.Usage == "DISPLAY" {
		f.Sign, f.SignSeparate = groupSign.position, groupSign.separate
	}
	if f.Sign != "" {
		if !f.Signed || f.Usage != "DISPLAY" {
			return fmt.Errorf("%s: %s: SIGN on an item that is not a signed display number", f.Pos, f.Name)
		}
		if f.SignSeparate {
			f.Length++
		}
	}
	return nil
}

// floatSizes are the sizes of the floating-point usages, which take no
// picture: single and double precision, the same in every dialect.
var floatSizes = map[string]int{"COMP-1": 4, "COMP-2": 8}

func redefined(siblings []*Field, name string) *Field {
	for _, s := range siblings {
		if s.Name == name {
			return s
		}
	}
	return nil
}
