# Conformance: NIST's COBOL-85 test suite

Cobolith reads 439 of the 459 programs of CCVS85, the COBOL Compiler Validation System for COBOL-85 of the National Institute of Standards and Technology. A program is read when it is lexed, its COPY statements expanded from the suite's library, parsed, and its records laid out in IBM COBOL. Whether it would compile, or run as the suite expects, is not tested.

The suite was created by NIST employees and is in the public domain; cobolith's tests use it with that acknowledgment, and do not redistribute it. `make conformance` fetches it from the Internet Archive's copy of NIST's address, checks it against the hash GNU Guix pins it by, and writes this report. Its programs are prepared as the suite's own program, EXEC85, prepares them with no control cards: the optional code a letter in column 7 flags is kept for the options it selects by default, A, E, H, L, T and Y, and made a comment for the rest. Its X-cards, the implementor's names it would substitute for XXXXXnnn, are left as written: most stand where any name may.

| Module | Programs | Read |
| --- | --: | --: |
| CM, Communication | 9 | 0 |
| DB, Debugging | 15 | 13 |
| EX, EXEC85, the suite's own program | 1 | 1 |
| IC, Inter-program communication | 47 | 47 |
| IF, Intrinsic functions | 45 | 45 |
| IX, Indexed I-O | 42 | 41 |
| NC, Nucleus | 95 | 91 |
| OB, Obsolete elements | 9 | 9 |
| RL, Relative I-O | 35 | 35 |
| RW, Report writer | 6 | 6 |
| SG, Segmentation | 13 | 13 |
| SM, Source text manipulation | 17 | 17 |
| SQ, Sequential I-O | 85 | 84 |
| ST, Sort-merge | 40 | 37 |

## Why the rest are not read

Each program stops at its first error. The reasons, most common first, with the programs each stops:

| Reason | Programs |
| --- | --- |
| `expected a number, found CD` | CM103M, CM104M, CM201M, CM303M, CM401M, DB205A, DB304M |
| `expected a literal, found XXXXX081` | NC103A, NC174A, NC254A |
| `expected a literal, found XXXXX063` | ST137A, ST147A |
| `XXXXX086 is not supported yet` | SQ401M |
| `continuation line continues nothing` | NC215A |
| `expected a literal, found XXXXX031` | CM101M |
| `expected a literal, found XXXXX033` | CM102M |
| `expected a literal, found XXXXX038` | CM105M |
| `expected a literal, found XXXXX042` | CM202M |
| `expected a literal, found XXXXX064` | ST144A |
| `no character positions` | IX110A |
