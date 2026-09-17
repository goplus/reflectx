	MOVQ	f+8(FP), DX;			\
	PCDATA  $PCDATA_StackMapIndex, $0;	\
	MOVQ	(DX), R12;			\
	CALL	R12;				\
