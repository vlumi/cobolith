# Cobolith

A Go library for reading COBOL: source and copybooks, the records they lay out, and the jobs that run them. Pre-production, below 1.0: it reads what the programs it was written against use, and its API moves with them.

- [cobol/](cobol/): lexing fixed-format and free-format source, expanding `COPY` with `REPLACING`, and parsing the divisions, data entries and statements of the subset read so far, every token with the file and line it came from; and the dialects, IBM and Micro Focus, and what they lay out differently.
- [layout/](layout/): record layouts from data description entries, in a dialect: position, length, picture, usage, `OCCURS`, `REDEFINES` and condition names.
- [record/](record/): reading and writing the fields of a record through its layout, including packed decimal and the overpunched signs of display numbers.
- [jcl/](jcl/): jobs, steps and DD statements from job files, with generations, temporary datasets and concatenations.

It is a module of its own within Copal for now, and imports nothing of it; it is to move to a repository of its own once its API settles. The plan is in [Copal's docs](../docs/plans/library.md).
