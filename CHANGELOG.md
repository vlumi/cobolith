# Changelog

What changes in each release, newest first. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the versions [Semantic Versioning](https://semver.org/spec/v2.0.0.html): below 1.0, a minor version may change the API.

## [Unreleased]

### Fixed

- `record.FromCopybook` reads the copybook in the dialect's encoding, so the fields of a Shift_JIS copybook are found by their names in UTF-8, as a program's are.

## [0.8.1] - 2026-10-22

### Added

- `cobol.ToUTF8` gives text in an encoding a source may be read in, Shift_JIS or none, in UTF-8, as the lexer gives its tokens: for what stands beside the source, such as the comments of the batch files that run it.

### Fixed

- The lines of a report group with more than one are numbered in order, `PAGE-HEADING-LINE-2`, rather than named after their `LINE` clauses, which two lines of one group may share.

## [0.8.0] - 2026-10-22

### Added

- The Report Writer: `REPORT IS` on an FD, which then has no records of its own; the `REPORT SECTION` as `Program.ReportSection`, each `RD` a `ReportDescription` with its controls and page limits; its report groups as data entries with `Type`, `TypeControl`, `Line`, `Column`, `Source` and `Sum`; and `INITIATE`, `GENERATE` and `TERMINATE`, which read the report or group they name. NIST's six Report Writer programs read with it: 439 of 459.
- `ReadDataEntriesIn`, a copybook of data description entries read in an encoding, as `ReadIn` reads source.
- `layout.ReportLines` lays out the lines a report prints as the records of its file, one for each `LINE` of each report group, its columns placed where `COLUMN` puts them and as wide as their pictures, an unnamed one named after its line and column, `PAGE-HEADING-COL-70`; `layout.Program` gives a file with a `REPORT IS` clause those records, and a column's `Source` and `Sum` ride on its `Field`.

### Changed

- A source read in Shift_JIS comes out in UTF-8: the lexer transcodes every token's text and comment, names, literals and comments alike, so that what a program is called and says reads wherever it goes. The data `record` reads and writes stays as the file has it. Cobolith gains its first dependency, `golang.org/x/text`, for the transcoding.

## [0.7.0] - 2026-10-20

### Added

- Fujitsu NetCOBOL, `cobol.Fujitsu` for `ENCODE(SJIS)` and `cobol.FujitsuUnicode` for `ENCODE(UTF8)`: source and data in Shift_JIS or UTF-8, national items two bytes a character in both, binary items of 2, 4 or 8 bytes, native binary and IEEE floating point little-endian, and `SYNCHRONIZED` as in IBM COBOL.
- `SYNCHRONIZED` on a group synchronizes each of its items.

### Fixed

- The slack bytes that align the first item of a group go before the group, as IBM's example of them shows, so the group starts where its first item does; they were counted in the group.

## [0.6.0] - 2026-10-20

### Added

- Hitachi COBOL2002, `cobol.Hitachi`, the first Japanese dialect: source and data in Shift_JIS, read two bytes at a time by `LexIn`, `ReadIn`, `ReadProgramIn` and `ReadProgramsIn`, which take the dialect or its encoding; Japanese names stay whole and are not upper-cased in their second bytes. National literals, `N"..."`, `NC"..."` and `NX"..."`, a token kind of their own, `National`. National items, `PIC N` and `USAGE NATIONAL`, and national-edited ones, `PIC NBN`, laid out at the dialect's `NationalSize` bytes a character: two in Hitachi COBOL2002, and an error where it is not known. `cobol.Dialect` has `Encoding` and `NationalSize`.

### Fixed

- A literal's bytes above 0x7F were encoded as UTF-8 when the literal was read, so an `X'...'` literal was right but a quoted one of such bytes was not.

## [0.5.0] - 2026-10-20

### Added

- `REPLACE`, which replaces text in the source after it until `REPLACE OFF`, as `COPY REPLACING` does; and a `REPLACING` operand with a subscript.
- SPECIAL-NAMES' `DECIMAL-POINT IS COMMA` and `CURRENCY SIGN`, in `cobol.Program` as `DecimalPointComma` and `CurrencySign`: the program's pictures and numeric literals are read as written and kept in the standard form, the decimal point a period and the currency symbol `$`.
- `RENAMES`, level 66: a field over the bytes of the fields it renames, after the record's own fields, with `Renames` and `RenamesThru` on the entry.
- `USE FOR DEBUGGING` declaratives, `Debugging` on `cobol.Declarative`, which name no file and run no file's error procedure.
- Abbreviated combined relations with their operator, `A > 1 AND < 9`, with `IS` and `NOT`; an arithmetic expression in parentheses as a relation's side; `EVALUATE` of a condition, of a condition name, and of `FALSE`, whose objects are conditions taken as their opposites.
- `PERFORM` with a subscripted count, `TEST BEFORE` without `WITH`; `SORT` and `MERGE` with `DUPLICATES IN ORDER` and `COLLATING SEQUENCE`; `STOP` with a figurative constant; `RECORD VARYING DEPENDING` with no lengths; a picture continued across lines.
- Of NIST's COBOL-85 suite, 433 programs of 459 are read, from 399. What stops the rest is the communication module and the report writer, the suite's X-card placeholders where a literal must stand, a picture of `P` alone, and a literal continued across lines with a quote at the margin.

### Fixed

- A `VALUE` whose literal reads like a clause word, `VALUE "PICTURE"`, is a value.
- A paragraph name that `COPY REPLACING` replaces is still a header.

## [0.4.0] - 2026-10-20

### Added

- `SYNCHRONIZED`, in IBM COBOL: slack bytes before a synchronized binary, floating-point or index item, to a multiple of its alignment from the start of the record, and at the end of each occurrence of a table that holds one, as IBM documents them. `cobol.DataEntry` and `layout.Field` have `Synchronized`, and `cobol.Dialect` `Synchronizes`; in Micro Focus COBOL, whose compiler directives decide, it is an error. Of NIST's COBOL-85 suite, 399 programs of 459 are read, from 388.

## [0.3.0] - 2026-10-20

### Added

- A paragraph name may repeat in another section. `ProcedureRef` has the `Section` and `ThruSection` of the paragraphs it names, as written after `OF` or `IN`, or as the program resolves a name alone: to its own section's paragraph by that name, or the only one. `Span` finds the paragraph of that section. A name alone that more than one other section has is an error. Of NIST's COBOL-85 suite, 388 programs of 459 are read, from 383.

## [0.2.0] - 2026-10-20

### Added

- `record` reads and writes binary fields, `COMP`, `COMP-4`, `BINARY` and `COMP-5`, through `Number` and `SetNumber`, and floating-point fields, `COMP-1` and `COMP-2`, through the new `Float` and `SetFloat`: IBM hexadecimal floating point in IBM COBOL.
- `cobol.Dialect` says the byte order of native binary and IEEE floating point, `NativeOrder`, nil where it is the platform's, as in Micro Focus COBOL, and whether floating point is hexadecimal, `HexFloat`.

### Changed

- `record.New` takes the dialect the record is laid out in, which says how its binary and floating-point fields hold their values.

## [0.1.0] - 2026-10-20

The first release, a library of its own.

### Added

- `cobol`: source in fixed format, or free after `>>SOURCE FORMAT IS FREE`; `COPY` with `REPLACING`; the divisions, file control and file descriptions, data description entries, and the statements of the procedure division, every token with the file and line it came from. `ReadProgram` and `Parse` read a program and the programs nested in it, `ReadPrograms` and `ParseAll` a file of programs one after the other.
- `cobol.Dialect`: IBM and Micro Focus COBOL, and what they lay out differently, the sizes of binary items and of `USAGE INDEX`.
- `layout`: record layouts from data description entries, in a dialect: each field's position, length, picture, usage and category, `OCCURS` with `DEPENDING ON`, `REDEFINES`, `SIGN`, and condition names.
- `record`: the fields of a record by name, through its layout: text, display numbers with their overpunched signs, and packed decimal.
- `jcl`: jobs, their steps and DD statements, with generations, temporary datasets, concatenations, `COND` tests and dispositions.
- `conformance`: a command that reads NIST's COBOL-85 test suite, CCVS85, and reports what it reads, by module, and why the rest stops: 383 of its 459 programs.

[Unreleased]: https://github.com/vlumi/cobolith/compare/v0.8.1...HEAD
[0.8.1]: https://github.com/vlumi/cobolith/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/vlumi/cobolith/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/vlumi/cobolith/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/vlumi/cobolith/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/vlumi/cobolith/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/vlumi/cobolith/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/vlumi/cobolith/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/vlumi/cobolith/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/vlumi/cobolith/releases/tag/v0.1.0
