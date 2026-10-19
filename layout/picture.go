package layout

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vlumi/copal/bedrock/cobol"
)

// Categories of elementary items, by their picture.
const (
	Alphanumeric       = "alphanumeric"
	AlphanumericEdited = "alphanumeric-edited"
	Numeric            = "numeric"
	NumericEdited      = "numeric-edited"
	Group              = "group"
)

// picture is what a PICTURE string says about an item's storage.
type picture struct {
	positions int // character positions in display form
	digits    int
	scale     int // digits after the implied decimal point
	signed    bool
	category  string
}

// parsePicture analyzes a picture string such as S9(7)V99, X(20) or
// Z,ZZ9.99-. The scaling position P is not supported.
func parsePicture(s string) (picture, error) {
	symbols, err := expandPicture(s)
	if err != nil {
		return picture{}, err
	}
	var p picture
	alpha, edited, decimal := false, false, false
	for i := 0; i < len(symbols); i++ {
		switch c := symbols[i]; c {
		case 'S':
			if i != 0 {
				return picture{}, fmt.Errorf("PIC %s: S must come first", s)
			}
			p.signed = true
		case 'V':
			decimal = true
		case '9':
			p.positions++
			p.digits++
			if decimal {
				p.scale++
			}
		case 'X', 'A':
			p.positions++
			alpha = true
		case 'Z', '*', '+', '-', '.', ',', 'B', '0', '/', '$':
			p.positions++
			edited = true
		case 'C', 'D':
			if pair := symbols[i:min(i+2, len(symbols))]; pair != "CR" && pair != "DB" {
				return picture{}, fmt.Errorf("PIC %s: %c is not part of CR or DB", s, c)
			}
			p.positions += 2
			edited = true
			i++
		default:
			return picture{}, fmt.Errorf("PIC %s: symbol %c is not supported yet", s, c)
		}
	}
	switch {
	case alpha && edited:
		p.category = AlphanumericEdited
	case alpha:
		p.category = Alphanumeric
	case edited:
		p.category = NumericEdited
	default:
		p.category = Numeric
	}
	if p.positions == 0 {
		return picture{}, fmt.Errorf("PIC %s: no character positions", s)
	}
	return p, nil
}

// expandPicture writes out repetitions: 9(3)V99 becomes 999V99.
func expandPicture(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '(' {
			b.WriteByte(s[i])
			continue
		}
		end := strings.IndexByte(s[i:], ')')
		if end < 0 || b.Len() == 0 {
			return "", fmt.Errorf("PIC %s: bad repetition", s)
		}
		n, err := strconv.Atoi(s[i+1 : i+end])
		if err != nil || n < 1 {
			return "", fmt.Errorf("PIC %s: bad repetition count", s)
		}
		last := b.String()[b.Len()-1]
		b.WriteString(strings.Repeat(string(last), n-1))
		i += end
	}
	return b.String(), nil
}

// storageSize is the number of bytes an elementary item takes with the
// given usage, in the dialect.
func storageSize(p picture, usage string, dialect cobol.Dialect) (int, error) {
	switch usage {
	case "DISPLAY":
		return p.positions, nil
	case "COMP-3":
		if p.category != Numeric {
			return 0, fmt.Errorf("COMP-3 needs a numeric picture")
		}
		return p.digits/2 + 1, nil
	case "COMP":
		if p.category != Numeric {
			return 0, fmt.Errorf("COMP needs a numeric picture")
		}
		if size, ok := dialect.BinarySize(p.digits); ok {
			return size, nil
		}
		return 0, fmt.Errorf("COMP with %d digits", p.digits)
	}
	return 0, fmt.Errorf("usage %s is not supported", usage)
}
