package fsync

//go:generate go run github.com/launchdarkly/go-options  -type foptions
type foptions struct {
	BufferSize int
	InitUpload bool
	ConfPath   string
	ThreadPoolSize int
}
