package jcl_test

import (
	"fmt"
	"strings"

	"github.com/vlumi/cobolith/jcl"
)

// Parse a job and list its steps, their COND tests and the datasets their
// DD statements bind.
func Example() {
	job, err := jcl.Parse(strings.NewReader(`//NIGHTLY  JOB
//SORT     EXEC PGM=SORTER
//INPUT    DD DSN=DATA.INPUT(0),DISP=SHR
//OUTPUT   DD DSN=&&SORTED,DISP=NEW,LRECL=80
//REPORT   EXEC PGM=REPORTER,COND=(4,LT)
//INPUT    DD DSN=&&SORTED,DISP=OLD
//REPORT   DD SYSOUT=*
`))
	if err != nil {
		panic(err)
	}
	for _, step := range job.Steps {
		fmt.Println(step.Name, step.Program, step.Cond != nil)
		for _, dd := range step.DDs {
			fmt.Printf("  %-6s %s temporary=%v sysout=%v\n", dd.Name, dd.DSN, dd.Temporary, dd.Sysout)
		}
	}
	// Output:
	// SORT SORTER false
	//   INPUT  DATA.INPUT temporary=false sysout=false
	//   OUTPUT SORTED temporary=true sysout=false
	// REPORT REPORTER true
	//   INPUT  SORTED temporary=true sysout=false
	//   REPORT  temporary=false sysout=true
}
