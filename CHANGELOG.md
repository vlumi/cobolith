# Changelog

What changes in each release, newest first. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the versions [Semantic Versioning](https://semver.org/spec/v2.0.0.html): below 1.0, a minor version may change the API.

## [Unreleased]

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

[Unreleased]: https://github.com/vlumi/cobolith/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/vlumi/cobolith/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/vlumi/cobolith/releases/tag/v0.1.0
