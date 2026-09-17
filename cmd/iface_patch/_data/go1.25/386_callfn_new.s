	/* call function */			\
	MOVL	f+4(FP), DX;			\
	MOVL	(DX), AX; 			\
	BTL	$0, AX;				\
	JCC	4(PC);				\
	ANDL	$-2, AX;			\
	MOVL	AX, DX;				\
	MOVL	(DX), AX;			\
	PCDATA  $PCDATA_StackMapIndex, $0;	\
	CALL	AX;				\
