package cbytecache

// DumpWriter is the interface that wraps the basic Write method.
type DumpWriter interface {
	// Write writes entry data to the underlying data stream.
	// It returns the number of bytes written from entry and any error encountered.
	Write(entry Entry) (int, error)
	// Flush indicates to writer that all cache data was dumped.
	Flush() error
}

// DumpReader is the interface that wraps the basic Read method.
type DumpReader interface {
	// Read reads entry from underlying data stream.
	// It returns entry and any error encountered.
	Read() (Entry, error)
}
