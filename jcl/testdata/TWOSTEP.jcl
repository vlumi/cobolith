//TWOSTEP  JOB
//*----------------------------------------------------------------*
//* A JOB OF TWO STEPS, THE SECOND BYPASSED AFTER A RETURN CODE    *
//* ABOVE 4                                                        *
//*----------------------------------------------------------------*
//REPORT   EXEC PGM=REPORTER
//INPUT    DD DSN=DATA.INPUT(0),DISP=SHR
//REPORT   DD SYSOUT=*
//SUMMARY  EXEC PGM=SUMMER,COND=(4,LT)
//INPUT    DD DSN=DATA.INPUT(0),DISP=SHR
//OUTPUT   DD DSN=DATA.SUMMARY(+1),DISP=NEW,
//            DCB=(RECFM=FB,LRECL=60)
