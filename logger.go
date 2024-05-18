package cbytecache

// Logger is the interface that wraps the basic logging methods.
type Logger interface {
	Printf(format string, v ...any)
	Print(v ...any)
	Println(v ...any)
}
