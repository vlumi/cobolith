      * AN ACCOUNT, 100 BYTES: DISPLAY NUMBERS, ONE OF THEM SIGNED.
       01  ACCT-MASTER-RECORD.
           05  AM-ACCT-NO              PIC 9(10).
           05  AM-CUST-NO              PIC 9(8).
           05  AM-SHORT-NAME           PIC X(20).
           05  AM-ACCT-TYPE            PIC X.
               88  AM-CURRENT          VALUE 'C'.
               88  AM-SAVINGS          VALUE 'S'.
           05  AM-STATUS               PIC X.
               88  AM-OPEN             VALUE 'O'.
               88  AM-CLOSED           VALUE 'X'.
           05  AM-OPEN-DATE            PIC 9(8).
           05  AM-LAST-ACTIVITY-DATE   PIC 9(8).
           05  AM-BALANCE              PIC S9(11)V99.
           05  AM-OVERDRAFT-LIMIT      PIC 9(7)V99.
           05  FILLER                  PIC X(22).
