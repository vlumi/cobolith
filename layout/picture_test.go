package layout

import "testing"

func TestParsePicture(t *testing.T) {
	tests := []struct {
		pic  string
		want picture
	}{
		{"X(20)", picture{positions: 20, category: Alphanumeric}},
		{"9(10)", picture{positions: 10, digits: 10, category: Numeric}},
		{"S9(11)V99", picture{positions: 13, digits: 13, scale: 2, signed: true, category: Numeric}},
		{"ZZ,ZZZ,ZZ9", picture{positions: 10, digits: 1, category: NumericEdited}},
		{"Z,ZZZ,ZZZ,ZZZ,ZZ9.99-", picture{positions: 21, digits: 3, category: NumericEdited}},
		{"9(5)CR", picture{positions: 7, digits: 5, category: NumericEdited}},
		{"XXBXX", picture{positions: 5, category: AlphanumericEdited}},
	}
	for _, test := range tests {
		got, err := parsePicture(test.pic)
		if err != nil {
			t.Errorf("PIC %s: %v", test.pic, err)
		} else if got != test.want {
			t.Errorf("PIC %s: got %+v, want %+v", test.pic, got, test.want)
		}
	}
}

func TestParsePictureErrors(t *testing.T) {
	for _, pic := range []string{"9S9", "9(0)", "(3)", "X(3", "99P", "9RD", "V"} {
		if _, err := parsePicture(pic); err == nil {
			t.Errorf("PIC %s parsed without error", pic)
		}
	}
}

func TestStorageSize(t *testing.T) {
	tests := []struct {
		pic, usage string
		want       int
	}{
		{"S9(11)V99", "DISPLAY", 13},
		{"S9(11)V99", "COMP-3", 7},
		{"S9(7)V99", "COMP-3", 5},
		{"S9(4)", "COMP", 2},
		{"9(9)", "COMP", 4},
		{"S9(18)", "COMP", 8},
	}
	for _, test := range tests {
		p, err := parsePicture(test.pic)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := storageSize(p, test.usage); err != nil || got != test.want {
			t.Errorf("PIC %s %s: %d, %v, want %d", test.pic, test.usage, got, err, test.want)
		}
	}
	if _, err := storageSize(picture{positions: 3, category: Alphanumeric}, "COMP-3"); err == nil {
		t.Error("COMP-3 accepted an alphanumeric picture")
	}
}
