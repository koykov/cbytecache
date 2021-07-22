package cbytecache

// MarshallerTo interface to write struct like Protobuf.
type MarshallerTo interface {
	Size() int
	MarshalTo([]byte) (int, error)
}
