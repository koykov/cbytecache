TEXT    ·indirectArenaID(SB), $0-12
MOVQ    addr+8(SP), AX
PCDATA  $0, $-1
MOVL    (AX), AX
MOVL    AX, r1+16(SP)
RET
