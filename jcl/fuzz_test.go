package jcl

import (
	"bytes"
	"os"
	"testing"
)

// FuzzParse parses whatever it is given as a job: a malformed job is an
// error, never a panic.
func FuzzParse(f *testing.F) {
	if job, err := os.ReadFile("testdata/TWOSTEP.jcl"); err == nil {
		f.Add(job)
	}
	f.Add([]byte("//JOB1 JOB\n//STEP1 EXEC PGM=X\n//DD1 DD DSN=A.B(+1),DISP=(NEW,CATLG)\n"))
	f.Fuzz(func(t *testing.T, source []byte) {
		_, _ = Parse(bytes.NewReader(source))
	})
}
