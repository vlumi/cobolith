# Cobolith

A Go library for reading COBOL: source and copybooks, the records they lay out, and the jobs that run them. It keeps where everything came from, file and line, copybook or program, and lays records out byte for byte, as the compiler would.

It is pre-production, below 1.0 and not yet tagged: what it reads, it reads carefully, and what it does not read is an error, never a guess; but its API moves while the programs it is run against teach it more.

```go
p, err := cobol.ReadProgram("ACCTUPD.cbl", []string{"copybooks"})
if err != nil {
	log.Fatal(err) // file:line: what was not read
}
records, err := layout.Program(p, cobol.IBM)
// each field of records.WorkingStorage: name, start, length, picture, usage
```

## Packages

| Package | Reads |
| --- | --- |
| [cobol/](cobol/) | Source in fixed format, or free after `>>SOURCE FORMAT IS FREE`; `COPY` with `REPLACING`, by words, qualified names, literals or pseudo-text; the divisions, file control and file descriptions, data description entries, and the statements of the procedure division, every token with the file and line it came from. A file of several programs, one after the other, with `ParseAll`, and programs nested in one. |
| [layout/](layout/) | Record layouts from data description entries, in a dialect: each field's position, length, picture, usage and category, `OCCURS` with `DEPENDING ON`, `REDEFINES`, `SIGN`, and condition names. |
| [record/](record/) | The fields of a record by name, through its layout: text, display numbers with their overpunched signs, and packed decimal. |
| [jcl/](jcl/) | Jobs, their steps and DD statements: datasets with generations, temporary datasets, concatenations, `SYSOUT`, `DUMMY`, in-stream data, `COND` tests and dispositions. |

Each package has an example that runs as a test.

## Dialects

COBOL leaves some of a record's layout to the compiler, and a `cobol.Dialect` says how one lays it out:

| | IBM | Micro Focus |
| --- | --- | --- |
| Binary items: `COMP`, `COMP-4`, `COMP-5`, `BINARY` | a halfword, fullword or doubleword: 2, 4 or 8 bytes | as many bytes as the digits need, 1 to 8 |
| `USAGE INDEX` | 4 bytes | an error: its size depends on more than the dialect |

Packed decimal, display numbers, `COMP-1` and `COMP-2` are laid out alike in both.

## What it has been run against

- **Two COBOL systems written for it**, in [Copal](../README.md), the project it was written in: a bank in IBM COBOL, fixed format and run by JCL, and a card processor in Micro Focus COBOL, free format and run by shell scripts. Every program of both is read and laid out, and both are compiled and run with GnuCOBOL, so their layouts are checked against what a compiler made of them.
- **NIST's COBOL-85 test suite**, CCVS85: 383 of its 459 programs are read and laid out. [conformance/](conformance/README.md) says how many of each module, and why each of the rest stops; in Copal, `make conformance` fetches the suite and writes the report again.
- **Its own tests**: tables of statement forms and of data description entries, the latter laid out in both dialects, each with what it reads as or the error it stops with. They cover 86% of the statements of `cobol`, 90% of `layout`, and 93% of `record` and of `jcl`.

## What it does not read yet

Each is an error that says so, with the file and line:

- `SYNCHRONIZED`, whose slack bytes depend on the compiler and, in a table, on each occurrence; and `USAGE POINTER`, whose size depends on the platform.
- `RENAMES`, level 66, and `PIC N`, national characters.
- In `record`, the values of binary and floating-point fields, which `layout` lays out.
- `REPLACE`, and `DECIMAL-POINT IS COMMA`.
- A paragraph name repeated in another section: a reference by name alone would find the wrong one.
- The report writer, the communication module, and debugging declaratives, `USE FOR DEBUGGING`.
- Dialects other than IBM and Micro Focus COBOL, and the additions of the 2002 and 2014 standards beyond those the two share; `EXEC SQL` and `EXEC CICS`.

## Acknowledgment

The conformance report uses CCVS85, the COBOL-85 test suite of the National Institute of Standards and Technology, an agency of the U.S. Department of Commerce. It was created by NIST employees and is in the public domain. It is fetched, not redistributed here, and its programs are prepared as its own EXEC85 prepares them, with no other change.

## License

MIT, as Copal's: see [LICENSE](../LICENSE). Cobolith is a module of its own within Copal for now, and imports nothing of it; it is to move to a repository of its own once its API settles. The plan is in [Copal's docs](../docs/plans/library.md).
