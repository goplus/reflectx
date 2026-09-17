	/* call function */			\
	MOVD	f+8(FP), R26;			\
	MOVD	(R26), R20;			\
	TBZ	$0, R20, 3(PC);			\
	BIC	$1, R20, R26;			\
	MOVD	(R26), R20;			\
	PCDATA	$PCDATA_StackMapIndex, $0;	\
	BL	(R20);				\
