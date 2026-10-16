      * SAMPLE: READS HOLD RECORDS
       IDENTIFICATION DIVISION.
       PROGRAM-ID. SAMPLE.
       AUTHOR. A. N. AUTHOR.
       DATE-WRITTEN. 1987-03-12.
       ENVIRONMENT DIVISION.
       INPUT-OUTPUT SECTION.
       FILE-CONTROL.
           SELECT HOLD-FILE ASSIGN TO HOLDIN
               ORGANIZATION IS LINE SEQUENTIAL
               FILE STATUS IS WS-STATUS.
       DATA DIVISION.
       FILE SECTION.
       FD  HOLD-FILE.
           COPY HOLDREC.
       WORKING-STORAGE SECTION.
       01  WS-STATUS                   PIC XX.
       01  WS-LIST.
           05  WS-COUNT                PIC 9(2) VALUE 0.
           05  WS-ITEM                 PIC X(10)
                   OCCURS 0 TO 20 TIMES DEPENDING ON WS-COUNT
                   INDEXED BY WS-IX.
       01  WS-GRADE                    PIC 9.
           88  PASSING                 VALUE 5 THRU 9.
       PROCEDURE DIVISION.
       MAIN-PARA.
           PERFORM READ-PARA
           STOP RUN.
      * READ ONE RECORD.
       READ-PARA.
           READ HOLD-FILE.
