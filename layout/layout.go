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
		if err := place(record, 1, "DISPLAY", dialect); err != nil {
			return nil, err
		}
	}
	return records, nil
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
		case e.Level == 78:
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

// place sets the position and length of f, starting at start, and of
// everything under it. A field without a usage takes its group's.
func place(f *Field, start int, usage string, dialect cobol.Dialect) error {
	f.Start = start
	if f.Usage == "" {
		f.Usage = usage
	}
	if len(f.Children) == 0 {
		return f.placeElementary(dialect)
	}
	if f.Picture != "" {
		return fmt.Errorf("%s: group %s has a picture", f.Pos, f.Name)
	}
	if f.Sign != "" {
		return fmt.Errorf("%s: group %s: SIGN on a group is not supported yet", f.Pos, f.Name)
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
		if err := place(child, childStart, f.Usage, dialect); err != nil {
			return err
		}
		if child.Redefines == "" {
			next = childStart + child.Size()
		}
		end = max(end, childStart+child.Size())
	}
	f.Length = end - start
	return nil
}

func (f *Field) placeElementary(dialect cobol.Dialect) error {
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

func redefined(siblings []*Field, name string) *Field {
	for _, s := range siblings {
		if s.Name == name {
			return s
		}
	}
	return nil
}
