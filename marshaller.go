package cbytecache

// MarshallerTo interface to write struct like Protobuf.
type MarshallerTo interface {
	Size() int
	MarshalTo([]byte) (int, error)
}

// Unmarshaller is the interface that wraps the basic Unmarshal method.
type Unmarshaller interface {
	Unmarshal([]byte) error
}
