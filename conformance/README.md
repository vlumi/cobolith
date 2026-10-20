# Conformance: NIST's COBOL-85 test suite

Cobolith reads 388 of the 459 programs of CCVS85, the COBOL Compiler Validation System for COBOL-85 of the National Institute of Standards and Technology. A program is read when it is lexed, its COPY statements expanded from the suite's library, parsed, and its records laid out in IBM COBOL. Whether it would compile, or run as the suite expects, is not tested.

The suite was created by NIST employees and is in the public domain; cobolith's tests use it with that acknowledgment, and do not redistribute it. `make conformance` fetches it from the Internet Archive's copy of NIST's address, checks it against the hash GNU Guix pins it by, and writes this report. Its programs are prepared as the suite's own program, EXEC85, prepares them with no control cards: the optional code a letter in column 7 flags is kept for the options it selects by default, A, E, H, L, T and Y, and made a comment for the rest. Its X-cards, the implementor's names it would substitute for XXXXXnnn, are left as written: most stand where any name may.

| Module | Programs | Read |
| --- | --: | --: |
| CM, Communication | 9 | 0 |
| DB, Debugging | 15 | 0 |
| EX, EXEC85, the suite's own program | 1 | 1 |
| IC, Inter-program communication | 47 | 47 |
| IF, Intrinsic functions | 45 | 45 |
| IX, Indexed I-O | 42 | 41 |
| NC, Nucleus | 95 | 70 |
| OB, Obsolete elements | 9 | 7 |
| RL, Relative I-O | 35 | 34 |
| RW, Report writer | 6 | 0 |
| SG, Segmentation | 13 | 13 |
| SM, Source text manipulation | 17 | 12 |
| SQ, Sequential I-O | 85 | 83 |
| ST, Sort-merge | 40 | 35 |

## Why the rest are not read

Each program stops at its first error. The reasons, most common first, with the programs each stops:

| Reason | Programs |
| --- | --- |
| `SYNCHRONIZED is not supported yet` | NC105A, NC106A, NC107A, NC108M, NC118A, NC119A, NC175A, NC176A, NC177A, NC201A, NC211A, NC250A, OBNC2M |
| `USE FOR DEBUGGING is not supported yet` | DB101A, DB102A, DB103M, DB104A, DB105A, DB201A, DB202A, DB203A, DB204A, DB301M, DB302M, DB303M, DB305M |
| `expected a number, found CD` | CM103M, CM104M, CM201M, CM303M, CM401M, DB205A, DB304M |
| `expected a number, found RD` | RW101A, RW102A, RW103A, RW104A |
| `SYNC is not supported yet` | NC138A, NC203A, NC251A |
| `expected a literal, found XXXXX081` | NC103A, NC174A, NC254A |
| `) is not supported yet` | NC205A, NC401M |
| `RENAMES is not supported yet` | NC209A, NC252A |
| `REPORT is not supported yet` | RW301M, RW302M |
| `expected a literal, found XXXXX063` | ST137A, ST147A |
| `COPY K2PRA REPLACING without BY` | SM202A |
| `COPY KP002 REPLACING without BY` | SM206A |
| `REPLACE is not supported yet` | SM401M |
| `TABLE5-NUM is not supported yet` | NC121M |
| `XXXXX086 is not supported yet` | SQ401M |
| `continuation line continues nothing` | NC215A |
| `expected RUN, found QUOTE` | OBNC1M |
| `expected a literal, found 12345678,91` | SM103A |
| `expected a literal, found XXXXX031` | CM101M |
| `expected a literal, found XXXXX033` | CM102M |
| `expected a literal, found XXXXX038` | CM105M |
| `expected a literal, found XXXXX042` | CM202M |
| `expected a literal, found XXXXX064` | ST144A |
| `expected a number, found DEPENDING` | SQ221A |
| `expected a number, found REPLACE` | SM208A |
| `expected an operand, found <` | RL211A |
| `no character positions` | IX110A |
| `unexpected IN` | ST127A |
| `unexpected IS` | ST140A                                                    TES00010 |
| `unexpected NUMERIC` | NC225A |
