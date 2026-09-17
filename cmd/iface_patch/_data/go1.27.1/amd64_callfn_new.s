	/* call function */			\
	MOVQ	f+8(FP), DX;			\
	MOVQ	(DX), R12;			\
	BTQ	$0, R12;			\
	JCC	4(PC);				\
	ANDQ	$-2, R12;			\
	MOVQ	R12, DX;			\
	MOVQ	(DX), R12;			\
	PCDATA  $PCDATA_StackMapIndex, $0;	\
	CALL	R12;				\
