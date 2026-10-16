       IDENTIFICATION DIVISION.
       PROGRAM-ID. CALLED.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01  WS-COUNT                    PIC 9(4) VALUE 0.
       LINKAGE SECTION.
       01  LK-REQUEST.
           05  LK-CODE                 PIC X(2).
           05  LK-AMOUNT               PIC 9(7).
       01  LK-RESPONSE                 PIC X(2).
       PROCEDURE DIVISION USING LK-REQUEST
                                LK-RESPONSE.
       MAIN-PARA.
           MOVE LK-CODE TO LK-RESPONSE
           GOBACK.
