package fsync

//go:generate go run github.com/launchdarkly/go-options  -type foptions
type foptions struct {
	BufferSize int
}
