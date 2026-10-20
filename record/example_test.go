package record_test

import (
	"fmt"

	"github.com/vlumi/cobolith/cobol"
	"github.com/vlumi/cobolith/record"
)

// Read fields of a record by name, through the layout of its copybook: a
// signed number overpunched on its last digit, in units of its scale.
func Example() {
	l, err := record.FromCopybook("testdata/ACCOUNT.cpy", nil, "ACCT-MASTER-RECORD", cobol.IBM)
	if err != nil {
		panic(err)
	}
	r := []byte("000001000600000105EOHIPPUS ELLA       CO2020090920261015000000001500}000030000                      ")
	name, _ := l.Text(r, "AM-SHORT-NAME")
	balance, _ := l.Number(r, "AM-BALANCE")
	fmt.Printf("%s: %.2f\n", name, float64(balance)/100)
	// Output:
	// EOHIPPUS ELLA: -150.00
}
