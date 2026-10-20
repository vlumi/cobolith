package cobol

import (
	"encoding/binary"
	"slices"
)

// Dialect is the COBOL a system is written in, by the compiler it was
// written for: its name, short and as a reader knows it, and what the
// language leaves to the compiler that bears on how its records are laid
// out.
type Dialect struct {
	Name, Title string
	// BinarySizes are the sizes, in bytes, a binary item can take, of
	// which it takes the smallest that holds its digits.
	BinarySizes []int
	// IndexSize is the size, in bytes, of an item of USAGE INDEX; 0 where
	// it depends on more than the dialect.
	IndexSize int
	// NativeOrder is the byte order of native binary, COMP-5, and of
	// IEEE floating point; nil where it is the platform's. COMP, COMP-4
	// and BINARY are big-endian in every dialect.
	NativeOrder binary.ByteOrder
	// HexFloat is whether COMP-1 and COMP-2 hold IBM hexadecimal floating
	// point; otherwise they hold IEEE 754, in the native byte order.
	HexFloat bool
	// Synchronizes is whether the dialect's rules say where SYNCHRONIZED
	// items go; otherwise compiler directives do.
	Synchronizes bool
	// Encoding is the encoding the source and its data are in, where it
	// is not ASCII or UTF-8: "Shift_JIS" for the Japanese compilers,
	// whose two-byte characters the lexer must not split or upper-case.
	Encoding string
	// NationalSize is the bytes one character of a national item, PIC N,
	// takes; 0 where the dialect's rules are not known.
	NationalSize int
}

var (
	// IBM is an IBM compiler's COBOL: a binary item is a halfword, a
	// fullword or a doubleword, and an index a fullword; native binary is
	// big-endian, as the mainframe is, and floating point hexadecimal.
	IBM = Dialect{Name: "ibm", Title: "IBM", BinarySizes: []int{2, 4, 8}, IndexSize: 4,
		NativeOrder: binary.BigEndian, HexFloat: true, Synchronizes: true}
	// MicroFocus is Micro Focus COBOL: a binary item takes as many bytes
	// as its digits need; native binary and floating point are in the
	// byte order of the platform it runs on.
	MicroFocus = Dialect{Name: "mf", Title: "Micro Focus", BinarySizes: []int{1, 2, 3, 4, 5, 6, 7, 8}}
)

// Hitachi is Hitachi COBOL2002: Japanese characters in Shift_JIS, two
// bytes each in the source and in a national item; a binary item a
// halfword, fullword or doubleword, as on IBM; floating point in IEEE;
// and native binary's byte order the platform's, so not known. Its
// index items' size and its SYNCHRONIZED boundaries are not read yet.
var Hitachi = Dialect{Name: "hitachi", Title: "Hitachi", BinarySizes: []int{2, 4, 8},
	Encoding: ShiftJIS, NationalSize: 2}

// Dialects are the dialects the library reads.
var Dialects = []Dialect{IBM, MicroFocus, Hitachi}

// ShiftJIS is the Encoding of the Japanese compilers' sources.
const ShiftJIS = "Shift_JIS"

// leadsShiftJIS reports whether a byte starts a two-byte Shift_JIS
// character, whose second byte is 0x40 to 0x7E or 0x80 to 0xFC.
func leadsShiftJIS(c byte) bool {
	return c >= 0x81 && c <= 0x9F || c >= 0xE0 && c <= 0xFC
}

// DialectNamed finds a dialect by its name.
func DialectNamed(name string) (Dialect, bool) {
	i := slices.IndexFunc(Dialects, func(d Dialect) bool { return d.Name == name })
	if i < 0 {
		return Dialect{}, false
	}
	return Dialects[i], true
}

// BinarySize is the bytes a binary item of so many digits takes, false if
// none of the dialect's sizes holds them.
func (d Dialect) BinarySize(digits int) (int, bool) {
	for _, size := range d.BinarySizes {
		if digits <= binaryDigits(size) {
			return size, true
		}
	}
	return 0, false
}

// binaryDigits is how many decimal digits a signed binary item of so many
// bytes holds in full: 2 in one byte, 4 in two, 9 in four.
func binaryDigits(size int) int {
	digits := 0
	for limit := uint64(1) << (8*size - 1); limit >= 10; limit /= 10 {
		digits++
	}
	return digits
}
