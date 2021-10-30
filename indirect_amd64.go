package cbytecache

// Indirect arenaID from raw pointer.
//
// Fix "checkptr: pointer arithmetic result points to invalid allocation" error in race mode.
//go:noescape
func indirectArenaID(_ uintptr) uint32
