package jcl

import (
	"reflect"
	"strings"
	"testing"
)

const daily = `//DAILY    JOB
//* VALIDATE THE DAY'S TRANSACTIONS
//VALID    EXEC PGM=TXVALID
//TRANIN   DD DSN=TRAN.DAILY,DISP=SHR
//TRANOUT  DD DSN=TRAN.VALID,DISP=NEW,         VALID DETAILS
//            DCB=(RECFM=FB,LRECL=80)
//REPORT   DD SYSOUT=*
//POST     EXEC PGM=TXPOST,COND=(4,LT)
//TRANIN   DD DSN=TRAN.VALID,DISP=OLD
`

func TestParse(t *testing.T) {
	job, err := Parse(strings.NewReader(daily))
	if err != nil {
		t.Fatal(err)
	}
	want := &Job{
		Name: "DAILY",
		Steps: []Step{
			{
				Name:    "VALID",
				Program: "TXVALID",
				Line:    3,
				DDs: []DD{
					{Name: "TRANIN", DSN: "TRAN.DAILY", Disp: Shr, Line: 4},
					{Name: "TRANOUT", DSN: "TRAN.VALID", Disp: New, LRECL: 80, Line: 5},
					{Name: "REPORT", Sysout: true, Line: 7},
				},
			},
			{
				Name:    "POST",
				Program: "TXPOST",
				Cond:    &Cond{Code: 4, Operator: "LT"},
				Line:    8,
				DDs: []DD{
					{Name: "TRANIN", DSN: "TRAN.VALID", Disp: Old, Line: 9},
				},
			},
		},
	}
	if !reflect.DeepEqual(job, want) {
		t.Errorf("got %+v\nwant %+v", job, want)
	}
}

func TestParseErrors(t *testing.T) {
	tests := map[string]string{
		"no JOB":          "//VALID EXEC PGM=TXVALID\n",
		"DD before EXEC":  "//J JOB\n//IN DD DSN=A,DISP=SHR\n",
		"no PGM":          "//J JOB\n//S EXEC COND=(4,LT)\n",
		"bad operator":    "//J JOB\n//S EXEC PGM=P,COND=(4,XX)\n",
		"new without DCB": "//J JOB\n//S EXEC PGM=P\n//OUT DD DSN=A,DISP=NEW\n",
		"unknown DISP":    "//J JOB\n//S EXEC PGM=P\n//IN DD DSN=A,DISP=MOD\n",
		"variable RECFM":  "//J JOB\n//S EXEC PGM=P\n//O DD DSN=A,DISP=NEW,DCB=(RECFM=VB,LRECL=80)\n",
		"no name":         "//J JOB\n// EXEC PGM=P\n",
		"not a statement": "//J JOB\nEXEC PGM=P\n",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(input)); err == nil {
				t.Error("parsed without error")
			}
		})
	}
}

func TestCondBypasses(t *testing.T) {
	tests := []struct {
		cond Cond
		rc   int
		want bool
	}{
		{Cond{4, "LT"}, 0, false},
		{Cond{4, "LT"}, 4, false},
		{Cond{4, "LT"}, 8, true},
		{Cond{0, "NE"}, 0, false},
		{Cond{0, "NE"}, 4, true},
		{Cond{8, "LE"}, 8, true},
		{Cond{8, "GT"}, 12, false},
	}
	for _, test := range tests {
		if got := test.cond.Bypasses(test.rc); got != test.want {
			t.Errorf("COND=(%d,%s) after RC %d: got %v, want %v",
				test.cond.Code, test.cond.Operator, test.rc, got, test.want)
		}
	}
}
