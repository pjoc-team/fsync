package fsync

// SyncServer sync server api
type SyncServer interface {
	Start() error

	// AddPath add file path
	AddPath(path string) error

	// Close close
	Close()
}
