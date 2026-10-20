# Conformance: NIST's COBOL-85 test suite

Cobolith reads 136 of the 459 programs of CCVS85, the COBOL Compiler Validation System for COBOL-85 of the National Institute of Standards and Technology. A program is read when it is lexed, its COPY statements expanded from the suite's library, parsed, and its records laid out in IBM COBOL. Whether it would compile, or run as the suite expects, is not tested.

The suite was created by NIST employees and is in the public domain; cobolith's tests use it with that acknowledgment, and do not redistribute it. `make conformance` fetches it from the Internet Archive's copy of NIST's address, checks it against the hash GNU Guix pins it by, and writes this report. Its programs are prepared as the suite's own program, EXEC85, prepares them with no control cards: the optional code a letter in column 7 flags is kept for the options it selects by default, A, E, H, L, T and Y, and made a comment for the rest. Its X-cards, the implementor's names it would substitute for XXXXXnnn, are left as written: most stand where any name may.

| Module | Programs | Read |
| --- | --: | --: |
| CM, Communication | 9 | 0 |
| DB, Debugging | 15 | 0 |
| EX, EXEC85, the suite's own program | 1 | 0 |
| IC, Inter-program communication | 47 | 25 |
| IF, Intrinsic functions | 45 | 33 |
| IX, Indexed I-O | 42 | 2 |
| NC, Nucleus | 95 | 39 |
| OB, Obsolete elements | 9 | 2 |
| RL, Relative I-O | 35 | 1 |
| RW, Report writer | 6 | 0 |
| SG, Segmentation | 13 | 0 |
| SM, Source text manipulation | 17 | 4 |
| SQ, Sequential I-O | 85 | 28 |
| ST, Sort-merge | 40 | 2 |

## Why the rest are not read

Each program stops at its first error. The reasons, most common first, with the programs each stops:

| Reason | Programs |
| --- | --- |
| `LABEL is not supported yet` | CM101M, CM102M, CM103M, CM104M, CM105M, CM202M, DB101A, DB102A, DB103M, DB104A, DB105A, DB201A, DB202A, DB203A, DB204A, DB205A, IX302M, NC104A, NC105A, OBSQ3A, OBSQ4A, OBSQ5A, RL102A, RL103A, RL104A, RL105A, RL106A, RL107A, RL108A, RL109A, RL110A, RL112A, RL113A, RL114A, RL115A, RL116A, RL117A, RL118A, RL119A, RL201A, RL202A, RL203A, RL204A, RL206A, RL208A, RL209A, RL210A, RL211A, RL212A, RL213A, RL302M, RW101A, RW102A, RW103A, RW104A, RW301M, RW302M, SG101A, SG102A, SG103A, SG104A, SG105A, SG106A, SG201A, SG202A, SG203A, SM101A, SM102A, SM103A, SM104A, SM201A, SM203A, SM204A, SQ302M, ST104A, ST105A, ST106A, ST107A, ST109A, ST110A, ST111A, ST112M, ST114M, ST115A, ST116A, ST117A, ST122A, ST123A, ST124A, ST125A, ST126A, ST133A, ST135A, ST136A, ST146A                                                    TES00010 |
| `BLOCK is not supported yet` | EXEC85, IC112A, IC115A, IX101A, IX102A, IX104A, IX105A, IX107A, IX108A, IX109A, IX110A, IX111A, IX112A, IX113A, IX114A, IX115A, IX116A, IX117A, IX118A, IX119A, IX120A, IX121A, IX201A, IX202A, IX203A, IX204A, IX216A, IX217A, IX218A, SQ104A, SQ107A, SQ108A, SQ110M, SQ114A, SQ117A, SQ122A, SQ127A, SQ136A, SQ137A, SQ138A, SQ148A, SQ153A, SQ202A, SQ204A, SQ206A, SQ213A, ST103A, ST121A, ST131A, ST132A |
| `DATA is not supported yet` | OBIC2A, OBSQ1A, SM105A, SM205A, SQ128A, ST101A, ST102A, ST108A, ST118A, ST119A, ST120A, ST127A, ST134A |
| `RESERVE is not supported yet` | IX205A, IX206A, IX401M, RL401M, SG204A, SQ203A, SQ226A, SQ401M, ST137A, ST139A, ST140A                                                    TES00010, ST144A, ST147A |
| `STATUS is not supported yet` | RL101A, RL111A, RL207A, SQ105A, SQ106A, SQ133A, SQ134A, SQ144A, SQ205A, SQ212A, SQ227A, SQ228A |
| `SYNCHRONIZED is not supported yet` | NC106A, NC107A, NC108M, NC118A, NC119A, NC175A, NC176A, NC177A, NC201A, NC211A, NC250A, OBNC2M |
| `expected an operand, found )` | IF119A, IF120A, IF121A, IF122A, IF123A, IF128A, IF129A, IF132A, IF137A, IF138A, IF141A, IF402M |
| `symbol P is not supported yet` | NC101A, NC111A, NC117A, NC120A, NC124A, NC134A, NC170A, NC171A, NC172A, NC173A |
| `ALTERNATE is not supported yet` | IX207A, IX208A, IX209A, IX210A, IX211A, IX212A, IX213A, IX214A, IX215A |
| `USAGE INDEX is not supported yet` | IC106A, IC107A, IC207A, IC208A, NC131A, NC133A, NC135A, NC206A, NC236A |
| `unexpected END` | NC237A, SQ103A, SQ109M, SQ112A, SQ115A, SQ121A, SQ220A, SQ223A, SQ224A |
| `unexpected NEXT` | IC201A, NC114M, NC231A, NC232A, NC233A, NC234A, NC235A, RL205A |
| `IDENTIFICATION after END PROGRAM` | IC222A, IC223A, IC224A, IC225A, IC226A, IC227A, IC237A |
| `CANCEL is not supported yet` | IC203A, IC205A, IC210A, IC215A |
| `LINAGE is not supported yet` | SQ201M, SQ208M, SQ209M, SQ210M |
| `USE FOR DEBUGGING is not supported yet` | DB301M, DB302M, DB303M, DB305M |
| `expected a number, found CD` | CM201M, CM303M, CM401M, DB304M |
| `ALTER is not supported yet` | NC302M, NC303M, OBNC1M |
| `IS is not supported yet` | IC228A, IC401M, SQ218A |
| `SEQUENTIAL is not supported yet` | SQ102A, SQ113A, SQ123A |
| `SYNC is not supported yet` | NC138A, NC203A, NC251A |
| `expected a literal, found XXXXX081` | NC103A, NC174A, NC254A |
| `unexpected CORRESPONDING` | NC202A, NC207A, NC253A |
| `GLOBAL is not supported yet` | IC233A, IC234A |
| `NUMBER1 is not supported yet` | SG303M, SG401M |
| `PADDING is not supported yet` | SQ216A, SQ217A |
| `RENAMES is not supported yet` | NC209A, NC252A |
| `continuation line continues nothing` | NC205A, NC215A |
| `unexpected WITH` | SQ211A, SQ215A |
| `) is not supported yet` | NC401M |
| `BEANO is not supported yet` | SG302M |
| `CODE-SET is not supported yet` | SQ111A |
| `COPY K2PRA REPLACING without BY` | SM202A |
| `COPY KP002 REPLACING without BY` | SM206A |
| `IN is not supported yet` | SQ124A |
| `INDEXED is not supported yet` | IX103A |
| `REPLACE is not supported yet` | SM401M |
| `SIGN on a group is not supported yet` | NC116A |
| `TABLE5-NUM is not supported yet` | NC121M |
| `XXXXX070 is not supported yet` | SQ219A |
| `a second procedure named PAR-1A is not supported yet` | NC208A |
| `a second procedure named PFM-INIT-F2-5 is not supported yet` | NC102A |
| `expected a number, found` | SQ222A |
| `expected a number, found DEPENDING` | SQ221A |
| `expected a number, found REPLACE` | SM208A |
| `unexpected NOT` | SQ126A |
| `unexpected NUMERIC` | NC225A |
