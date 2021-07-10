package cbytecache

// Compact entry type.
//
// One uint64 value represents 3 values:
// * offset in shard data
// * length of entry bytes
// * expire time
// like this:
// |--------64 --------48 --------40 --------32 --------24 --------16 --------08 --------00|
//  ^                 ^   ^                 ^   ^                                       ^
//        expire                length                           offset
type entry uint64
