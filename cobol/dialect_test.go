package cobol

import "testing"

func TestBinarySize(t *testing.T) {
	for _, c := range []struct {
		dialect Dialect
		digits  int
		size    int
	}{
		{IBM, 1, 2}, {IBM, 4, 2}, {IBM, 5, 4}, {IBM, 9, 4}, {IBM, 10, 8}, {IBM, 18, 8},
		{MicroFocus, 1, 1}, {MicroFocus, 2, 1}, {MicroFocus, 4, 2}, {MicroFocus, 6, 3},
		{MicroFocus, 9, 4}, {MicroFocus, 11, 5}, {MicroFocus, 18, 8},
	} {
		if size, ok := c.dialect.BinarySize(c.digits); !ok || size != c.size {
			t.Errorf("%s, %d digits: %d bytes, want %d", c.dialect.Name, c.digits, size, c.size)
		}
	}
	if _, ok := IBM.BinarySize(19); ok {
		t.Error("19 digits fit in a binary item")
	}
}
