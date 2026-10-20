# Cobolith

A Go library for reading COBOL: source and copybooks, the records they lay out, and the jobs that run them. It keeps where everything came from, file and line, copybook or program, and lays records out byte for byte, as the compiler would.

It is pre-production, below 1.0: what it reads, it reads carefully, and what it does not read is an error, never a guess; but its API moves while the programs it is run against teach it more. [CHANGELOG.md](CHANGELOG.md) says what each release changes.

`go get github.com/vlumi/cobolith` and, for a program that reads an account file through a copybook:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/vlumi/cobolith/cobol"
    "github.com/vlumi/cobolith/layout"
    "github.com/vlumi/cobolith/record"
)

func main() {
    // The program, its copybooks expanded; an error says file:line: what was not read.
    p, err := cobol.ReadProgram("ACCTLIST.cbl", []string{"copybooks"})
    if err != nil {
        log.Fatal(err)
    }
    for _, para := range p.Paragraphs {
        cobol.Walk(para.Statements, func(s cobol.Statement) {
            fmt.Printf("%s %d: %s\n", para.Name, s.Pos.Line, s.Text)
        })
    }

    // Its records, laid out as the dialect's compiler lays them out.
    records, err := layout.Program(p, cobol.IBM)
    if err != nil {
        log.Fatal(err)
    }
    account := records.Files[0].Records[0]
    for _, f := range account.Children {
        fmt.Printf("%-22s %3d+%-3d %s\n", f.Name, f.Start, f.Length, f.Picture)
    }

    // A record of the file, read field by field through that layout.
    data, err := os.ReadFile("accounts.dat")
    if err != nil {
        log.Fatal(err)
    }
    l := record.New(account, cobol.IBM)
    name, _ := l.Text(data[:l.Length], "AM-SHORT-NAME")
    balance, _ := l.Number(data[:l.Length], "AM-BALANCE")
    fmt.Printf("%s: %.2f\n", name, float64(balance)/100)
}
```

```text
MAIN-PARA 17: OPEN INPUT ACCT-FILE
MAIN-PARA 18: PERFORM UNTIL WS-EOF = 'Y'
MAIN-PARA 19: READ ACCT-FILE
MAIN-PARA 20: MOVE 'Y' TO WS-EOF
MAIN-PARA 21: PERFORM ADD-BALANCE
...
AM-ACCT-NO               1+10  9(10)
AM-CUST-NO              11+8   9(8)
AM-SHORT-NAME           19+20  X(20)
...
AM-BALANCE              57+13  S9(11)V99
AM-OVERDRAFT-LIMIT      70+9   9(7)V99
FILLER                  79+22  X(22)
EOHIPPUS ELLA: -150.00
```

`jcl.ParseFile` reads the job that runs such a program: its steps, and the datasets their DD statements bind to the program's files.

## Packages

| Package | Reads |
| --- | --- |
| [cobol/](cobol/) | Source in fixed format, or free after `>>SOURCE FORMAT IS FREE`; `COPY` with `REPLACING`, by words, qualified names, literals or pseudo-text; the divisions, file control and file descriptions, data description entries, and the statements of the procedure division, every token with the file and line it came from; `REPLACE`; and SPECIAL-NAMES' `DECIMAL-POINT IS COMMA` and `CURRENCY SIGN`, whose pictures and literals are kept in the standard form. A file of several programs, one after the other, with `ParseAll`, and programs nested in one. |
| [layout/](layout/) | Record layouts from data description entries, in a dialect: each field's position, length, picture, usage and category, `OCCURS` with `DEPENDING ON`, `REDEFINES`, `RENAMES`, `SIGN`, `SYNCHRONIZED`, and condition names. |
| [record/](record/) | The fields of a record by name, through its layout: text, display numbers with their overpunched signs, packed decimal, binary, and floating point. |
| [jcl/](jcl/) | Jobs, their steps and DD statements: datasets with generations, temporary datasets, concatenations, `SYSOUT`, `DUMMY`, in-stream data, `COND` tests and dispositions. |

Each package has an example that runs as a test.

## Dialects

COBOL leaves some of a record's layout to the compiler, and a `cobol.Dialect` says how one lays it out, and in which encoding its source is read:

| | IBM | Micro Focus | Hitachi COBOL2002 | Fujitsu NetCOBOL |
| --- | --- | --- | --- | --- |
| Source and data | ASCII or EBCDIC, one byte a character | the same | Shift_JIS, read two bytes at a time | Shift_JIS, or UTF-8 in `FujitsuUnicode` |
| National items, `PIC N` | an error | an error | 2 bytes a character | 2 bytes a character: Shift_JIS, or UTF-16 |
| Binary items: `COMP`, `COMP-4`, `COMP-5`, `BINARY` | a halfword, fullword or doubleword: 2, 4 or 8 bytes | as many bytes as the digits need, 1 to 8 | 2, 4 or 8 bytes | 2, 4 or 8 bytes |
| `USAGE INDEX` | 4 bytes | an error: its size depends on more than the dialect | an error | an error: 4 or 8 bytes by the platform |
| Native binary, `COMP-5` | big-endian | the platform's byte order: an error in `record` | an error | little-endian |
| Floating point, `COMP-1` and `COMP-2` | IBM hexadecimal floating point | IEEE 754 in the platform's byte order: an error in `record` | an error | IEEE 754, little-endian |
| `SYNCHRONIZED` | slack bytes before a binary, floating-point or index item, to a multiple of 2, 4 or 8 from the start of the record, and at the end of each occurrence of a table that holds one | an error: compiler directives decide | an error | as IBM |

The Japanese dialects' source and data are in Shift_JIS, which the lexer reads two bytes at a time, so Japanese names and literals stay whole; a national item takes two bytes a character, as the manuals say and opensource COBOL, the Japanese GnuCOBOL fork, lays it out. NetCOBOL's `ENCODE(UTF8)`, the default in a Unicode locale, keeps alphanumeric data in UTF-8 and national data in UTF-16: `cobol.FujitsuUnicode`. Its `ENCODE(UTF8,UTF32)`, its variable-length source format with no right margin, and the Japanese dialects' index items are not read.

IBM COBOL's slack bytes are as IBM documents them, with IBM's own example among the tests; GnuCOBOL's `-std=ibm` lays out the same records alike, but aligns a table's items from the table's start rather than the record's. Packed decimal and display numbers are laid out alike in every dialect, and so are `COMP-1` and `COMP-2`, four and eight bytes.

## What it has been run against

- **Two COBOL systems written for it**, in [Copal](https://github.com/vlumi/copal), which reads COBOL batch systems with it: a bank in IBM COBOL, fixed format and run by JCL, and a card processor in Micro Focus COBOL, free format and run by shell scripts. Every program of both is read and laid out, and both are compiled and run with GnuCOBOL, so their layouts are checked against what a compiler made of them.
- **NIST's COBOL-85 test suite**, CCVS85: 433 of its 459 programs are read and laid out. [conformance/](conformance/README.md) says how many of each module, and why each of the rest stops; `make conformance` fetches the suite and writes the report again.
- **Its own tests**: tables of statement forms and of data description entries, the latter laid out in both dialects, each with what it reads as or the error it stops with. They cover 86% of the statements of `cobol`, 90% of `layout`, and 93% of `record` and of `jcl`.

## What it does not read yet

Each is an error that says so, with the file and line:

- `USAGE POINTER`, whose size depends on the platform.
- National items, `PIC N`, in IBM and Micro Focus COBOL, whose sizes are not verified yet; and a picture of `P` alone.
- The report writer and the communication module.
- Dialects other than IBM, Micro Focus, Hitachi and Fujitsu COBOL, and the additions of the 2002 and 2014 standards beyond those the two share; `EXEC SQL` and `EXEC CICS`.

## Acknowledgment

The conformance report uses CCVS85, the COBOL-85 test suite of the National Institute of Standards and Technology, an agency of the U.S. Department of Commerce. It was created by NIST employees and is in the public domain. It is fetched, not redistributed here, and its programs are prepared as its own EXEC85 prepares them, with no other change.

## License

MIT: see [LICENSE](LICENSE).
