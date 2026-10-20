# Conformance: NIST's COBOL-85 test suite

Cobolith reads 338 of the 459 programs of CCVS85, the COBOL Compiler Validation System for COBOL-85 of the National Institute of Standards and Technology. A program is read when it is lexed, its COPY statements expanded from the suite's library, parsed, and its records laid out in IBM COBOL. Whether it would compile, or run as the suite expects, is not tested.

The suite was created by NIST employees and is in the public domain; cobolith's tests use it with that acknowledgment, and do not redistribute it. `make conformance` fetches it from the Internet Archive's copy of NIST's address, checks it against the hash GNU Guix pins it by, and writes this report. Its programs are prepared as the suite's own program, EXEC85, prepares them with no control cards: the optional code a letter in column 7 flags is kept for the options it selects by default, A, E, H, L, T and Y, and made a comment for the rest. Its X-cards, the implementor's names it would substitute for XXXXXnnn, are left as written: most stand where any name may.

| Module | Programs | Read |
| --- | --: | --: |
| CM, Communication | 9 | 0 |
| DB, Debugging | 15 | 0 |
| EX, EXEC85, the suite's own program | 1 | 1 |
| IC, Inter-program communication | 47 | 30 |
| IF, Intrinsic functions | 45 | 45 |
| IX, Indexed I-O | 42 | 38 |
| NC, Nucleus | 95 | 58 |
| OB, Obsolete elements | 9 | 6 |
| RL, Relative I-O | 35 | 34 |
| RW, Report writer | 6 | 0 |
| SG, Segmentation | 13 | 0 |
| SM, Source text manipulation | 17 | 12 |
| SQ, Sequential I-O | 85 | 79 |
| ST, Sort-merge | 40 | 35 |

## Why the rest are not read

Each program stops at its first error. The reasons, most common first, with the programs each stops:

| Reason | Programs |
| --- | --- |
| `SYNCHRONIZED is not supported yet` | NC105A, NC106A, NC107A, NC108M, NC118A, NC119A, NC175A, NC176A, NC177A, NC201A, NC211A, NC250A, OBNC2M |
| `USE FOR DEBUGGING is not supported yet` | DB101A, DB102A, DB103M, DB104A, DB105A, DB201A, DB202A, DB203A, DB204A, DB301M, DB302M, DB303M, DB305M |
| `USAGE INDEX is not supported yet` | IC106A, IC107A, IC207A, IC208A, NC131A, NC133A, NC135A, NC206A, NC236A |
| `IDENTIFICATION after END PROGRAM` | IC222A, IC223A, IC224A, IC225A, IC226A, IC227A, IC237A |
| `expected a number, found CD` | CM103M, CM104M, CM201M, CM303M, CM401M, DB205A, DB304M |
| `CANCEL is not supported yet` | IC203A, IC205A, IC210A, IC215A |
| `expected a number, found RD` | RW101A, RW102A, RW103A, RW104A |
| `ALTER is not supported yet` | NC302M, NC303M, OBNC1M |
| `SIGN on a group is not supported yet` | NC116A, NC117A, NC120A |
| `SORT-PARA is not supported yet` | SG104A, SG105A, SG106A |
| `SYNC is not supported yet` | NC138A, NC203A, NC251A |
| `expected a literal, found XXXXX081` | NC103A, NC174A, NC254A |
| `IS is not supported yet` | IC228A, IC401M |
| `NUMBER1 is not supported yet` | SG303M, SG401M |
| `RENAMES is not supported yet` | NC209A, NC252A |
| `REPORT is not supported yet` | RW301M, RW302M |
| `SECT-SG-02-001 is not supported yet` | SG102A, SG202A |
| `SECT-SG-03-001 is not supported yet` | SG103A, SG203A |
| `continuation line continues nothing` | NC205A, NC215A |
| `expected END, found END-OF-PAGE` | SQ201M, SQ208M |
| `expected a literal, found XXXXX063` | ST137A, ST147A |
| `) is not supported yet` | NC401M |
| `BEANO is not supported yet` | SG302M |
| `COPY K2PRA REPLACING without BY` | SM202A |
| `COPY KP002 REPLACING without BY` | SM206A |
| `HOUSEKEEPING is not supported yet` | SG201A |
| `IN is not supported yet` | IX215A |
| `REPLACE is not supported yet` | SM401M |
| `SECT-IC219-0001 is not supported yet` | OBIC2A |
| `SECT-SG-01-001 is not supported yet` | SG101A |
| `SRT-1 is not supported yet` | SG204A |
| `TABLE5-NUM is not supported yet` | NC121M |
| `XXXXX086 is not supported yet` | SQ401M |
| `a second procedure named INPUT-PROCESS is not supported yet` | IX218A |
| `a second procedure named OPEN-PASS-GF-01 is not supported yet` | SQ114A |
| `a second procedure named PAR-1A is not supported yet` | NC208A |
| `a second procedure named PFM-INIT-F2-5 is not supported yet` | NC102A |
| `a second procedure named READ-INIT-F1-04 is not supported yet` | IX207A |
| `expected END, found EOP` | SQ209M |
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
