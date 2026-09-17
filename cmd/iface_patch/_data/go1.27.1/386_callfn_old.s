	/* call function */			\
	MOVL	f+4(FP), DX;			\
	MOVL	(DX), AX; 			\
	PCDATA  $PCDATA_StackMapIndex, $0;	\
	CALL	AX;				\
