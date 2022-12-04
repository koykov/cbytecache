package cbytecache

type DumpWriter interface {
	Write(entry Entry) error
	Flush() error
}

type DumpReader interface {
	Read() (Entry, error)
}
