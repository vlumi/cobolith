// Package record reads and writes the fields of fixed-length records by
// name, using the record layouts computed from the copybooks.
//
// Numbers are integers in units of the field's scale: 1,520.00 in a field
// with two decimals is 152000. Signed display numbers carry their sign
// overpunched on the last digit, EBCDIC style: {ABCDEFGHI for +0 to +9,
// }JKLMNOPQR for -0 to -9. Packed decimal (COMP-3) numbers hold two digits
// a byte and the sign in the last half-byte: C positive, D negative, F
// unsigned. Binary numbers (COMP, COMP-4, BINARY) are big-endian, in two's
// complement when signed; native binary (COMP-5) is in the dialect's
// native byte order. Floating point (COMP-1, COMP-2) is read and written
// as such, by Float and SetFloat.
package record

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/vlumi/cobolith/cobol"
	"github.com/vlumi/cobolith/layout"
)

// Layout is one record's layout, with its fields by name, and the dialect
// it is laid out in.
type Layout struct {
	Name    string
	Length  int
	root    *layout.Field
	fields  map[string]*layout.Field
	dialect cobol.Dialect
}

// FromCopybook reads the layout of the named record in a copybook, written
// in the dialect.
func FromCopybook(path string, copybookDirs []string, name string, dialect cobol.Dialect) (*Layout, error) {
	entries, err := cobol.ReadDataEntries(path, copybookDirs)
	if err != nil {
		return nil, err
	}
	records, err := layout.Records(entries, dialect)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.Name == name {
			return New(r, dialect), nil
		}
	}
	return nil, fmt.Errorf("%s: no record %s", path, name)
}

// New indexes the fields of a record laid out in a dialect by name. A name
// used more than once, such as FILLER, cannot be looked up.
func New(root *layout.Field, dialect cobol.Dialect) *Layout {
	l := &Layout{Name: root.Name, Length: root.Length, root: root, fields: map[string]*layout.Field{}, dialect: dialect}
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
	switch f.Usage {
	case "COMP-3":
		return unpack(name, b)
	case "COMP", "COMP-5":
		order, err := l.binaryOrder(f)
		if err != nil {
			return 0, err
		}
		return unbinary(b, order, f.Signed), nil
	}
	return unzone(name, b, f)
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
	if len(digits) > f.Digits && f.Usage != "COMP-5" {
		return fmt.Errorf("%s: %d does not fit in %d digits", name, value, f.Digits)
	}
	switch f.Usage {
	case "COMP-3":
		pack(b, digits, f.Signed, value < 0)
	case "COMP", "COMP-5":
		order, err := l.binaryOrder(f)
		if err != nil {
			return err
		}
		if !fitsBinary(value, len(b), f.Signed) {
			return fmt.Errorf("%s: %d does not fit in %d bytes", name, value, len(b))
		}
		putBinary(b, order, value)
	default:
		zone(b, digits, f, value < 0)
	}
	return nil
}

// Float returns a floating-point field's value.
func (l *Layout) Float(r []byte, name string) (float64, error) {
	b, err := l.floating(r, name)
	if err != nil {
		return 0, err
	}
	if l.dialect.HexFloat {
		return unhex(b), nil
	}
	if len(b) == 4 {
		return float64(math.Float32frombits(l.dialect.NativeOrder.Uint32(b))), nil
	}
	return math.Float64frombits(l.dialect.NativeOrder.Uint64(b)), nil
}

// SetFloat writes a value into a floating-point field.
func (l *Layout) SetFloat(r []byte, name string, value float64) error {
	b, err := l.floating(r, name)
	if err != nil {
		return err
	}
	switch {
	case l.dialect.HexFloat:
		return hex(b, name, value)
	case len(b) == 4:
		l.dialect.NativeOrder.PutUint32(b, math.Float32bits(float32(value)))
	default:
		l.dialect.NativeOrder.PutUint64(b, math.Float64bits(value))
	}
	return nil
}

// floating finds the bytes of a floating-point field, in a dialect that
// says how they hold it.
func (l *Layout) floating(r []byte, name string) ([]byte, error) {
	f, err := l.field(name)
	if err != nil {
		return nil, err
	}
	switch {
	case f.Usage != "COMP-1" && f.Usage != "COMP-2":
		return nil, fmt.Errorf("%s: not a floating-point field", name)
	case !l.dialect.HexFloat && l.dialect.NativeOrder == nil:
		return nil, fmt.Errorf("%s: %s in %s COBOL is in the platform's byte order", name, f.Usage, l.dialect.Title)
	}
	return l.bytes(r, f)
}

// binaryOrder is the byte order of a binary field: big-endian, or for
// native binary the dialect's, if it has one.
func (l *Layout) binaryOrder(f *layout.Field) (binary.ByteOrder, error) {
	if f.Usage == "COMP" {
		return binary.BigEndian, nil
	}
	if l.dialect.NativeOrder == nil {
		return nil, fmt.Errorf("%s: COMP-5 in %s COBOL is in the platform's byte order", f.Name, l.dialect.Title)
	}
	return l.dialect.NativeOrder, nil
}

func (l *Layout) numeric(r []byte, name string) (*layout.Field, []byte, error) {
	f, err := l.field(name)
	if err != nil {
		return nil, nil, err
	}
	if f.Category != layout.Numeric || f.Usage == "COMP-1" || f.Usage == "COMP-2" {
		return nil, nil, fmt.Errorf("%s: not a number in display, packed decimal or binary", name)
	}
	b, err := l.bytes(r, f)
	return f, b, err
}

// unzone reads a display number, with its sign if it has one: in a byte
// of its own, or overpunched on its first or last digit, as its SIGN
// clause says.
func unzone(name string, b []byte, f *layout.Field) (int64, error) {
	sign, punched := int64(1), -1
	switch {
	case !f.Signed:
	case f.SignSeparate:
		at := separateSign(f, b)
		switch b[at] {
		case '-':
			sign = -1
		case '+':
		default:
			return 0, fmt.Errorf("%s: %q has no sign", name, b)
		}
		b = append(slices.Clone(b[:at]), b[at+1:]...)
	case f.Sign == "LEADING":
		punched = 0
	default:
		punched = len(b) - 1
	}
	var n int64
	for i, c := range b {
		digit := strings.IndexByte("0123456789", c)
		if i == punched {
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

// zone writes digits as a display number, with its sign where its SIGN
// clause says, overpunched on the last digit if it has none.
func zone(b []byte, digits string, f *layout.Field, negativeValue bool) {
	switch {
	case !f.Signed:
		copy(b, digits)
	case f.SignSeparate:
		at := separateSign(f, b)
		b[at] = '+'
		if negativeValue {
			b[at] = '-'
		}
		if at == 0 {
			copy(b[1:], digits)
		} else {
			copy(b, digits)
		}
	default:
		copy(b, digits)
		at := len(b) - 1
		if f.Sign == "LEADING" {
			at = 0
		}
		punch := positive
		if negativeValue {
			punch = negative
		}
		b[at] = punch[digits[at]-'0']
	}
}

// separateSign is the index of a number's separate sign byte.
func separateSign(f *layout.Field, b []byte) int {
	if f.Sign == "LEADING" {
		return 0
	}
	return len(b) - 1
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

// unbinary reads a binary number of any length up to eight bytes, in two's
// complement when signed.
func unbinary(b []byte, order binary.ByteOrder, signed bool) int64 {
	full := make([]byte, 8)
	if order == binary.BigEndian {
		copy(full[8-len(b):], b)
	} else {
		copy(full, b)
	}
	var n uint64
	if order == binary.BigEndian {
		n = binary.BigEndian.Uint64(full)
	} else {
		n = binary.LittleEndian.Uint64(full)
	}
	if bits := uint(8 * len(b)); signed && bits < 64 && n&(1<<(bits-1)) != 0 {
		n |= ^uint64(0) << bits
	}
	return int64(n)
}

// putBinary writes a number into the bytes of a binary field.
func putBinary(b []byte, order binary.ByteOrder, value int64) {
	full := make([]byte, 8)
	if order == binary.BigEndian {
		binary.BigEndian.PutUint64(full, uint64(value))
		copy(b, full[8-len(b):])
	} else {
		binary.LittleEndian.PutUint64(full, uint64(value))
		copy(b, full[:len(b)])
	}
}

// fitsBinary reports whether a number fits in so many bytes of binary.
func fitsBinary(value int64, size int, signed bool) bool {
	if size >= 8 {
		return signed || value >= 0
	}
	bits := uint(8 * size)
	if signed {
		return value >= -(1<<(bits-1)) && value < 1<<(bits-1)
	}
	return value >= 0 && value < 1<<bits
}

// unhex reads an IBM hexadecimal floating-point number: a sign bit, an
// exponent of 16 in seven bits, excess 64, and a fraction in the rest.
func unhex(b []byte) float64 {
	var fraction uint64
	for _, c := range b[1:] {
		fraction = fraction<<8 | uint64(c)
	}
	if fraction == 0 {
		return 0
	}
	value := float64(fraction) / math.Pow(2, float64(8*(len(b)-1))) * math.Pow(16, float64(int(b[0]&0x7F)-64))
	if b[0]&0x80 != 0 {
		return -value
	}
	return value
}

// hex writes an IBM hexadecimal floating-point number, normalized: the
// fraction's first hex digit not zero.
func hex(b []byte, name string, value float64) error {
	clear(b)
	if value == 0 {
		return nil
	}
	given := value
	sign := byte(0)
	if value < 0 {
		sign, value = 0x80, -value
	}
	fractionBits := 8 * (len(b) - 1)
	exponent := 0
	for value >= 1 {
		value /= 16
		exponent++
	}
	for value < 1.0/16 {
		value *= 16
		exponent--
	}
	fraction := uint64(math.Round(value * math.Pow(2, float64(fractionBits))))
	if fraction == 1<<fractionBits {
		fraction >>= 4
		exponent++
	}
	if exponent+64 < 0 || exponent+64 > 127 {
		return fmt.Errorf("%s: %g is out of the range of hexadecimal floating point", name, given)
	}
	b[0] = sign | byte(exponent+64)
	for i := len(b) - 1; i >= 1; i-- {
		b[i] = byte(fraction)
		fraction >>= 8
	}
	return nil
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
