package cbytecache

type Listener interface {
	Listen(key string, body []byte)
}
