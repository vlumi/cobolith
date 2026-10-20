// Command conformance reads the programs of NIST's COBOL-85 test suite,
// CCVS85, with cobolith: each program lexed with its COPY statements
// expanded from the suite's library, parsed, and its records laid out in
// IBM COBOL; and writes a report of how many it reads, by module, and why
// the rest fail.
//
//	go run ./conformance -suite newcob.val -out conformance/README.md
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vlumi/cobolith/cobol"
	"github.com/vlumi/cobolith/layout"
)

func main() {
	suite := flag.String("suite", "", "the suite's population file, newcob.val")
	out := flag.String("out", "", "the report to write")
	flag.Parse()
	if err := run(*suite, *out); err != nil {
		fmt.Fprintln(os.Stderr, "conformance:", err)
		os.Exit(1)
	}
}

func run(suite, out string) error {
	f, err := os.Open(suite)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	members, err := readPopulation(f)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "ccvs85")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	programs, err := extract(members, dir)
	if err != nil {
		return err
	}
	results := make([]result, len(programs))
	for i, name := range programs {
		results[i] = result{program: name}
		if err := read(dir, name); err != nil {
			results[i].failure = strings.ReplaceAll(err.Error(), dir+string(filepath.Separator), "")
		}
	}
	return os.WriteFile(out, []byte(report(results)), 0o644)
}

// extract writes the programs and library texts to files of their own
// under a directory, a program as NAME.CBL and a library text as NAME.CPY,
// and lists the programs.
func extract(members []member, dir string) ([]string, error) {
	var programs []string
	for _, m := range members {
		var path string
		switch m.kind {
		case "COBOL":
			path = filepath.Join(dir, m.name+".CBL")
			programs = append(programs, m.name)
		case "CLBRY":
			path = filepath.Join(dir, m.name+".CPY")
		default:
			continue
		}
		if err := os.WriteFile(path, []byte(strings.Join(m.lines, "\n")+"\n"), 0o644); err != nil {
			return nil, err
		}
	}
	return programs, nil
}

// read reads a program as cobolith does: lexed and parsed, its copybooks
// from the suite's library, and its records laid out in IBM COBOL.
func read(dir, name string) error {
	p, err := cobol.ReadProgram(filepath.Join(dir, name+".CBL"), []string{dir})
	if err != nil {
		return err
	}
	_, err = layout.Program(p, cobol.IBM)
	return err
}
