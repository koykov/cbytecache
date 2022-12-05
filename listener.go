package cbytecache

// Listener is the interface that wraps the basic Listen method.
type Listener interface {
	Listen(entry Entry) error
}
