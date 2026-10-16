       IDENTIFICATION DIVISION.
       PROGRAM-ID. STATEMENTS.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01  WS-A                        PIC 9.
       PROCEDURE DIVISION.
       MAIN-PARA.
           MOVE WS-A TO WS-B WS-C(WS-I)
           ADD 1 WS-A TO WS-B ROUNDED
           SUBTRACT WS-A FROM WS-B GIVING WS-C
           DIVIDE WS-A BY 365 GIVING WS-D ROUNDED
           COMPUTE WS-E = WS-A * (WS-B + 1) - FUNCTION MOD(WS-C, 7)
           SET DONE TO TRUE
           SET WS-X TO WS-I
           STRING 'A' WS-F(1:6) DELIMITED BY SIZE INTO WS-G
           IF WS-A = 1 OR 2 AND NOT DONE
               PERFORM OTHER-PARA
           ELSE
               IF WS-B IS NOT NUMERIC
                   GO TO OTHER-PARA
               END-IF
           END-IF
           EVALUATE WS-A ALSO TRUE
               WHEN 1 ALSO WS-B > 2
               WHEN 3 THRU 5 ALSO ANY
                   CONTINUE
               WHEN OTHER
                   DISPLAY 'NO' WS-A
           END-EVALUATE
           PERFORM VARYING WS-I FROM 1 BY 1 UNTIL WS-I > WS-N
               READ IN-FILE
                   AT END
                       SET DONE TO TRUE
                   NOT AT END
                       WRITE OUT-RECORD FROM IN-RECORD
               END-READ
           END-PERFORM
           PERFORM OTHER-PARA THRU OTHER-EXIT UNTIL DONE
           SEARCH WS-ENTRY
               AT END
                   MOVE 0 TO WS-A
               WHEN WS-CODE(WS-CX) = 'DP'
                   MOVE WS-TEXT(WS-CX) TO WS-G
           END-SEARCH
           OPEN INPUT IN-FILE OUTPUT OUT-FILE
           SORT WORK-FILE ON ASCENDING KEY WS-A
               USING IN-FILE GIVING OUT-FILE
           GO TO OTHER-PARA OTHER-EXIT DEPENDING ON WS-A.
           IF WS-A GREATER THAN OR EQUAL TO WS-B
               STOP RUN.
           STOP RUN.
       OTHER-PARA.
           EXIT PROGRAM.
       OTHER-EXIT.
           EXIT.
