package cbytecache

type Listener interface {
	Listen(entry Entry) error
}
