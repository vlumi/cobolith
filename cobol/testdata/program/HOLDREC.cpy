      * A HOLD RECORD
       01  HOLD-RECORD.
           05  HR-TYPE                 PIC X.
               88  HR-OPEN             VALUE 'O'.
               88  HR-SETTLED          VALUES ARE 'S' 'X'.
           05  HR-BODY                 PIC X(19).
           05  HR-DETAIL REDEFINES HR-BODY.
               10  HR-ACCT-NO          PIC 9(10).
               10  HR-AMOUNT           PIC S9(7)V99 COMP-3.
               10  FILLER              PIC X(5).
