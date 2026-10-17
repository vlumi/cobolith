// Package record reads and writes the fields of fixed-length records by
// name, using the record layouts Copal computes from the copybooks.
//
// Numbers are integers in units of the field's scale: 1,520.00 in a field
// with two decimals is 152000. Signed display numbers carry their sign
// overpunched on the last digit, EBCDIC style: {ABCDEFGHI for +0 to +9,
// }JKLMNOPQR for -0 to -9. Packed decimal (COMP-3) numbers hold two digits
// a byte and the sign in the last half-byte: C positive, D negative, F
// unsigned.
package record

import (
	"fmt"
	"strings"

	"github.com/vlumi/copal/bedrock/cobol"
	"github.com/vlumi/copal/bedrock/layout"
)

// Layout is one record's layout, with its fields by name.
type Layout struct {
	Name   string
	Length int
	root   *layout.Field
	fields map[string]*layout.Field
}

// FromCopybook reads the layout of the named record in a copybook.
func FromCopybook(path string, copybookDirs []string, name string) (*Layout, error) {
	entries, err := cobol.ReadDataEntries(path, copybookDirs)
	if err != nil {
		return nil, err
	}
	records, err := layout.Records(entries)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.Name == name {
			return New(r), nil
		}
	}
	return nil, fmt.Errorf("%s: no record %s", path, name)
}

// New indexes a record layout's fields by name. A name used more than once,
// such as FILLER, cannot be looked up.
func New(root *layout.Field) *Layout {
	l := &Layout{Name: root.Name, Length: root.Length, root: root, fields: map[string]*layout.Field{}}
	seen := map[string]int{}
	var index func(f *layout.Field)
	index = func(f *layout.Field) {
		seen[f.Name]++
		l.fields[f.Name] = f
		for _, child := range f.Children {
			index(child)
		}
	}
	index(root)
	for name, n := range seen {
		if n > 1 {
			delete(l.fields, name)
		}
	}
	return l
}

// Fields lists the record's elementary fields in the order they are
// declared, those of every redefinition included.
func (l *Layout) Fields() []*layout.Field {
	var fields []*layout.Field
	var walk func(f *layout.Field)
	walk = func(f *layout.Field) {
		if len(f.Children) == 0 {
			fields = append(fields, f)
		}
		for _, child := range f.Children {
			walk(child)
		}
	}
	walk(l.root)
	return fields
}

// Blank returns a record of spaces.
func (l *Layout) Blank() []byte {
	return []byte(strings.Repeat(" ", l.Length))
}

func (l *Layout) field(name string) (*layout.Field, error) {
	f, ok := l.fields[name]
	if !ok {
		return nil, fmt.Errorf("%s: no field %s", l.Name, name)
	}
	return f, nil
}

func (l *Layout) bytes(r []byte, f *layout.Field) ([]byte, error) {
	if len(r) != l.Length {
		return nil, fmt.Errorf("%s: record of %d bytes, want %d", l.Name, len(r), l.Length)
	}
	return r[f.Start-1 : f.Start-1+f.Length], nil
}

// Text returns a field's contents with trailing spaces removed.
func (l *Layout) Text(r []byte, name string) (string, error) {
	f, err := l.field(name)
	if err != nil {
		return "", err
	}
	b, err := l.bytes(r, f)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(b), " "), nil
}

// SetText writes text into a field, padded with spaces.
func (l *Layout) SetText(r []byte, name, text string) error {
	f, err := l.field(name)
	if err != nil {
		return err
	}
	b, err := l.bytes(r, f)
	if err != nil {
		return err
	}
	if len(text) > len(b) {
		return fmt.Errorf("%s: %q is longer than %d", name, text, len(b))
	}
	copy(b, text+strings.Repeat(" ", len(b)-len(text)))
	return nil
}

const (
	positive = "{ABCDEFGHI"
	negative = "}JKLMNOPQR"
)

// Number returns a numeric field's value in units of its scale.
func (l *Layout) Number(r []byte, name string) (int64, error) {
	f, b, err := l.numeric(r, name)
	if err != nil {
		return 0, err
	}
	if f.Usage == "COMP-3" {
		return unpack(name, b)
	}
	return unzone(name, b, f.Signed)
}

// SetNumber writes a value, in units of the field's scale, into a numeric
// field.
func (l *Layout) SetNumber(r []byte, name string, value int64) error {
	f, b, err := l.numeric(r, name)
	if err != nil {
		return err
	}
	if value < 0 && !f.Signed {
		return fmt.Errorf("%s: %d in an unsigned field", name, value)
	}
	digits := fmt.Sprintf("%0*d", f.Digits, abs(value))
	if len(digits) > f.Digits {
		return fmt.Errorf("%s: %d does not fit in %d digits", name, value, f.Digits)
	}
	if f.Usage == "COMP-3" {
		pack(b, digits, f.Signed, value < 0)
	} else {
		zone(b, digits, f.Signed, value < 0)
	}
	return nil
}

func (l *Layout) numeric(r []byte, name string) (*layout.Field, []byte, error) {
	f, err := l.field(name)
	if err != nil {
		return nil, nil, err
	}
	if f.Category != layout.Numeric || f.Usage != "DISPLAY" && f.Usage != "COMP-3" {
		return nil, nil, fmt.Errorf("%s: not a numeric display or packed decimal field", name)
	}
	b, err := l.bytes(r, f)
	return f, b, err
}

// unzone reads a display number, its sign overpunched if it has one.
func unzone(name string, b []byte, signed bool) (int64, error) {
	var n int64
	sign := int64(1)
	for i, c := range b {
		digit := strings.IndexByte("0123456789", c)
		if i == len(b)-1 && signed {
			if d := strings.IndexByte(positive, c); d >= 0 {
				digit = d
			} else if d := strings.IndexByte(negative, c); d >= 0 {
				digit, sign = d, -1
			}
		}
		if digit < 0 {
			return 0, fmt.Errorf("%s: %q is not a number", name, b)
		}
		n = n*10 + int64(digit)
	}
	return sign * n, nil
}

// zone writes digits as a display number, overpunching the sign.
func zone(b []byte, digits string, signed, negativeValue bool) {
	copy(b, digits)
	if signed {
		last := digits[len(digits)-1] - '0'
		if negativeValue {
			b[len(b)-1] = negative[last]
		} else {
			b[len(b)-1] = positive[last]
		}
	}
}

// unpack reads a packed decimal number.
func unpack(name string, b []byte) (int64, error) {
	var n int64
	for i, c := range b {
		nibbles := []byte{c >> 4, c & 0x0F}
		if i == len(b)-1 {
			nibbles = nibbles[:1]
		}
		for _, d := range nibbles {
			if d > 9 {
				return 0, fmt.Errorf("%s: % X is not packed decimal", name, b)
			}
			n = n*10 + int64(d)
		}
	}
	switch b[len(b)-1] & 0x0F {
	case 0x0C, 0x0F:
		return n, nil
	case 0x0D:
		return -n, nil
	}
	return 0, fmt.Errorf("%s: % X has no sign", name, b)
}

// pack writes digits as a packed decimal number. A field's bytes hold an
// odd number of digits; an even number of digits gets a leading zero.
func pack(b []byte, digits string, signed, negativeValue bool) {
	nibbles := strings.Repeat("0", 2*len(b)-1-len(digits)) + digits
	sign := byte(0x0F)
	if signed {
		sign = 0x0C
		if negativeValue {
			sign = 0x0D
		}
	}
	for i := range b {
		high := nibbles[2*i] - '0'
		low := sign
		if 2*i+1 < len(nibbles) {
			low = nibbles[2*i+1] - '0'
		}
		b[i] = high<<4 | low
	}
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
